package output

import (
	"fmt"
	"io"
)

func Writeln(w io.Writer, args ...any) error {
	_, err := fmt.Fprintln(w, args...)
	return err
}

func Writef(w io.Writer, format string, args ...any) error {
	_, err := fmt.Fprintf(w, format, args...)
	return err
}

func WithPager(stdout, stderr io.Writer, fn func(io.Writer) error) error {
	p := NewPagerTo(stdout, stderr)
	runErr := fn(p)
	closeErr := p.Close()
	if IsBrokenPipeError(runErr) {
		runErr = nil
	}
	if IsBrokenPipeError(closeErr) {
		closeErr = nil
	}
	if runErr != nil {
		return runErr
	}

	return closeErr
}
