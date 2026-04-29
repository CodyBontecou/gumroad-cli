package schema

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/antiwork/gumroad-cli/internal/cmdutil"
	"github.com/antiwork/gumroad-cli/internal/output"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

func NewSchemaCmd(rootProvider func() *cobra.Command) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "schema [command...]",
		Short: "Print the schema of a command as JSON",
		Long: `Emit a machine-readable description of a command's flags, positional args,
and required environment variables. Designed for agents that need to
discover the CLI surface at runtime instead of preloading docs.

Examples:
  gumroad schema sales refund
  gumroad schema products create
  gumroad schema       # list every command path`,
		RunE: func(c *cobra.Command, args []string) error {
			opts := cmdutil.OptionsFrom(c)
			root := rootProvider()
			if root == nil {
				return fmt.Errorf("schema: command tree unavailable")
			}

			if len(args) == 0 {
				return printCommandList(opts, root)
			}

			target, missing, err := resolveCommand(root, args)
			if err != nil {
				return err
			}
			if target == nil {
				return cmdutil.UsageErrorf(c, "unknown command: %q", strings.Join(missing, " "))
			}

			payload := describeCommand(target)
			return output.PrintJSON(opts.Out(), mustMarshal(payload), opts.JQExpr)
		},
	}
	return cmd
}

func resolveCommand(root *cobra.Command, path []string) (*cobra.Command, []string, error) {
	current := root
	for i, name := range path {
		next, _, err := current.Find([]string{name})
		if err != nil || next == nil || next == current {
			return nil, path[:i+1], nil
		}
		current = next
	}
	return current, nil, nil
}

func printCommandList(opts cmdutil.Options, root *cobra.Command) error {
	var paths []string
	walkCommands(root, func(c *cobra.Command) {
		if c == root {
			return
		}
		paths = append(paths, c.CommandPath())
	})
	payload := map[string]any{
		"root":     root.Name(),
		"commands": paths,
	}
	return output.PrintJSON(opts.Out(), mustMarshal(payload), opts.JQExpr)
}

func walkCommands(c *cobra.Command, fn func(*cobra.Command)) {
	fn(c)
	for _, child := range c.Commands() {
		if child.Hidden || child.Name() == "help" {
			continue
		}
		walkCommands(child, fn)
	}
}

type commandSchema struct {
	Path        string       `json:"path"`
	Name        string       `json:"name"`
	Short       string       `json:"short,omitempty"`
	Long        string       `json:"long,omitempty"`
	Use         string       `json:"use"`
	Example     string       `json:"example,omitempty"`
	Subcommands []string     `json:"subcommands,omitempty"`
	Args        []argSchema  `json:"args,omitempty"`
	Flags       []flagSchema `json:"flags,omitempty"`
	Globals     []flagSchema `json:"globals,omitempty"`
	Env         []envSchema  `json:"env,omitempty"`
	Hidden      bool         `json:"hidden,omitempty"`
}

type argSchema struct {
	Name     string `json:"name"`
	Required bool   `json:"required"`
}

type flagSchema struct {
	Name        string `json:"name"`
	Shorthand   string `json:"shorthand,omitempty"`
	Type        string `json:"type"`
	Default     string `json:"default,omitempty"`
	Description string `json:"description,omitempty"`
	Required    bool   `json:"required,omitempty"`
}

type envSchema struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

func describeCommand(c *cobra.Command) commandSchema {
	out := commandSchema{
		Path:    c.CommandPath(),
		Name:    c.Name(),
		Short:   strings.TrimSpace(c.Short),
		Long:    strings.TrimSpace(c.Long),
		Use:     c.Use,
		Example: strings.TrimSpace(c.Example),
		Hidden:  c.Hidden,
	}

	for _, sub := range c.Commands() {
		if sub.Hidden || sub.Name() == "help" {
			continue
		}
		out.Subcommands = append(out.Subcommands, sub.Name())
	}

	for _, a := range positionalArgs(c.Use) {
		out.Args = append(out.Args, argSchema{
			Name:     strings.Trim(a, "<>[]"),
			Required: strings.HasPrefix(a, "<"),
		})
	}

	c.LocalFlags().VisitAll(func(f *pflag.Flag) {
		out.Flags = append(out.Flags, describeFlag(f))
	})
	c.InheritedFlags().VisitAll(func(f *pflag.Flag) {
		out.Globals = append(out.Globals, describeFlag(f))
	})

	out.Env = inferEnv(c)
	return out
}

func describeFlag(f *pflag.Flag) flagSchema {
	desc := strings.TrimSpace(f.Usage)
	required := strings.Contains(strings.ToLower(desc), "(required)")
	return flagSchema{
		Name:        f.Name,
		Shorthand:   f.Shorthand,
		Type:        f.Value.Type(),
		Default:     f.DefValue,
		Description: desc,
		Required:    required,
	}
}

func positionalArgs(use string) []string {
	var args []string
	for _, field := range strings.Fields(use) {
		if (strings.HasPrefix(field, "<") && strings.HasSuffix(field, ">")) ||
			(strings.HasPrefix(field, "[") && strings.HasSuffix(field, "]")) {
			args = append(args, field)
		}
	}
	return args
}

func inferEnv(c *cobra.Command) []envSchema {
	path := c.CommandPath()
	var env []envSchema
	if requiresAPIToken(path) {
		env = append(env, envSchema{
			Name:        "GUMROAD_ACCESS_TOKEN",
			Description: "Gumroad API access token used by this command.",
		})
	}
	if strings.HasPrefix(stripRoot(path), "admin") {
		env = append(env, envSchema{
			Name:        "GUMROAD_ADMIN_ACCESS_TOKEN",
			Description: "Admin OAuth token required for admin subcommands.",
		})
	}
	return env
}

func requiresAPIToken(path string) bool {
	rest := stripRoot(path)
	if rest == "" {
		return false
	}
	for _, prefix := range []string{"auth", "completion", "schema", "skill"} {
		if rest == prefix || strings.HasPrefix(rest, prefix+" ") {
			return false
		}
	}
	return true
}

func stripRoot(path string) string {
	parts := strings.SplitN(path, " ", 2)
	if len(parts) < 2 {
		return ""
	}
	return parts[1]
}

func mustMarshal(v any) []byte {
	b, err := json.Marshal(v)
	if err != nil {
		return []byte(fmt.Sprintf(`{"error":%q}`, err.Error()))
	}
	return b
}
