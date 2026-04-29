package cmdutil_test

import (
	"strings"
	"testing"

	"github.com/antiwork/gumroad-cli/internal/cmdutil"
	"github.com/spf13/cobra"
)

func TestParseJSONBody_FlatStringObject(t *testing.T) {
	v, err := cmdutil.ParseJSONBody(`{"name":"Big","description":"A pack"}`, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if v.Get("name") != "Big" {
		t.Errorf("name=%q", v.Get("name"))
	}
	if v.Get("description") != "A pack" {
		t.Errorf("description=%q", v.Get("description"))
	}
}

func TestParseJSONBody_NumbersAndBools(t *testing.T) {
	v, err := cmdutil.ParseJSONBody(`{"price":1000,"published":true,"discount":0.5}`, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if v.Get("price") != "1000" {
		t.Errorf("price=%q want 1000", v.Get("price"))
	}
	if v.Get("published") != "true" {
		t.Errorf("published=%q want true", v.Get("published"))
	}
	if v.Get("discount") != "0.5" {
		t.Errorf("discount=%q want 0.5", v.Get("discount"))
	}
}

func TestParseJSONBody_ArraysUseBracketSuffix(t *testing.T) {
	v, err := cmdutil.ParseJSONBody(`{"tags":["art","digital","craft"]}`, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got := v["tags[]"]
	want := []string{"art", "digital", "craft"}
	if len(got) != len(want) {
		t.Fatalf("tags[]=%v want %v", got, want)
	}
	for i := range got {
		if got[i] != want[i] {
			t.Errorf("tags[%d]=%q want %q", i, got[i], want[i])
		}
	}
}

func TestParseJSONBody_NullSkipsField(t *testing.T) {
	v, err := cmdutil.ParseJSONBody(`{"name":"Big","description":null}`, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := v["description"]; ok {
		t.Errorf("expected description to be omitted when null, got %q", v.Get("description"))
	}
	if v.Get("name") != "Big" {
		t.Errorf("name lost")
	}
}

func TestParseJSONBody_RejectsNonObject(t *testing.T) {
	_, err := cmdutil.ParseJSONBody(`["not","an","object"]`, nil)
	if err == nil {
		t.Fatal("expected error for top-level array")
	}
}

func TestParseJSONBody_RejectsNestedObject(t *testing.T) {
	_, err := cmdutil.ParseJSONBody(`{"meta":{"key":"value"}}`, nil)
	if err == nil {
		t.Fatal("expected error for nested object")
	}
}

func TestParseJSONBody_RejectsInvalidJSON(t *testing.T) {
	_, err := cmdutil.ParseJSONBody(`{not json`, nil)
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestParseJSONBody_ReadsFromStdinDash(t *testing.T) {
	stdin := strings.NewReader(`{"name":"FromStdin"}`)
	v, err := cmdutil.ParseJSONBody("-", stdin)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if v.Get("name") != "FromStdin" {
		t.Errorf("name=%q", v.Get("name"))
	}
}

func TestParseJSONBody_DashWithoutStdinErrors(t *testing.T) {
	_, err := cmdutil.ParseJSONBody("-", nil)
	if err == nil {
		t.Fatal("expected error when - given but no stdin reader available")
	}
}

func TestParseJSONBody_ArrayWithMixedScalars(t *testing.T) {
	v, err := cmdutil.ParseJSONBody(`{"flags":[true,42,"x"]}`, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got := v["flags[]"]
	want := []string{"true", "42", "x"}
	if len(got) != len(want) {
		t.Fatalf("got %v want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("got[%d]=%q want %q", i, got[i], want[i])
		}
	}
}

func TestParseJSONBody_ArrayWithNullErrors(t *testing.T) {
	if _, err := cmdutil.ParseJSONBody(`{"tags":["a",null]}`, nil); err == nil {
		t.Fatal("expected error for null inside array")
	}
}

func TestParseJSONBody_ArrayWithObjectElementErrors(t *testing.T) {
	if _, err := cmdutil.ParseJSONBody(`{"tags":[{"k":"v"}]}`, nil); err == nil {
		t.Fatal("expected error for object inside array")
	}
}

func TestRejectFlagsWithJSONBody_NoOpWithoutFlag(t *testing.T) {
	cmd := &cobra.Command{Use: "test"}
	cmd.Flags().String("json-body", "", "")
	cmd.Flags().String("name", "", "")
	if err := cmdutil.RejectFlagsWithJSONBody(cmd, "name"); err != nil {
		t.Errorf("unset --json-body should not conflict, got %v", err)
	}
}

func TestRejectFlagsWithJSONBody_RejectsConflict(t *testing.T) {
	cmd := &cobra.Command{Use: "test"}
	cmd.Flags().String("json-body", "", "")
	cmd.Flags().String("name", "", "")
	_ = cmd.Flags().Set("json-body", `{}`)
	_ = cmd.Flags().Set("name", "x")
	err := cmdutil.RejectFlagsWithJSONBody(cmd, "name", "description")
	if err == nil || !strings.Contains(err.Error(), "--name") {
		t.Errorf("expected conflict error mentioning --name, got %v", err)
	}
}

func TestRejectFlagsWithJSONBody_NoConflictListed(t *testing.T) {
	cmd := &cobra.Command{Use: "test"}
	cmd.Flags().String("json-body", "", "")
	cmd.Flags().String("name", "", "")
	_ = cmd.Flags().Set("json-body", `{}`)
	if err := cmdutil.RejectFlagsWithJSONBody(cmd, "name"); err != nil {
		t.Errorf("no conflict expected, got %v", err)
	}
}
