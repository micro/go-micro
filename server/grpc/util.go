package grpc

import (
	"context"
	"io"
	"os"
	"sync"

	"google.golang.org/grpc/codes"
)

// convertCode converts a standard Go error into its canonical code. Note that
// this is only used to translate the error returned by the server applications.
func convertCode(err error) codes.Code {
	switch err {
	case nil:
		return codes.OK
	case io.EOF:
		return codes.OutOfRange
	case io.ErrClosedPipe, io.ErrNoProgress, io.ErrShortBuffer, io.ErrShortWrite, io.ErrUnexpectedEOF:
		return codes.FailedPrecondition
	case os.ErrInvalid:
		return codes.InvalidArgument
	case context.Canceled:
		return codes.Canceled
	case context.DeadlineExceeded:
		return codes.DeadlineExceeded
	}
	switch {
	case os.IsExist(err):
		return codes.AlreadyExists
	case os.IsNotExist(err):
		return codes.NotFound
	case os.IsPermission(err):
		return codes.PermissionDenied
	}
	return codes.Unknown
}

func wait(ctx context.Context) *sync.WaitGroup {
	if ctx == nil {
		return nil
	}
	wg, ok := ctx.Value("wait").(*sync.WaitGroup)
	if !ok {
		return nil
	}
	return wg
}
