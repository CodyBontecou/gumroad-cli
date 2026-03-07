package output

import (
	"bytes"
	"os"
	"strings"
	"testing"
)

func TestStyler_MethodsEnabled(t *testing.T) {
	style := Styler{enabled: true}

	cases := []struct {
		got  string
		want string
	}{
		{style.Bold("hello"), "\033[1m"},
		{style.Red("error"), "\033[31m"},
		{style.Green("ok"), "\033[32m"},
		{style.Yellow("warn"), "\033[33m"},
		{style.Dim("faded"), "\033[2m"},
	}

	for _, tc := range cases {
		if !strings.Contains(tc.got, tc.want) {
			t.Fatalf("expected %q in %q", tc.want, tc.got)
		}
	}
}

func TestColorWrappersUseGlobalOverride(t *testing.T) {
	resetColorEnabledForTest(t)
	SetColorEnabledForTesting(true)
	for _, got := range []string{
		Bold("hello"),
		Red("error"),
		Green("ok"),
		Yellow("warn"),
		Dim("faded"),
	} {
		if !strings.Contains(got, "\033[") {
			t.Fatalf("expected ANSI output, got %q", got)
		}
	}
}

func TestStyler_MethodsDisabled(t *testing.T) {
	style := Styler{enabled: false}

	for _, got := range []string{
		style.Bold("hello"),
		style.Red("error"),
		style.Green("ok"),
		style.Yellow("warn"),
		style.Dim("faded"),
	} {
		if strings.Contains(got, "\033[") {
			t.Fatalf("expected plain text, got %q", got)
		}
	}
}

func TestIsTTY(t *testing.T) {
	if IsTTY() {
		t.Error("expected IsTTY()=false in test environment")
	}
}

func TestNewStylerForWriter_DisablesColorForNonTTYWriter(t *testing.T) {
	resetColorEnabledForTest(t)

	if NewStylerForWriter(&bytes.Buffer{}, false).Enabled() {
		t.Fatal("expected non-file writer to disable color")
	}
}

func TestNewStyler_HonorsNoColorAndTERM(t *testing.T) {
	resetColorEnabledForTest(t)

	origStdoutTTY := stdoutIsTerminal
	stdoutIsTerminal = func() bool { return true }
	t.Cleanup(func() { stdoutIsTerminal = origStdoutTTY })

	t.Setenv("NO_COLOR", "")
	t.Setenv("TERM", "xterm-256color")
	if !NewStyler(false).Enabled() {
		t.Fatal("expected color in a capable terminal")
	}
	if NewStyler(true).Enabled() {
		t.Fatal("expected --no-color to disable color")
	}

	t.Setenv("TERM", "dumb")
	if NewStyler(false).Enabled() {
		t.Fatal("expected TERM=dumb to disable color")
	}
}

func TestNewStylerForWriter_HonorsColorOverride(t *testing.T) {
	resetColorEnabledForTest(t)

	SetColorEnabledForTesting(true)
	if !NewStylerForWriter(os.Stdout, true).Enabled() {
		t.Fatal("expected explicit override to force color on")
	}

	SetColorEnabledForTesting(false)
	if NewStylerForWriter(os.Stdout, false).Enabled() {
		t.Fatal("expected explicit override to force color off")
	}
}

func TestSetStdoutIsTerminalForTestingAndReset(t *testing.T) {
	orig := stdoutIsTerminal
	t.Cleanup(func() { stdoutIsTerminal = orig })

	SetStdoutIsTerminalForTesting(true)
	if !stdoutIsTerminal() {
		t.Fatal("expected stdout TTY override to be enabled")
	}

	ResetStdoutIsTerminalForTesting()
	if stdoutIsTerminal() {
		t.Fatal("expected reset stdout TTY detection to match the test environment")
	}
}
