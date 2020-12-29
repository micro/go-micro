package grpc

// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.
//
// Meh, we need to get rid of this shit

import (
	"context"
	"errors"
	"reflect"
	"sync"
	"unicode"
	"unicode/utf8"

	"github.com/micro/go-micro/v2/logger"
	"github.com/micro/go-micro/v2/server"
)

var (
	// Precompute the reflect type for error. Can't use error directly
	// because Typeof takes an empty interface value. This is annoying.
	typeOfError = reflect.TypeOf((*error)(nil)).Elem()
)

type methodType struct {
	method      reflect.Method
	ArgType     reflect.Type
	ReplyType   reflect.Type
	ContextType reflect.Type
	stream      bool
}

type service struct {
	name   string                 // name of service
	rcvr   reflect.Value          // receiver of methods for the service
	typ    reflect.Type           // type of the receiver
	method map[string]*methodType // registered methods
}

// server represents an RPC Server.
type rServer struct {
	mu         sync.Mutex // protects the serviceMap
	serviceMap map[string]*service
}

// Is this an exported - upper case - name?
func isExported(name string) bool {
	rune, _ := utf8.DecodeRuneInString(name)
	return unicode.IsUpper(rune)
}

// Is this type exported or a builtin?
func isExportedOrBuiltinType(t reflect.Type) bool {
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	// PkgPath will be non-empty even for an exported type,
	// so we need to check the type name as well.
	return isExported(t.Name()) || t.PkgPath() == ""
}

// prepareEndpoint() returns a methodType for the provided method or nil
// in case if the method was unsuitable.
func prepareEndpoint(method reflect.Method) *methodType {
	mtype := method.Type
	mname := method.Name
	var replyType, argType, contextType reflect.Type
	var stream bool

	// Endpoint() must be exported.
	if method.PkgPath != "" {
		return nil
	}

	switch mtype.NumIn() {
	case 3:
		// assuming streaming
		argType = mtype.In(2)
		contextType = mtype.In(1)
		stream = true
	case 4:
		// method that takes a context
		argType = mtype.In(2)
		replyType = mtype.In(3)
		contextType = mtype.In(1)
	default:
		if logger.V(logger.ErrorLevel, logger.DefaultLogger) {
			logger.Errorf("method %v of %v has wrong number of ins: %v", mname, mtype, mtype.NumIn())
		}
		return nil
	}

	if stream {
		// check stream type
		streamType := reflect.TypeOf((*server.Stream)(nil)).Elem()
		if !argType.Implements(streamType) {
			if logger.V(logger.ErrorLevel, logger.DefaultLogger) {
				logger.Errorf("%v argument does not implement Streamer interface: %v", mname, argType)
			}
			return nil
		}
	} else {
		// if not stream check the replyType

		// First arg need not be a pointer.
		if !isExportedOrBuiltinType(argType) {
			if logger.V(logger.ErrorLevel, logger.DefaultLogger) {
				logger.Errorf("%v argument type not exported: %v", mname, argType)
			}
			return nil
		}

		if replyType.Kind() != reflect.Ptr {
			if logger.V(logger.ErrorLevel, logger.DefaultLogger) {
				logger.Errorf("method %v reply type not a pointer: %v", mname, replyType)
			}
			return nil
		}

		// Reply type must be exported.
		if !isExportedOrBuiltinType(replyType) {
			if logger.V(logger.ErrorLevel, logger.DefaultLogger) {
				logger.Errorf("method %v reply type not exported: %v", mname, replyType)
			}
			return nil
		}
	}

	// Endpoint() needs one out.
	if mtype.NumOut() != 1 {
		if logger.V(logger.ErrorLevel, logger.DefaultLogger) {
			logger.Errorf("method %v has wrong number of outs: %v", mname, mtype.NumOut())
		}
		return nil
	}
	// The return type of the method must be error.
	if returnType := mtype.Out(0); returnType != typeOfError {
		if logger.V(logger.ErrorLevel, logger.DefaultLogger) {
			logger.Errorf("method %v returns %v not error", mname, returnType.String())
		}
		return nil
	}
	return &methodType{method: method, ArgType: argType, ReplyType: replyType, ContextType: contextType, stream: stream}
}

func (server *rServer) register(rcvr interface{}) error {
	server.mu.Lock()
	defer server.mu.Unlock()
	if server.serviceMap == nil {
		server.serviceMap = make(map[string]*service)
	}
	s := new(service)
	s.typ = reflect.TypeOf(rcvr)
	s.rcvr = reflect.ValueOf(rcvr)
	sname := reflect.Indirect(s.rcvr).Type().Name()
	if sname == "" {
		logger.Fatalf("rpc: no service name for type %v", s.typ.String())
	}
	if !isExported(sname) {
		s := "rpc Register: type " + sname + " is not exported"
		if logger.V(logger.ErrorLevel, logger.DefaultLogger) {
			logger.Error(s)
		}
		return errors.New(s)
	}
	if _, present := server.serviceMap[sname]; present {
		return errors.New("rpc: service already defined: " + sname)
	}
	s.name = sname
	s.method = make(map[string]*methodType)

	// Install the methods
	for m := 0; m < s.typ.NumMethod(); m++ {
		method := s.typ.Method(m)
		if mt := prepareEndpoint(method); mt != nil {
			s.method[method.Name] = mt
		}
	}

	if len(s.method) == 0 {
		s := "rpc Register: type " + sname + " has no exported methods of suitable type"
		if logger.V(logger.ErrorLevel, logger.DefaultLogger) {
			logger.Error(s)
		}
		return errors.New(s)
	}
	server.serviceMap[s.name] = s
	return nil
}

func (m *methodType) prepareContext(ctx context.Context) reflect.Value {
	if contextv := reflect.ValueOf(ctx); contextv.IsValid() {
		return contextv
	}
	return reflect.Zero(m.ContextType)
}
