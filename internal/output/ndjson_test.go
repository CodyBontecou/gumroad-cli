package output_test

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/antiwork/gumroad-cli/internal/output"
)

func TestPrintNDJSON_OneRecordPerLine(t *testing.T) {
	var buf bytes.Buffer
	err := output.PrintNDJSON(&buf, func(writeItem func(any) error) error {
		if err := writeItem(map[string]any{"id": "a"}); err != nil {
			return err
		}
		if err := writeItem(map[string]any{"id": "b"}); err != nil {
			return err
		}
		return writeItem(map[string]any{"id": "c"})
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	lines := strings.Split(strings.TrimRight(buf.String(), "\n"), "\n")
	if len(lines) != 3 {
		t.Fatalf("got %d lines, want 3:\n%s", len(lines), buf.String())
	}
	for i, want := range []string{"a", "b", "c"} {
		var rec map[string]any
		if err := json.Unmarshal([]byte(lines[i]), &rec); err != nil {
			t.Fatalf("line %d not valid JSON: %v\n%q", i, err, lines[i])
		}
		if rec["id"] != want {
			t.Errorf("line %d id=%v want %s", i, rec["id"], want)
		}
	}
}

func TestPrintNDJSON_NoTopLevelEnvelope(t *testing.T) {
	var buf bytes.Buffer
	if err := output.PrintNDJSON(&buf, func(writeItem func(any) error) error {
		return writeItem(map[string]any{"id": "x"})
	}); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(buf.String(), `"items"`) {
		t.Errorf("expected no envelope, got %q", buf.String())
	}
	if strings.HasPrefix(buf.String(), "[") {
		t.Errorf("expected no array envelope, got %q", buf.String())
	}
}

func TestPrintNDJSON_EmptyEmitsNoOutput(t *testing.T) {
	var buf bytes.Buffer
	if err := output.PrintNDJSON(&buf, func(writeItem func(any) error) error {
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	if buf.Len() != 0 {
		t.Errorf("expected empty output, got %q", buf.String())
	}
}
