package server

import (
	"fmt"
	"sync"

	"go-micro.dev/v5/codec"
	"go-micro.dev/v5/registry"
)

// setRegistered will set the service as registered safely.
func (s *rpcServer) setRegistered(b bool) {
	s.Lock()
	defer s.Unlock()

	s.registered = b
}

// isRegistered will check if the service has already been registered.
func (s *rpcServer) isRegistered() bool {
	s.RLock()
	defer s.RUnlock()

	return s.registered
}

// setStarted will set started state safely.
func (s *rpcServer) setStarted(b bool) {
	s.Lock()
	defer s.Unlock()

	s.started = b
}

// isStarted will check if the service has already been started.
func (s *rpcServer) isStarted() bool {
	s.RLock()
	defer s.RUnlock()

	return s.started
}

// setWg will set the waitgroup safely.
func (s *rpcServer) setWg(wg *sync.WaitGroup) {
	s.Lock()
	defer s.Unlock()

	s.wg = wg
}

// getWaitgroup returns the global waitgroup safely.
func (s *rpcServer) getWg() *sync.WaitGroup {
	s.RLock()
	defer s.RUnlock()

	return s.wg
}

// setOptsAddr will set the address in the service options safely.
func (s *rpcServer) setOptsAddr(addr string) {
	s.Lock()
	defer s.Unlock()

	s.opts.Address = addr
}

func (s *rpcServer) getCachedService() *registry.Service {
	s.RLock()
	defer s.RUnlock()

	return s.rsvc
}

func (s *rpcServer) Options() Options {
	s.RLock()
	defer s.RUnlock()

	return s.opts
}

// swapAddr swaps the address found in the config and the transport address.
func (s *rpcServer) swapAddr(config Options, addr string) string {
	s.Lock()
	defer s.Unlock()

	a := config.Address
	s.opts.Address = addr
	return a
}

func (s *rpcServer) newCodec(contentType string) (codec.NewCodec, error) {
	if cf, ok := s.opts.Codecs[contentType]; ok {
		return cf, nil
	}

	if cf, ok := DefaultCodecs[contentType]; ok {
		return cf, nil
	}

	return nil, fmt.Errorf("unsupported Content-Type: %s", contentType)
}
