package output

import (
	"errors"
	"io"
	"syscall"
)

// IsBrokenPipeError reports whether err represents a closed downstream pipe.
func IsBrokenPipeError(err error) bool {
	return errors.Is(err, io.ErrClosedPipe) || errors.Is(err, syscall.EPIPE)
}
