package completion

import (
	"github.com/antiwork/gr/internal/cmdutil"
	"github.com/spf13/cobra"
)

func NewCompletionCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "completion <bash|zsh|fish|powershell>",
		Short: "Generate shell completion script",
		Long: `Generate shell completion script for gr.

  # Bash
  source <(gr completion bash)

  # Zsh
  gr completion zsh > "${fpath[1]}/_gr"

  # Fish
  gr completion fish | source

  # PowerShell
  gr completion powershell | Out-String | Invoke-Expression`,
		Example: `  gr completion bash
  gr completion zsh`,
		Args: func(cmd *cobra.Command, args []string) error {
			if err := cmdutil.ExactArgs(1)(cmd, args); err != nil {
				return err
			}
			for _, shell := range cmd.ValidArgs {
				if args[0] == shell {
					return nil
				}
			}
			return cmdutil.UsageErrorf(cmd, "invalid shell: %s", args[0])
		},
		ValidArgs: []string{"bash", "zsh", "fish", "powershell"},
		RunE: func(c *cobra.Command, args []string) error {
			switch args[0] {
			case "bash":
				return c.Root().GenBashCompletion(c.OutOrStdout())
			case "zsh":
				return c.Root().GenZshCompletion(c.OutOrStdout())
			case "fish":
				return c.Root().GenFishCompletion(c.OutOrStdout(), true)
			case "powershell":
				return c.Root().GenPowerShellCompletionWithDesc(c.OutOrStdout())
			default:
				return cmdutil.UsageErrorf(c, "invalid shell: %s", args[0])
			}
		},
	}

	return cmd
}
