package subscribers

import (
	"github.com/spf13/cobra"
)

func NewSubscribersCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "subscribers",
		Short: "Manage product subscribers",
		Example: `  gr subscribers list --product <id>
  gr subscribers view <id>`,
	}

	cmd.AddCommand(newListCmd())
	cmd.AddCommand(newViewCmd())

	return cmd
}
