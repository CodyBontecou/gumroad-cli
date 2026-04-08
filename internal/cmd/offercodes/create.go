package offercodes

import (
	"net/url"
	"strconv"

	"github.com/antiwork/gumroad-cli/internal/cmdutil"
	"github.com/spf13/cobra"
)

func newCreateCmd() *cobra.Command {
	var product, name, amount string
	var amountCents, percentOff, maxPurchaseCount int
	var universal bool

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create an offer code",
		Args:  cmdutil.ExactArgs(0),
		Long: `Create an offer code for a product.

Use either --amount (flat discount) or --percent-off (percentage discount), not both.`,
		RunE: func(c *cobra.Command, args []string) error {
			if err := cmdutil.RequirePositiveIntFlag(c, "amount-cents", amountCents); err != nil {
				return err
			}
			if err := cmdutil.RequirePercentFlag(c, "percent-off", percentOff); err != nil {
				return err
			}
			if err := cmdutil.RequireNonNegativeIntFlag(c, "max-purchase-count", maxPurchaseCount); err != nil {
				return err
			}
			if product == "" {
				return cmdutil.MissingFlagError(c, "--product")
			}
			if name == "" {
				return cmdutil.MissingFlagError(c, "--name")
			}

			cents, hasAmountOff, err := cmdutil.ResolveMoneyFlag(c, "amount", "amount-cents", "amount", "", amountCents, amount, false)
			if err != nil {
				return err
			}
			if hasAmountOff && cents == 0 {
				return cmdutil.UsageErrorf(c, "--amount must be greater than 0")
			}

			hasPercentOff := c.Flags().Changed("percent-off")
			hasMaxPurchaseCount := c.Flags().Changed("max-purchase-count")
			if hasAmountOff && hasPercentOff {
				amountFlag := "--amount"
				if c.Flags().Changed("amount-cents") {
					amountFlag = "--amount-cents"
				}
				return cmdutil.UsageErrorf(c, "flags %s and --percent-off cannot be used together", amountFlag)
			}
			if !hasAmountOff && !hasPercentOff {
				return cmdutil.UsageErrorf(c, "one of --amount or --percent-off is required")
			}

			params := url.Values{}
			params.Set("name", name)
			if hasAmountOff {
				params.Set("amount_off", strconv.Itoa(cents))
			}
			if hasPercentOff {
				params.Set("percent_off", strconv.Itoa(percentOff))
			}
			if hasMaxPurchaseCount {
				params.Set("max_purchase_count", strconv.Itoa(maxPurchaseCount))
			}
			if universal {
				params.Set("universal", "true")
			}

			opts := cmdutil.OptionsFrom(c)
			return cmdutil.RunRequestWithSuccess(opts, "Creating offer code...", "POST", cmdutil.JoinPath("products", product, "offer_codes"), params, "Offer code "+name+" created.")
		},
	}

	cmd.Flags().StringVar(&product, "product", "", "Product ID (required)")
	cmd.Flags().StringVar(&name, "name", "", "Offer code name (required)")
	cmd.Flags().StringVar(&amount, "amount", "", "Flat discount amount (e.g. 5, 5.00)")
	cmd.Flags().IntVar(&amountCents, "amount-cents", 0, "Flat discount in cents (deprecated, use --amount)")
	_ = cmd.Flags().MarkHidden("amount-cents")
	cmd.Flags().IntVar(&percentOff, "percent-off", 0, "Percentage discount")
	cmd.Flags().IntVar(&maxPurchaseCount, "max-purchase-count", 0, "Maximum number of uses")
	cmd.Flags().BoolVar(&universal, "universal", false, "Universal offer code")

	return cmd
}
