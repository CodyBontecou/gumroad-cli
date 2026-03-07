package output

import (
	"bytes"
	"errors"
	"strings"
	"testing"
)

func TestPrintPlain(t *testing.T) {
	var buf bytes.Buffer
	rows := [][]string{
		{"a", "b", "c"},
		{"d", "e", "f"},
	}
	if err := PrintPlain(&buf, rows); err != nil {
		t.Fatalf("PrintPlain failed: %v", err)
	}
	expected := "a\tb\tc\nd\te\tf\n"
	if buf.String() != expected {
		t.Errorf("got %q, want %q", buf.String(), expected)
	}
}

func TestPrintPlain_Empty(t *testing.T) {
	var buf bytes.Buffer
	if err := PrintPlain(&buf, nil); err != nil {
		t.Fatalf("PrintPlain failed: %v", err)
	}
	if buf.Len() != 0 {
		t.Errorf("expected empty output, got %q", buf.String())
	}
}

func TestPrintPlain_EscapesControlCharacters(t *testing.T) {
	var buf bytes.Buffer
	rows := [][]string{
		{"a\tb", "line1\nline2", "c\\d", "carriage\rreturn", "escape\x1b", "nul\x00bell\a", "c1\u0085"},
	}
	if err := PrintPlain(&buf, rows); err != nil {
		t.Fatalf("PrintPlain failed: %v", err)
	}
	expected := "a\\tb\tline1\\nline2\tc\\\\d\tcarriage\\rreturn\tescape\\x1b\tnul\\x00bell\\x07\tc1\\u0085\n"
	if buf.String() != expected {
		t.Errorf("got %q, want %q", buf.String(), expected)
	}
}

func TestPrintPlain_WriteError(t *testing.T) {
	err := PrintPlain(errWriter{}, [][]string{{"a"}})
	if !errors.Is(err, errTestWrite) {
		t.Fatalf("got %v, want %v", err, errTestWrite)
	}
}

func TestWriteUnicodeEscape_SupplementaryRune(t *testing.T) {
	var b strings.Builder
	writeUnicodeEscape(&b, rune(0x1D173))

	if got := b.String(); got != `\U0001d173` {
		t.Fatalf("got %q, want %q", got, `\U0001d173`)
	}
}
