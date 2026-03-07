package payouts

import (
	"github.com/spf13/cobra"
)

func NewPayoutsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "payouts",
		Short: "View your payouts",
		Example: `  gr payouts list
  gr payouts view <id>
  gr payouts upcoming`,
	}

	cmd.AddCommand(newListCmd())
	cmd.AddCommand(newViewCmd())
	cmd.AddCommand(newUpcomingCmd())

	return cmd
}
