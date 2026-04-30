package output

import (
	"bytes"
	"io"
	"strings"
	"testing"
	"time"
)

func TestProgressBar_RendersOnFakeTTY(t *testing.T) {
	orig := isProgressTerminal
	isProgressTerminal = func(io.Writer) bool { return true }
	defer func() { isProgressTerminal = orig }()
	t.Setenv("TERM", "xterm-256color")

	buf := &concurrentBuffer{}
	p := NewProgressBar(buf, "uploading", 100)
	p.Set(25)
	p.Add(25)
	p.SetLabel("almost there")
	p.Done("uploaded 50 of 100")

	out := buf.String()
	if !strings.Contains(out, "uploaded 50 of 100") {
		t.Fatalf("expected final message: %q", out)
	}
	if !strings.Contains(out, "almost there") {
		t.Fatalf("expected updated label to render: %q", out)
	}
}

func TestProgressBar_IndeterminateOnFakeTTY(t *testing.T) {
	orig := isProgressTerminal
	isProgressTerminal = func(io.Writer) bool { return true }
	defer func() { isProgressTerminal = orig }()
	t.Setenv("TERM", "xterm-256color")

	buf := &concurrentBuffer{}
	p := NewProgressBar(buf, "fetching", 0)
	p.Set(0)
	p.Fail("network error")

	if !strings.Contains(buf.String(), "fetching") {
		t.Fatalf("expected indeterminate label: %q", buf.String())
	}
	if !strings.Contains(buf.String(), "network error") {
		t.Fatalf("expected failure message: %q", buf.String())
	}
}

func TestProgressBar_DoneSilentOnDumbTerminal(t *testing.T) {
	orig := isProgressTerminal
	isProgressTerminal = func(io.Writer) bool { return true }
	defer func() { isProgressTerminal = orig }()
	t.Setenv("TERM", "dumb")

	buf := &concurrentBuffer{}
	NewProgressBar(buf, "x", 100).Done("complete")
	if !strings.Contains(buf.String(), "complete") {
		t.Fatalf("expected fallback final message under TERM=dumb: %q", buf.String())
	}
}

func TestProgressBar_NonTTYNoOp(t *testing.T) {
	var buf bytes.Buffer
	p := NewProgressBar(&buf, "uploading", 100)
	p.Set(50)
	p.Add(10)
	p.SetLabel("almost there")
	if buf.Len() != 0 {
		t.Fatalf("expected no rendering on non-TTY, got %q", buf.String())
	}
}

func TestProgressBar_DoneAndFailNonTTY(t *testing.T) {
	var done bytes.Buffer
	NewProgressBar(&done, "uploading", 100).Done("uploaded")
	if !strings.Contains(done.String(), "uploaded") {
		t.Fatalf("expected fallback final message: %q", done.String())
	}

	var fail bytes.Buffer
	NewProgressBar(&fail, "uploading", 100).Fail("bad")
	if !strings.Contains(fail.String(), "bad") {
		t.Fatalf("expected fallback failure message: %q", fail.String())
	}
}

func TestProgressBar_NilWriterDefaultsToStderr(t *testing.T) {
	p := NewProgressBar(nil, "x", 0)
	if p.writer == nil {
		t.Fatal("expected default writer to be set")
	}
}

func TestRenderBar_DisabledThemeUsesAscii(t *testing.T) {
	got := renderBar(NewTheme(false), 10, 4)
	if got != "[====      ]" {
		t.Fatalf("got %q", got)
	}
}

func TestRenderBar_EnabledThemeEmitsANSI(t *testing.T) {
	got := renderBar(NewTheme(true), 10, 4)
	if !strings.Contains(got, "\x1b[") {
		t.Fatalf("expected ANSI in enabled bar: %q", got)
	}
}

func TestFormatProgressEta(t *testing.T) {
	if got := formatProgressEta(0, 100, time.Second); got != "—" {
		t.Fatalf("zero current: got %q", got)
	}
	if got := formatProgressEta(50, 100, 0); got != "—" {
		t.Fatalf("zero elapsed: got %q", got)
	}
	if got := formatProgressEta(50, 100, 5*time.Second); !strings.Contains(got, "left") {
		t.Fatalf("expected ETA: %q", got)
	}
}

func TestFormatElapsed(t *testing.T) {
	if got := formatElapsed(2 * time.Second); got != "2.0s" {
		t.Fatalf("seconds: %q", got)
	}
	if got := formatElapsed(95 * time.Second); got != "1m35s" {
		t.Fatalf("minutes: %q", got)
	}
}

// Discard sink used by SetMessage concurrent tests already in spinner_test.go,
// re-exported here for documentation that progress also tolerates io.Discard.
var _ io.Writer = io.Discard
