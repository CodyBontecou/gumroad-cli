package purchases

import (
	"fmt"
	"net/url"

	"github.com/antiwork/gumroad-cli/internal/admincmd"
	"github.com/antiwork/gumroad-cli/internal/api"
	"github.com/antiwork/gumroad-cli/internal/cmdutil"
	"github.com/antiwork/gumroad-cli/internal/output"
	"github.com/spf13/cobra"
)

type purchaseResponse struct {
	Purchase purchase `json:"purchase"`
}

type purchase struct {
	ID                              string      `json:"id"`
	Email                           string      `json:"email"`
	SellerEmail                     string      `json:"seller_email"`
	ProductName                     string      `json:"product_name"`
	ProductAlias                    string      `json:"link_name"`
	ProductID                       string      `json:"product_id"`
	FormattedTotalPrice             string      `json:"formatted_total_price"`
	PriceCents                      api.JSONInt `json:"price_cents"`
	CurrencyType                    string      `json:"currency_type"`
	AmountRefundableCentsInCurrency api.JSONInt `json:"amount_refundable_cents_in_currency"`
	PurchaseState                   string      `json:"purchase_state"`
	RefundStatus                    string      `json:"refund_status"`
	CreatedAt                       string      `json:"created_at"`
	ReceiptURL                      string      `json:"receipt_url"`
}

func newViewCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "view <purchase-id>",
		Short: "View an admin purchase record",
		Args:  cmdutil.ExactArgs(1),
		RunE: func(c *cobra.Command, args []string) error {
			opts := cmdutil.OptionsFrom(c)
			path := cmdutil.JoinPath("purchases", args[0])
			return admincmd.RunGetDecoded[purchaseResponse](opts, "Fetching purchase...", path, url.Values{}, func(resp purchaseResponse) error {
				return renderPurchase(opts, resp.Purchase)
			})
		},
	}
}

func renderPurchase(opts cmdutil.Options, p purchase) error {
	product := p.ProductName
	if product == "" {
		product = p.ProductAlias
	}
	if product == "" {
		product = p.ProductID
	}

	amount := p.FormattedTotalPrice
	if amount == "" && p.PriceCents != 0 {
		amount = fmt.Sprintf("%d cents", p.PriceCents)
	}

	status := p.PurchaseState
	if p.RefundStatus != "" {
		if status != "" {
			status += ", "
		}
		status += p.RefundStatus
	}

	if opts.PlainOutput {
		return output.PrintPlain(opts.Out(), [][]string{
			{p.ID, p.Email, p.SellerEmail, product, amount, status, p.CreatedAt, p.ReceiptURL},
		})
	}

	headlineFromID := false
	headline := product
	if headline == "" {
		headline = p.ID
		headlineFromID = true
	}
	if amount != "" {
		headline += "  " + amount
	}

	rows := make([][2]string, 0, 6)
	if !headlineFromID {
		rows = append(rows, [2]string{"Purchase ID", p.ID})
	}
	if p.Email != "" {
		rows = append(rows, [2]string{"Buyer", p.Email})
	}
	if p.SellerEmail != "" {
		rows = append(rows, [2]string{"Seller", p.SellerEmail})
	}
	if status != "" {
		rows = append(rows, [2]string{"Status", status})
	}
	if p.CreatedAt != "" {
		rows = append(rows, [2]string{"Date", p.CreatedAt})
	}
	if p.ReceiptURL != "" {
		rows = append(rows, [2]string{"Receipt", p.ReceiptURL})
	}

	theme := opts.Theme()
	return theme.PrintCard(opts.Out(), headline, rows)
}
