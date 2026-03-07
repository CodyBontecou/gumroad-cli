package output

import "errors"

var errTestWrite = errors.New("write failed")

type errWriter struct{}

func (errWriter) Write([]byte) (int, error) {
	return 0, errTestWrite
}
