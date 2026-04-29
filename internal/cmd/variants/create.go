package variants

import (
	"net/url"
	"strconv"

	"github.com/antiwork/gumroad-cli/internal/cmdutil"
	"github.com/antiwork/gumroad-cli/internal/output"
	"github.com/spf13/cobra"
)

type createVariantResponse struct {
	Variant struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	} `json:"variant"`
}

var createBodyFlags = []string{"name", "description", "price-difference", "max-purchase-count"}

func newCreateCmd() *cobra.Command {
	var product, category, name, description, priceDifference, jsonBody string
	var maxPurchaseCount int

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a variant",
		Args:  cmdutil.ExactArgs(0),
		RunE: func(c *cobra.Command, args []string) error {
			opts := cmdutil.OptionsFrom(c)

			if product == "" {
				return cmdutil.MissingFlagError(c, "--product")
			}
			if category == "" {
				return cmdutil.MissingFlagError(c, "--category")
			}

			path := cmdutil.JoinPath("products", product, "variant_categories", category, "variants")

			if c.Flags().Changed("json-body") {
				if err := cmdutil.RejectFlagsWithJSONBody(c, createBodyFlags...); err != nil {
					return err
				}
				params, err := cmdutil.ParseJSONBody(jsonBody, opts.In())
				if err != nil {
					return cmdutil.UsageErrorf(c, "%s", err.Error())
				}
				return runCreateRequest(opts, path, params)
			}

			if err := cmdutil.RequireNonNegativeIntFlag(c, "max-purchase-count", maxPurchaseCount); err != nil {
				return err
			}
			if name == "" {
				return cmdutil.MissingFlagError(c, "--name")
			}

			flags := c.Flags()
			hasPriceDifference := flags.Changed("price-difference")
			hasMaxPurchaseCount := flags.Changed("max-purchase-count")

			params := url.Values{}
			params.Set("name", name)
			if description != "" {
				params.Set("description", description)
			}
			if hasPriceDifference {
				cents, err := cmdutil.ParseSignedMoney("price-difference", priceDifference, "price", "")
				if err != nil {
					return cmdutil.UsageErrorf(c, "%s", err.Error())
				}
				params.Set("price_difference_cents", strconv.Itoa(cents))
			}
			if hasMaxPurchaseCount {
				params.Set("max_purchase_count", strconv.Itoa(maxPurchaseCount))
			}

			return runCreateRequest(opts, path, params)
		},
	}

	cmd.Flags().StringVar(&product, "product", "", "Product ID (required)")
	cmd.Flags().StringVar(&category, "category", "", "Variant category ID (required)")
	cmd.Flags().StringVar(&name, "name", "", "Variant name (required)")
	cmd.Flags().StringVar(&description, "description", "", "Variant description")
	cmd.Flags().StringVar(&priceDifference, "price-difference", "", "Price difference (e.g. 5.00, -1.50)")
	cmd.Flags().IntVar(&maxPurchaseCount, "max-purchase-count", 0, "Maximum number of purchases")
	cmd.Flags().StringVar(&jsonBody, "json-body", "", "Raw JSON body (or '-' to read from stdin) — replaces individual body flags")

	return cmd
}

func runCreateRequest(opts cmdutil.Options, path string, params url.Values) error {
	return cmdutil.RunRequestDecoded[createVariantResponse](opts,
		"Creating variant...", "POST", path, params,
		func(resp createVariantResponse) error {
			v := resp.Variant
			if opts.PlainOutput {
				return output.PrintPlain(opts.Out(), [][]string{{v.ID, v.Name}})
			}
			if opts.Quiet {
				return nil
			}
			s := opts.Style()
			return output.Writef(opts.Out(), "%s %s (%s)\n",
				s.Bold("Created variant:"), v.Name, s.Dim(v.ID))
		})
}
