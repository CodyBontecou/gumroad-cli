package buy

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/antiwork/gumroad-cli/internal/api"
	"github.com/antiwork/gumroad-cli/internal/cmdutil"
	"github.com/antiwork/gumroad-cli/internal/config"
	"github.com/antiwork/gumroad-cli/internal/output"
	"github.com/spf13/cobra"
)

const (
	defaultLineItemUID = "li-0"
	ordersPath         = "/orders"
)

type buyLineItemResult struct {
	UID             string `json:"uid"`
	Success         bool   `json:"success"`
	ErrorMessage    string `json:"error_message"`
	Name            string `json:"name"`
	Permalink       string `json:"permalink"`
	ContentURL      string `json:"content_url"`
	RedirectToken   string `json:"redirect_token"`
	RequiresAction  bool   `json:"requires_action"`
	ClientSecret    string `json:"client_secret"`
	ConfirmationURL string `json:"confirmation_url"`
}

type buyResponse struct {
	Success   bool                         `json:"success"`
	Message   string                       `json:"message"`
	LineItems map[string]buyLineItemResult `json:"line_items"`
}

func NewBuyCmd() *cobra.Command {
	var paymentMethodID, customerID, email, variant, offerCode string
	var quantity, priceCents, tipCents int

	cmd := &cobra.Command{
		Use:   "buy <permalink>",
		Short: "Buy a product as the authenticated user",
		Long: "Buy a product on Gumroad as the authenticated user.\n\n" +
			"Pre-mint a Stripe PaymentMethod (pm_xxx) on Gumroad's Stripe platform " +
			"and pass it via --payment-method-id. 3DS / SCA challenges cannot be " +
			"completed in a CLI; when one is required, the command prints a " +
			"confirmation URL and exits non-zero so you can finish in a browser.\n\n" +
			"Requires an OAuth token with the create_purchases scope.",
		Example: `  gumroad buy abc123 --payment-method-id pm_card_visa --price-cents 500 --yes
  gumroad buy abc123 --pm pm_card_visa --price-cents 500 --yes
  gumroad buy abc123 --pm pm_card_visa --price-cents 500 --quantity 2 --offer-code SAVE10 --yes`,
		Args: cmdutil.ExactArgs(1),
		RunE: func(c *cobra.Command, args []string) error {
			opts := cmdutil.OptionsFrom(c)
			flags := c.Flags()
			permalink := args[0]

			if paymentMethodID == "" {
				return cmdutil.MissingFlagError(c, "--payment-method-id")
			}
			if !flags.Changed("price-cents") {
				return cmdutil.MissingFlagError(c, "--price-cents")
			}

			if flags.Changed("quantity") {
				if err := cmdutil.RequirePositiveIntFlag(c, "quantity", quantity); err != nil {
					return err
				}
			}
			if err := cmdutil.RequireNonNegativeIntFlag(c, "price-cents", priceCents); err != nil {
				return err
			}
			if flags.Changed("tip-cents") {
				if err := cmdutil.RequireNonNegativeIntFlag(c, "tip-cents", tipCents); err != nil {
					return err
				}
			}

			body := buildOrderBody(orderBodyInput{
				PaymentMethodID: paymentMethodID,
				CustomerID:      customerID,
				CustomerIDSet:   flags.Changed("customer-id"),
				Email:           email,
				EmailSet:        flags.Changed("email"),
				Permalink:       permalink,
				PerceivedCents:  priceCents,
				Quantity:        quantity,
				Variant:         variant,
				VariantSet:      flags.Changed("variant"),
				OfferCode:       offerCode,
				OfferCodeSet:    flags.Changed("offer-code"),
				TipCents:        tipCents,
				TipCentsSet:     flags.Changed("tip-cents"),
			})

			if opts.DryRun {
				return renderDryRun(opts, body)
			}

			summary := buildBuySummary(permalink, paymentMethodID, priceCents, quantity)
			ok, err := cmdutil.ConfirmAction(opts, "Buy "+summary+"?")
			if err != nil {
				return err
			}
			if !ok {
				return cmdutil.PrintCancelledAction(opts, "buy "+permalink, permalink)
			}

			token, err := config.Token()
			if err != nil {
				return err
			}

			data, err := cmdutil.RunWithTokenData(opts, token, "Purchasing...",
				func(client *api.Client) (json.RawMessage, error) {
					return client.PostJSON(ordersPath, body)
				})
			if err != nil {
				return err
			}
			if opts.UsesJSONOutput() {
				return cmdutil.PrintJSONResponse(opts, data)
			}
			resp, err := cmdutil.DecodeJSON[buyResponse](data)
			if err != nil {
				return err
			}
			return renderBuyResult(opts, resp)
		},
	}

	cmd.Flags().StringVar(&paymentMethodID, "payment-method-id", "", "Stripe PaymentMethod ID (pm_xxx) on Gumroad's platform (required)")
	cmd.Flags().StringVar(&paymentMethodID, "pm", "", "Alias for --payment-method-id")
	cmd.Flags().StringVar(&customerID, "customer-id", "", "Stripe customer (cus_xxx) for saved-card flows")
	cmd.Flags().StringVar(&email, "email", "", "Override receipt email (default: the OAuth user's email)")
	cmd.Flags().IntVar(&quantity, "quantity", 1, "Quantity to purchase")
	cmd.Flags().IntVar(&priceCents, "price-cents", 0, "Perceived price in cents; the server validates this against the product's listed price (required)")
	cmd.Flags().StringVar(&variant, "variant", "", "Variant external id")
	cmd.Flags().StringVar(&offerCode, "offer-code", "", "Discount code")
	cmd.Flags().IntVar(&tipCents, "tip-cents", 0, "Optional tip in cents")

	return cmd
}

type orderBodyInput struct {
	PaymentMethodID string
	CustomerID      string
	CustomerIDSet   bool
	Email           string
	EmailSet        bool
	Permalink       string
	PerceivedCents  int
	Quantity        int
	Variant         string
	VariantSet      bool
	OfferCode       string
	OfferCodeSet    bool
	TipCents        int
	TipCentsSet     bool
}

func buildOrderBody(in orderBodyInput) map[string]any {
	lineItem := map[string]any{
		"uid":                   defaultLineItemUID,
		"permalink":             in.Permalink,
		"perceived_price_cents": in.PerceivedCents,
		"quantity":              in.Quantity,
	}
	if in.VariantSet && in.Variant != "" {
		lineItem["variants"] = []string{in.Variant}
	}
	if in.OfferCodeSet && in.OfferCode != "" {
		lineItem["discount_code"] = in.OfferCode
	}
	if in.TipCentsSet {
		lineItem["tip_cents"] = in.TipCents
	}

	body := map[string]any{
		"stripe_payment_method_id": in.PaymentMethodID,
		"line_items":               []map[string]any{lineItem},
	}
	if in.CustomerIDSet && in.CustomerID != "" {
		body["stripe_customer_id"] = in.CustomerID
	}
	if in.EmailSet && in.Email != "" {
		body["email"] = in.Email
	}
	return body
}

func buildBuySummary(permalink, paymentMethodID string, perceivedCents, quantity int) string {
	priceLabel := cmdutil.FormatMoney(perceivedCents, "")
	totalLabel := cmdutil.FormatMoney(perceivedCents*quantity, "")
	return fmt.Sprintf("%s, %s × %d = %s, paid with %s",
		permalink, priceLabel, quantity, totalLabel, paymentMethodID)
}

func renderDryRun(opts cmdutil.Options, body map[string]any) error {
	if opts.UsesJSONOutput() {
		payload := map[string]any{
			"dry_run": true,
			"request": map[string]any{
				"method": "POST",
				"path":   ordersPath,
				"body":   body,
			},
		}
		data, err := json.Marshal(payload)
		if err != nil {
			return fmt.Errorf("could not encode dry-run output: %w", err)
		}
		return output.PrintJSON(opts.Out(), data, opts.JQExpr)
	}

	if opts.PlainOutput {
		data, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("could not encode dry-run output: %w", err)
		}
		return output.PrintPlain(opts.Out(), [][]string{{"POST", ordersPath, string(data)}})
	}

	style := opts.Style()
	if err := output.Writeln(opts.Out(), style.Yellow("Dry run")+": POST "+ordersPath); err != nil {
		return err
	}
	data, err := json.MarshalIndent(body, "", "  ")
	if err != nil {
		return fmt.Errorf("could not encode dry-run output: %w", err)
	}
	return output.Writeln(opts.Out(), string(data))
}

func renderBuyResult(opts cmdutil.Options, resp buyResponse) error {
	item, ok := resp.LineItems[defaultLineItemUID]
	if !ok {
		for _, value := range resp.LineItems {
			item = value
			ok = true
			break
		}
	}
	if !ok {
		return errors.New("purchase response missing line items")
	}

	style := opts.Style()

	if item.RequiresAction {
		if err := output.Writeln(opts.Err(), style.Yellow("3DS verification required.")); err != nil {
			return err
		}
		if item.ConfirmationURL != "" {
			if err := output.Writef(opts.Err(), "Complete in browser: %s\n", item.ConfirmationURL); err != nil {
				return err
			}
		}
		return errors.New("3DS verification required; complete in browser to finish the purchase")
	}

	if !item.Success {
		message := item.ErrorMessage
		if message == "" {
			message = "purchase failed"
		}
		if err := output.Writeln(opts.Err(), style.Red(message)); err != nil {
			return err
		}
		return errors.New(message)
	}

	if opts.PlainOutput {
		return output.PrintPlain(opts.Out(), [][]string{{
			item.Permalink, item.Name, item.ContentURL, item.RedirectToken,
		}})
	}

	if opts.Quiet {
		return nil
	}

	heading := "Purchased " + item.Name
	if err := output.Writeln(opts.Out(), style.Bold(heading)); err != nil {
		return err
	}
	if item.ContentURL != "" {
		if err := output.Writef(opts.Out(), "%s %s\n", style.Dim("Content:"), item.ContentURL); err != nil {
			return err
		}
	}
	if item.RedirectToken != "" {
		if err := output.Writef(opts.Out(), "%s %s\n", style.Dim("Redirect token:"), item.RedirectToken); err != nil {
			return err
		}
	}
	return nil
}
