package output

import (
	"errors"
	"fmt"
	"io"
	"testing"
)

func TestIsBrokenPipeError(t *testing.T) {
	if !IsBrokenPipeError(io.ErrClosedPipe) {
		t.Fatal("expected io.ErrClosedPipe to be detected")
	}
	if !IsBrokenPipeError(fmt.Errorf("wrapped: %w", io.ErrClosedPipe)) {
		t.Fatal("expected wrapped io.ErrClosedPipe to be detected")
	}
	if IsBrokenPipeError(errors.New("boom")) {
		t.Fatal("did not expect unrelated error to be detected as broken pipe")
	}
}
