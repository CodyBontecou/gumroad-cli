package schema_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/antiwork/gumroad-cli/internal/cmd/schema"
	"github.com/antiwork/gumroad-cli/internal/testutil"
	"github.com/spf13/cobra"
)

func newFakeRoot() *cobra.Command {
	root := &cobra.Command{Use: "gumroad", Short: "fake"}
	root.PersistentFlags().Bool("json", false, "Output as JSON")

	products := &cobra.Command{Use: "products", Short: "Manage products"}
	root.AddCommand(products)

	view := &cobra.Command{
		Use:   "view <id>",
		Short: "View a product",
		RunE:  func(c *cobra.Command, args []string) error { return nil },
	}
	view.Flags().String("name", "", "New product name")
	view.Flags().Int("limit", 0, "Page limit")
	products.AddCommand(view)

	return root
}

func runSchema(t *testing.T, args []string) string {
	t.Helper()
	cmd := schema.NewSchemaCmd(newFakeRoot)
	bound := testutil.Command(cmd, testutil.JSONOutput())
	bound.SetArgs(args)
	return testutil.CaptureStdout(func() { testutil.MustExecute(t, bound) })
}

func TestSchema_TopLevelLists(t *testing.T) {
	out := runSchema(t, []string{})
	var resp map[string]any
	if err := json.Unmarshal([]byte(out), &resp); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, out)
	}
	commands, ok := resp["commands"].([]any)
	if !ok {
		t.Fatalf("expected commands array, got %T", resp["commands"])
	}
	want := "gumroad products view"
	found := false
	for _, c := range commands {
		if c == want {
			found = true
		}
	}
	if !found {
		t.Errorf("commands list missing %q: %v", want, commands)
	}
}

func TestSchema_DescribesLeafCommand(t *testing.T) {
	out := runSchema(t, []string{"products", "view"})
	var resp map[string]any
	if err := json.Unmarshal([]byte(out), &resp); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, out)
	}
	if resp["name"] != "view" {
		t.Errorf("name=%v want view", resp["name"])
	}
	if resp["path"] != "gumroad products view" {
		t.Errorf("path=%v", resp["path"])
	}
	args, _ := resp["args"].([]any)
	if len(args) != 1 {
		t.Fatalf("args=%v", args)
	}
	first, _ := args[0].(map[string]any)
	if first["name"] != "id" {
		t.Errorf("arg name=%v", first["name"])
	}
	if first["required"] != true {
		t.Errorf("arg required=%v", first["required"])
	}
}

func TestSchema_FlagsHaveTypes(t *testing.T) {
	out := runSchema(t, []string{"products", "view"})
	var resp map[string]any
	if err := json.Unmarshal([]byte(out), &resp); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, out)
	}
	flags, _ := resp["flags"].([]any)
	types := map[string]string{}
	for _, f := range flags {
		fm, _ := f.(map[string]any)
		types[fm["name"].(string)] = fm["type"].(string)
	}
	if types["name"] != "string" {
		t.Errorf("name type=%v", types["name"])
	}
	if types["limit"] != "int" {
		t.Errorf("limit type=%v", types["limit"])
	}
}

func TestSchema_GlobalFlagsListed(t *testing.T) {
	out := runSchema(t, []string{"products", "view"})
	var resp map[string]any
	if err := json.Unmarshal([]byte(out), &resp); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	globals, _ := resp["globals"].([]any)
	found := false
	for _, g := range globals {
		gm, _ := g.(map[string]any)
		if gm["name"] == "json" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected global --json flag in globals: %v", globals)
	}
}

func TestSchema_UnknownCommandErrors(t *testing.T) {
	cmd := schema.NewSchemaCmd(newFakeRoot)
	bound := testutil.Command(cmd, testutil.JSONOutput())
	bound.SetArgs([]string{"does", "not", "exist"})
	if err := bound.Execute(); err == nil || !strings.Contains(err.Error(), "unknown") {
		t.Fatalf("expected unknown command error, got %v", err)
	}
}

func TestSchema_EnvForAPICommand(t *testing.T) {
	out := runSchema(t, []string{"products", "view"})
	var resp map[string]any
	if err := json.Unmarshal([]byte(out), &resp); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	env, _ := resp["env"].([]any)
	found := false
	for _, e := range env {
		em, _ := e.(map[string]any)
		if em["name"] == "GUMROAD_ACCESS_TOKEN" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected GUMROAD_ACCESS_TOKEN in env: %v", env)
	}
}
