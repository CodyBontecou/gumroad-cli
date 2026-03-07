package output

import (
	"io"
	"testing"
	"time"
)

func TestSpinner_NonTTY_DoesNotActivate(t *testing.T) {
	orig := isSpinnerTerminal
	isSpinnerTerminal = func(io.Writer) bool { return false }
	defer func() { isSpinnerTerminal = orig }()

	s := NewSpinner("loading")
	s.Start()
	defer s.Stop()

	s.mu.Lock()
	active := s.active
	s.mu.Unlock()

	if active {
		t.Error("spinner should not be active on non-TTY")
	}
}

func TestSpinner_TTY_ActivatesAndStops(t *testing.T) {
	orig := isSpinnerTerminal
	isSpinnerTerminal = func(io.Writer) bool { return true }
	defer func() { isSpinnerTerminal = orig }()

	s := NewSpinner("loading")
	s.Start()

	// Give goroutine time to start
	time.Sleep(10 * time.Millisecond)

	s.mu.Lock()
	active := s.active
	s.mu.Unlock()
	if !active {
		t.Error("spinner should be active on TTY after Start")
	}

	s.Stop()

	s.mu.Lock()
	active = s.active
	s.mu.Unlock()
	if active {
		t.Error("spinner should not be active after Stop")
	}
}

func TestSpinner_DoubleStop(t *testing.T) {
	orig := isSpinnerTerminal
	isSpinnerTerminal = func(io.Writer) bool { return true }
	defer func() { isSpinnerTerminal = orig }()

	s := NewSpinner("loading")
	s.Start()
	time.Sleep(10 * time.Millisecond)

	s.Stop()
	// Second stop should not panic
	s.Stop()
}

func TestSpinner_StopWithoutStart(t *testing.T) {
	s := NewSpinner("loading")
	// Should not panic
	s.Stop()
}

func TestSpinner_TermDumb_DoesNotActivate(t *testing.T) {
	orig := isSpinnerTerminal
	isSpinnerTerminal = func(io.Writer) bool { return true }
	defer func() { isSpinnerTerminal = orig }()

	t.Setenv("TERM", "dumb")

	s := NewSpinner("loading")
	s.Start()
	defer s.Stop()

	s.mu.Lock()
	active := s.active
	s.mu.Unlock()

	if active {
		t.Error("spinner should not be active when TERM=dumb")
	}
}

func TestSpinner_TermDumb_UsesCachedValue(t *testing.T) {
	orig := isSpinnerTerminal
	isSpinnerTerminal = func(io.Writer) bool { return true }
	defer func() { isSpinnerTerminal = orig }()

	t.Setenv("TERM", "dumb")

	s := NewSpinner("test")
	s.Start()
	defer s.Stop()

	s.mu.Lock()
	active := s.active
	s.mu.Unlock()

	if active {
		t.Error("spinner activated despite TERM=dumb")
	}
}

func TestNewSpinner(t *testing.T) {
	s := NewSpinner("test message")
	if s.message != "test message" {
		t.Errorf("got message=%q, want %q", s.message, "test message")
	}
	if s.done == nil {
		t.Error("done channel should not be nil")
	}
	if s.active {
		t.Error("spinner should not be active by default")
	}
}
