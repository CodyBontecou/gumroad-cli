package sales

import (
	"github.com/spf13/cobra"
)

func NewSalesCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "sales",
		Short: "Manage your sales",
		Example: `  gr sales list
  gr sales view <id>
  gr sales refund <id>`,
	}

	cmd.AddCommand(newListCmd())
	cmd.AddCommand(newViewCmd())
	cmd.AddCommand(newRefundCmd())
	cmd.AddCommand(newShipCmd())
	cmd.AddCommand(newResendReceiptCmd())

	return cmd
}
