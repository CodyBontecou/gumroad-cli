package sales

import (
	"fmt"
	"net/url"
	"strconv"

	"github.com/antiwork/gumroad-cli/internal/cmdutil"
	"github.com/spf13/cobra"
)

func newRefundCmd() *cobra.Command {
	var amount string
	var amountCents int

	cmd := &cobra.Command{
		Use:   "refund <id>",
		Short: "Refund a sale",
		Args:  cmdutil.ExactArgs(1),
		RunE: func(c *cobra.Command, args []string) error {
			opts := cmdutil.OptionsFrom(c)

			if err := cmdutil.RequirePositiveIntFlag(c, "amount-cents", amountCents); err != nil {
				return err
			}

			cents, hasAmount, err := cmdutil.ResolveMoneyFlag(c, "amount", "amount-cents", "amount", "", amountCents, amount, false)
			if err != nil {
				return err
			}
			if hasAmount && cents == 0 {
				return cmdutil.UsageErrorf(c, "--amount must be greater than 0")
			}

			// Build the human-readable amount description used in prompts and messages.
			amountDesc := refundAmountDesc(cents, c.Flags().Changed("amount"))

			msg := "Refund sale " + args[0] + "?"
			if cents > 0 {
				msg = fmt.Sprintf("Refund %s on sale %s?", amountDesc, args[0])
			}

			ok, err := cmdutil.ConfirmAction(opts, msg)
			if err != nil {
				return err
			}
			if !ok {
				action := "refund sale " + args[0]
				if cents > 0 {
					action = fmt.Sprintf("refund %s on sale %s", amountDesc, args[0])
				}
				return cmdutil.PrintCancelledAction(opts, action)
			}

			params := url.Values{}
			successMessage := "Sale " + args[0] + " refunded."
			if cents > 0 {
				params.Set("amount_cents", strconv.Itoa(cents))
				successMessage = fmt.Sprintf("Refunded %s on sale %s.", amountDesc, args[0])
			}

			return cmdutil.RunRequestWithSuccess(opts, "Refunding sale...", "PUT", cmdutil.JoinPath("sales", args[0], "refund"), params, successMessage)
		},
	}

	cmd.Flags().StringVar(&amount, "amount", "", "Partial refund amount (e.g. 5, 5.00)")
	cmd.Flags().IntVar(&amountCents, "amount-cents", 0, "Partial refund amount in cents (deprecated, use --amount)")
	_ = cmd.Flags().MarkHidden("amount-cents")

	return cmd
}

// refundAmountDesc returns a human-readable description of the refund amount.
// When the new --amount flag was used, it normalizes to "N.NN" format.
// When the deprecated --amount-cents flag was used, it shows "N cents".
func refundAmountDesc(cents int, usedNewFlag bool) string {
	if usedNewFlag {
		return cmdutil.FormatMoney(cents, "")
	}
	return fmt.Sprintf("%d cents", cents)
}
