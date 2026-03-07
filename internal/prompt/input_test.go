package prompt

import (
	"bytes"
	"errors"
	"os"
	"strings"
	"testing"
)

type errReader struct {
	err error
}

func (r errReader) Read([]byte) (int, error) {
	return 0, r.err
}

func pipeReader(t *testing.T, input string) *os.File {
	t.Helper()

	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("Pipe failed: %v", err)
	}
	t.Cleanup(func() { _ = r.Close() })

	go func() {
		if _, err := w.Write([]byte(input)); err != nil {
			panic(err)
		}
		_ = w.Close()
	}()

	return r
}

func captureProcessStderr(t *testing.T) func() string {
	t.Helper()

	oldStderr := os.Stderr
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("Pipe failed: %v", err)
	}
	os.Stderr = w

	return func() string {
		t.Helper()
		_ = w.Close()
		os.Stderr = oldStderr

		var buf bytes.Buffer
		if _, err := buf.ReadFrom(r); err != nil {
			t.Fatalf("ReadFrom failed: %v", err)
		}
		_ = r.Close()
		return buf.String()
	}
}

func TestTokenInput_FromPipe(t *testing.T) {
	token, err := TokenInput(pipeReader(t, "my-secret-token\n"), &bytes.Buffer{}, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if token != "my-secret-token" {
		t.Errorf("got token=%q, want my-secret-token", token)
	}
}

func TestTokenInput_TrimWhitespace(t *testing.T) {
	token, err := TokenInput(pipeReader(t, "  token-with-spaces  \n"), &bytes.Buffer{}, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if token != "token-with-spaces" {
		t.Errorf("got token=%q, want trimmed", token)
	}
}

func TestTokenInput_EmptyPipe(t *testing.T) {
	token, err := TokenInput(pipeReader(t, ""), &bytes.Buffer{}, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if token != "" {
		t.Errorf("got token=%q, want empty token", token)
	}
}

func TestTokenInput_FromPipe_ReadsToEOF(t *testing.T) {
	want := strings.Repeat("a", 5000)
	token, err := TokenInput(pipeReader(t, want+"\n"), &bytes.Buffer{}, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if token != want {
		t.Errorf("got token length=%d, want %d", len(token), len(want))
	}
}

func TestTokenInput_FromPipe_RejectsOversizedInput(t *testing.T) {
	_, err := TokenInput(pipeReader(t, strings.Repeat("a", maxTokenInputBytes+1)), &bytes.Buffer{}, false)
	if err == nil {
		t.Fatal("expected oversized stdin token to fail")
	}
	if !strings.Contains(err.Error(), "too large") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestTokenInput_DefaultsToProcessStdin(t *testing.T) {
	oldStdin := os.Stdin
	os.Stdin = pipeReader(t, "default-token\n")
	defer func() { os.Stdin = oldStdin }()

	token, err := TokenInput(nil, &bytes.Buffer{}, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if token != "default-token" {
		t.Fatalf("got token=%q, want default-token", token)
	}
}

func TestTokenInput_NoInput_ReturnsError(t *testing.T) {
	orig := isTerminal
	isTerminal = func(int) bool { return true }
	defer func() { isTerminal = orig }()

	_, err := TokenInput(os.Stdin, &bytes.Buffer{}, true)
	if err == nil {
		t.Fatal("expected error when noInput is set")
	}
	if !strings.Contains(err.Error(), "--no-input") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestTokenInput_Interactive_Success(t *testing.T) {
	origTerm := isTerminal
	isTerminal = func(int) bool { return true }
	defer func() { isTerminal = origTerm }()

	origRead := readPassword
	readPassword = func(fd int) ([]byte, error) {
		return []byte("  interactive-token  "), nil
	}
	defer func() { readPassword = origRead }()

	var out bytes.Buffer
	token, err := TokenInput(os.Stdin, &out, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if token != "interactive-token" {
		t.Errorf("got token=%q, want interactive-token", token)
	}
	if !strings.Contains(out.String(), "Enter your Gumroad API token: ") {
		t.Fatalf("expected prompt in output, got %q", out.String())
	}
}

func TestTokenInput_Interactive_DefaultsToProcessStderr(t *testing.T) {
	origTerm := isTerminal
	isTerminal = func(int) bool { return true }
	defer func() { isTerminal = origTerm }()

	origRead := readPassword
	readPassword = func(fd int) ([]byte, error) {
		return []byte("interactive-default"), nil
	}
	defer func() { readPassword = origRead }()

	finish := captureProcessStderr(t)
	token, err := TokenInput(os.Stdin, nil, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if token != "interactive-default" {
		t.Fatalf("got token=%q, want interactive-default", token)
	}
	if got := finish(); !strings.Contains(got, "Enter your Gumroad API token: ") {
		t.Fatalf("expected prompt in stderr, got %q", got)
	}
}

func TestTokenInput_Interactive_Error(t *testing.T) {
	origTerm := isTerminal
	isTerminal = func(int) bool { return true }
	defer func() { isTerminal = origTerm }()

	origRead := readPassword
	readPassword = func(fd int) ([]byte, error) {
		return nil, os.ErrClosed
	}
	defer func() { readPassword = origRead }()

	_, err := TokenInput(os.Stdin, &bytes.Buffer{}, false)
	if err == nil {
		t.Fatal("expected error from readPassword failure")
	}
	if !strings.Contains(err.Error(), "could not read token") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestSecretInput_ReadError(t *testing.T) {
	want := errors.New("boom")

	_, err := SecretInput("prompt", "token", errReader{err: want}, &bytes.Buffer{}, false, "hint")
	if !errors.Is(err, want) {
		t.Fatalf("got %v, want wrapped %v", err, want)
	}
	if !strings.Contains(err.Error(), "could not read token from stdin") {
		t.Fatalf("unexpected error: %v", err)
	}
}
