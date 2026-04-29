package cmdutil_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/antiwork/gumroad-cli/internal/cmdutil"
	"github.com/spf13/cobra"
)

func TestValidateResourceID_RejectsInjectionChars(t *testing.T) {
	cases := []struct {
		name string
		id   string
	}{
		{"empty", ""},
		{"question mark", "abc?fields=name"},
		{"hash", "abc#fragment"},
		{"percent encoded traversal", "abc%2e%2e"},
		{"bare percent", "abc%def"},
		{"control char", "abc\x01def"},
		{"newline", "abc\ndef"},
		{"carriage return", "abc\rdef"},
		{"tab", "abc\tdef"},
		{"forward slash", "abc/def"},
		{"backslash", `abc\def`},
		{"dot dot", ".."},
		{"single dot", "."},
		{"leading slash", "/abc"},
		{"DEL char", "abc\x7fdef"},
		{"space", "abc def"},
		{"ANSI escape", "abc\x1b[31mdef"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := cmdutil.ValidateResourceID(tc.id)
			if err == nil {
				t.Fatalf("expected error for id=%q", tc.id)
			}
		})
	}
}

func TestValidateResourceID_AcceptsValidIDs(t *testing.T) {
	valid := []string{
		"abc123",
		"cGksPcArAUU8j_XTYsrnQ==",
		"long-id-with-dashes",
		"prod_abc",
		"v1.2",
		"A1B2C3",
	}
	for _, id := range valid {
		t.Run(id, func(t *testing.T) {
			if err := cmdutil.ValidateResourceID(id); err != nil {
				t.Errorf("expected %q to be valid, got %v", id, err)
			}
		})
	}
}

func TestValidateResourceID_ErrorMessageNamesProblem(t *testing.T) {
	err := cmdutil.ValidateResourceID("abc?fields=name")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "?") {
		t.Errorf("expected error to mention the offending char, got %q", err.Error())
	}
}

func TestSafeIDArgs_RejectsInjectedID(t *testing.T) {
	cmd := &cobra.Command{
		Use:  "view <id>",
		Args: cmdutil.SafeIDArgs(1),
		RunE: func(c *cobra.Command, args []string) error { return nil },
	}
	cmd.SetArgs([]string{"abc?fields=name"})
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected SafeIDArgs to reject injected ID")
	}
}

func TestSafeIDArgs_RejectsTraversal(t *testing.T) {
	cmd := &cobra.Command{
		Use:  "view <id>",
		Args: cmdutil.SafeIDArgs(1),
		RunE: func(c *cobra.Command, args []string) error { return nil },
	}
	cmd.SetArgs([]string{".."})
	if err := cmd.Execute(); err == nil {
		t.Fatal("expected SafeIDArgs to reject path traversal")
	}
}

func TestSafeIDArgs_AcceptsValidID(t *testing.T) {
	called := false
	cmd := &cobra.Command{
		Use:  "view <id>",
		Args: cmdutil.SafeIDArgs(1),
		RunE: func(c *cobra.Command, args []string) error {
			called = true
			return nil
		},
	}
	cmd.SetArgs([]string{"prod_abc123"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !called {
		t.Fatal("expected RunE to be called for valid ID")
	}
}

func TestSafeIDArgs_RejectsWrongArgCount(t *testing.T) {
	cmd := &cobra.Command{
		Use:  "view <id>",
		Args: cmdutil.SafeIDArgs(1),
		RunE: func(c *cobra.Command, args []string) error { return nil },
	}
	cmd.SetArgs([]string{})
	if err := cmd.Execute(); err == nil {
		t.Fatal("expected SafeIDArgs to require positional arg")
	}
}

func TestRequireSafeIDFlag_RejectsInjected(t *testing.T) {
	cmd := &cobra.Command{Use: "list"}
	var product string
	cmd.Flags().StringVar(&product, "product", "", "product id")
	if err := cmd.Flags().Set("product", "abc?fields=name"); err != nil {
		t.Fatalf("flag set: %v", err)
	}
	err := cmdutil.RequireSafeIDFlag(cmd, "product", product)
	if err == nil {
		t.Fatal("expected RequireSafeIDFlag to reject injected value")
	}
}

func TestRequireSafeIDFlag_AcceptsUnchangedFlag(t *testing.T) {
	cmd := &cobra.Command{Use: "list"}
	var product string
	cmd.Flags().StringVar(&product, "product", "", "product id")
	if err := cmdutil.RequireSafeIDFlag(cmd, "product", product); err != nil {
		t.Errorf("unset flag should pass, got %v", err)
	}
}
