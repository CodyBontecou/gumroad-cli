package variants

import (
	"net/url"

	"github.com/antiwork/gumroad-cli/internal/cmdutil"
	"github.com/spf13/cobra"
)

func newDeleteCmd() *cobra.Command {
	var product, category string

	cmd := &cobra.Command{
		Use:   "delete <variant_id>",
		Short: "Delete a variant",
		Args:  cmdutil.SafeIDArgs(1),
		RunE: func(c *cobra.Command, args []string) error {
			opts := cmdutil.OptionsFrom(c)
			if product == "" {
				return cmdutil.MissingFlagError(c, "--product")
			}
			if category == "" {
				return cmdutil.MissingFlagError(c, "--category")
			}
			if err := cmdutil.RequireSafeIDFlag(c, "product", product); err != nil {
				return err
			}
			if err := cmdutil.RequireSafeIDFlag(c, "category", category); err != nil {
				return err
			}

			ok, err := cmdutil.ConfirmAction(opts, "Delete variant "+args[0]+"?")
			if err != nil {
				return err
			}
			if !ok {
				return cmdutil.PrintCancelledAction(opts, "delete variant "+args[0], args[0])
			}

			path := cmdutil.JoinPath("products", product, "variant_categories", category, "variants", args[0])
			return cmdutil.RunRequestWithSuccess(opts, "Deleting variant...", "DELETE", path, url.Values{}, args[0], "Variant "+args[0]+" deleted.")
		},
	}

	cmd.Flags().StringVar(&product, "product", "", "Product ID (required)")
	cmd.Flags().StringVar(&category, "category", "", "Variant category ID (required)")

	return cmd
}
