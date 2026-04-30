package output

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"testing"
)

func TestWriteln(t *testing.T) {
	var buf bytes.Buffer
	if err := Writeln(&buf, "hello", "world"); err != nil {
		t.Fatalf("Writeln failed: %v", err)
	}
	if buf.String() != "hello world\n" {
		t.Fatalf("got %q, want %q", buf.String(), "hello world\n")
	}
}

func TestWritef(t *testing.T) {
	var buf bytes.Buffer
	if err := Writef(&buf, "%s %d", "value", 7); err != nil {
		t.Fatalf("Writef failed: %v", err)
	}
	if buf.String() != "value 7" {
		t.Fatalf("got %q, want %q", buf.String(), "value 7")
	}
}

func TestWritelnAndWritef_PropagateErrors(t *testing.T) {
	if err := Writeln(errWriter{}, "hello"); !errors.Is(err, errTestWrite) {
		t.Fatalf("Writeln got %v, want %v", err, errTestWrite)
	}
	if err := Writef(errWriter{}, "%s", "hello"); !errors.Is(err, errTestWrite) {
		t.Fatalf("Writef got %v, want %v", err, errTestWrite)
	}
}

func TestWithPager_PassthroughWriter(t *testing.T) {
	var buf bytes.Buffer
	if err := WithPager(&buf, &buf, func(w io.Writer) error {
		_, err := w.Write([]byte("paged"))
		return err
	}); err != nil {
		t.Fatalf("WithPager failed: %v", err)
	}
	if buf.String() != "paged" {
		t.Fatalf("got %q, want %q", buf.String(), "paged")
	}
}

func TestWithPager_PropagatesCallbackError(t *testing.T) {
	want := errors.New("boom")
	var buf bytes.Buffer
	if err := WithPager(&buf, &buf, func(io.Writer) error { return want }); !errors.Is(err, want) {
		t.Fatalf("got %v, want %v", err, want)
	}
}

func TestNewStylerForWriter_DisablesColorForNonTTYWriterInWriteTests(t *testing.T) {
	resetColorEnabledForTest(t)
	t.Setenv("NO_COLOR", "")

	if NewStylerForWriter(&bytes.Buffer{}, false).Enabled() {
		t.Fatal("expected color to be disabled for non-TTY writer")
	}
}

func TestIsTerminalWriter_FileBackedNonTTY(t *testing.T) {
	file, err := os.CreateTemp(t.TempDir(), "writer-*")
	if err != nil {
		t.Fatalf("CreateTemp failed: %v", err)
	}
	defer file.Close()

	if isTerminalWriter(file) {
		t.Fatal("expected temp file to not be treated as a terminal")
	}
}

func TestDefaultPagerCommand(t *testing.T) {
	t.Setenv("PAGER", "")
	if got := defaultPagerCommand(); got != "" {
		t.Fatalf("got %q, want empty string when PAGER is explicitly empty", got)
	}

	os.Unsetenv("PAGER")
	if got := defaultPagerCommand(); got != "less -FIRX" {
		t.Fatalf("got %q, want %q", got, "less -FIRX")
	}
}

func TestPagerCommand_Empty(t *testing.T) {
	if cmd := pagerCommand(""); cmd != nil {
		t.Fatalf("expected nil command, got %#v", cmd)
	}
}

func TestPagerCommand_Whitespace(t *testing.T) {
	if cmd := pagerCommand("   "); cmd != nil {
		t.Fatalf("expected nil command for whitespace-only pager, got %#v", cmd)
	}
}

func TestNewSpinnerTo_NilWriterUsesStderr(t *testing.T) {
	s := NewSpinnerTo("loading", nil)
	if s.writer != os.Stderr {
		t.Fatalf("got writer=%v, want os.Stderr", s.writer)
	}
}

func TestTerminalWidthFor_NonFileWriterFallsBack(t *testing.T) {
	if got := TerminalWidthFor(&bytes.Buffer{}, 77); got != 77 {
		t.Fatalf("got %d, want 77", got)
	}
}

func TestFetchAndDecode_InvalidURL(t *testing.T) {
	if _, err := fetchAndDecode(context.Background(), "://bad-url"); err == nil {
		t.Fatal("expected error for invalid URL")
	}
}

func TestTable_RenderShortRow(t *testing.T) {
	var buf bytes.Buffer
	tbl := NewTable("A", "B")
	tbl.AddRow("value")

	if err := tbl.Render(&buf); err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	lines := strings.Split(strings.TrimRight(buf.String(), "\n"), "\n")
	if len(lines) != 3 {
		t.Fatalf("expected 3 lines (header + separator + row), got %d: %q", len(lines), buf.String())
	}
	if !strings.Contains(lines[2], "value") {
		t.Fatalf("expected row to include first cell, got %q", lines[2])
	}
}

func TestWithPager_PropagatesPagerExitError(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("pager exit status test is POSIX-specific")
	}

	origTTY := isStdoutTerminal
	isStdoutTerminal = func() bool { return true }
	defer func() { isStdoutTerminal = origTTY }()

	t.Setenv("PAGER", "sh -c 'cat >/dev/null; exit 42'")
	t.Setenv("TERM", "xterm-256color")

	err := WithPager(os.Stdout, os.Stderr, func(w io.Writer) error {
		_, err := fmt.Fprintln(w, "hello")
		return err
	})

	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) {
		t.Fatalf("got %v, want pager exit error", err)
	}
}
