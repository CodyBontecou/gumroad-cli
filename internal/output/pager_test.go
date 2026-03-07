package output

import (
	"fmt"
	"io"
	"os"
	"runtime"
	"testing"
)

func TestPager_NonTTY_WritesToStdout(t *testing.T) {
	// In test environment, stdout is not a TTY
	// so NewPager should fall back to os.Stdout
	p := NewPager()
	if p.cmd != nil {
		t.Error("pager should not spawn a process when not a TTY")
		p.Close()
	}
	if p.writer != os.Stdout {
		t.Error("pager should write to os.Stdout when not a TTY")
	}
	if err := p.Close(); err != nil {
		t.Errorf("Close() returned error: %v", err)
	}
}

func TestPager_NilWriterFallsBackToStdout(t *testing.T) {
	p := NewPagerTo(nil, nil)
	if p.cmd != nil {
		t.Error("pager should not spawn a process when writer is nil in tests")
		p.Close()
	}
	if p.writer != os.Stdout {
		t.Error("pager should fall back to os.Stdout when writer is nil")
	}
	if err := p.Close(); err != nil {
		t.Errorf("Close() returned error: %v", err)
	}
}

func TestPager_TermDumb_WritesToStdout(t *testing.T) {
	origTTY := isStdoutTerminal
	isStdoutTerminal = func() bool { return true }
	defer func() { isStdoutTerminal = origTTY }()

	t.Setenv("TERM", "dumb")

	p := NewPager()
	if p.cmd != nil {
		t.Error("pager should not spawn a process when TERM=dumb")
		p.Close()
	}
	if err := p.Close(); err != nil {
		t.Errorf("Close() returned error: %v", err)
	}
}

func TestPager_CustomPager(t *testing.T) {
	origTTY := isStdoutTerminal
	isStdoutTerminal = func() bool { return true }
	defer func() { isStdoutTerminal = origTTY }()

	t.Setenv("PAGER", "cat")
	t.Setenv("TERM", "xterm-256color")

	p := NewPager()
	if p.cmd == nil {
		t.Fatal("pager should spawn a process with TTY and valid PAGER")
	}

	_, err := p.Write([]byte("hello\n"))
	if err != nil {
		t.Errorf("Write() returned error: %v", err)
	}

	if err := p.Close(); err != nil {
		t.Errorf("Close() returned error: %v", err)
	}
}

func TestPager_WhitespacePager_FallsBack(t *testing.T) {
	origTTY := isStdoutTerminal
	isStdoutTerminal = func() bool { return true }
	defer func() { isStdoutTerminal = origTTY }()

	t.Setenv("PAGER", "   ")
	t.Setenv("TERM", "xterm-256color")

	p := NewPager()
	if p.cmd != nil {
		t.Error("pager should fall back to stdout when PAGER is whitespace")
		p.Close()
	}
	if err := p.Close(); err != nil {
		t.Errorf("Close() returned error: %v", err)
	}
}

func TestPager_InvalidPager_FallsBack(t *testing.T) {
	origTTY := isStdoutTerminal
	isStdoutTerminal = func() bool { return true }
	defer func() { isStdoutTerminal = origTTY }()

	t.Setenv("PAGER", "/nonexistent/binary")
	t.Setenv("TERM", "xterm-256color")

	p := NewPager()
	if p.cmd != nil {
		t.Error("pager should fall back to stdout when pager binary doesn't exist")
		p.Close()
	}
	if err := p.Close(); err != nil {
		t.Errorf("Close() returned error: %v", err)
	}
}

func TestPager_ShellParsedPager(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell pager test is POSIX-specific")
	}

	origTTY := isStdoutTerminal
	isStdoutTerminal = func() bool { return true }
	defer func() { isStdoutTerminal = origTTY }()

	t.Setenv("PAGER", "sh -c 'cat >/dev/null'")
	t.Setenv("TERM", "xterm-256color")

	p := NewPager()
	if p.cmd == nil {
		t.Fatal("pager should spawn a process for shell-parsed PAGER")
	}

	if _, err := p.Write([]byte("hello\n")); err != nil {
		t.Fatalf("Write() returned error: %v", err)
	}
	if err := p.Close(); err != nil {
		t.Fatalf("Close() returned error: %v", err)
	}
}

func TestWithPager_IgnoresBrokenPipeFromEarlyPagerExit(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("broken pipe pager test is POSIX-specific")
	}

	origTTY := isStdoutTerminal
	isStdoutTerminal = func() bool { return true }
	defer func() { isStdoutTerminal = origTTY }()

	t.Setenv("PAGER", "sh -c 'head -n1 >/dev/null'")
	t.Setenv("TERM", "xterm-256color")

	err := WithPager(os.Stdout, os.Stderr, func(w io.Writer) error {
		for i := 0; i < 4096; i++ {
			if _, err := fmt.Fprintln(w, "hello"); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("WithPager returned error for early pager exit: %v", err)
	}
}
