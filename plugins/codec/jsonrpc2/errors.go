package jsonrpc2

import (
	"encoding/json"
	"fmt"
	"strings"
)

var (
	// Actual returned error may have different message.
	errParse       = NewError(-32700, "Parse error")
	errRequest     = NewError(-32600, "Invalid request")
	errMethod      = NewError(-32601, "Method not found")
	errParams      = NewError(-32602, "Invalid params")
	errInternal    = NewError(-32603, "Internal error")
	errServer      = NewError(-32000, "Server error")
	errServerError = NewError(-32001, "jsonrpc2.Error: json.Marshal failed")
)

// Error represent JSON-RPC 2.0 "Error object".
type Error struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// NewError returns an Error with given code and message.
func NewError(code int, message string) *Error {
	return &Error{Code: code, Message: message}
}

// newError returns an Error with auto-detected code for given message.
func newError(message string) *Error {
	switch {
	case strings.HasPrefix(message, "rpc: service/method request ill-formed"):
		return NewError(errMethod.Code, message)
	case strings.HasPrefix(message, "rpc: can't find service"):
		return NewError(errMethod.Code, message)
	case strings.HasPrefix(message, "rpc: can't find method"):
		return NewError(errMethod.Code, message)
	default:
		return NewError(errServer.Code, message)
	}
}

// ServerError convert errors returned by Client.Call() into Error.
// User should check for rpc.ErrShutdown and io.ErrUnexpectedEOF before
// calling ServerError.
func ServerError(rpcerr error) *Error {
	if rpcerr == nil {
		return nil
	}
	if err, ok := rpcerr.(*Error); ok {
		if err.Code == errInternal.Code && err.Data != nil {
			if err2, ok := err.Data.(*Error); ok {
				// Use alternate error when ReadResponseBody fail on other call.
				return err2
			}
		}
		return err
	}
	keepData := true
	errmsg := rpcerr.Error()
	if s := strings.Index(errmsg, "{"); strings.HasPrefix(errmsg, "reading ") && s != -1 && strings.HasSuffix(errmsg, "}") {
		// ReadResponseBody fail on this call.
		errmsg = errmsg[s:]
		keepData = false
	}
	e := &Error{}
	err := json.Unmarshal([]byte(errmsg), e)
	if err != nil {
		panic(fmt.Sprintf("not a jsonrpc2 error: %s (%#q)", err, rpcerr))
	}
	if e.Code == errInternal.Code && e.Data != nil && !keepData {
		// ReadResponseBody fail on this call.
		e.Data = nil
	}
	return e
}

// Error returns JSON representation of Error.
func (e *Error) Error() string {
	buf, err := json.Marshal(e)
	if err != nil {
		msg, err := json.Marshal(err.Error())
		if err != nil {
			msg = []byte(`"` + errServerError.Message + `"`)
		}
		return fmt.Sprintf(`{"code":%d,"message":%s}`, errServerError.Code, string(msg))
	}
	return string(buf)
}
