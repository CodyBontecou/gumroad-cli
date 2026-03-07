package products

import (
	"github.com/antiwork/gr/internal/cmd/skus"
	"github.com/spf13/cobra"
)

func NewProductsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "products",
		Short: "Manage your Gumroad products",
		Long: "Manage your Gumroad products.\n\n" +
			"Gumroad's API does not support creating or updating products. " +
			"Use the web UI to create or edit products; `gr` supports listing, viewing, deleting, enabling, and disabling them.",
		Example: `  gr products list
  gr products view <id>
  gr products delete <id>
  gr products skus <id>`,
	}

	cmd.AddCommand(newListCmd())
	cmd.AddCommand(newViewCmd())
	cmd.AddCommand(newDeleteCmd())
	cmd.AddCommand(newEnableCmd())
	cmd.AddCommand(newDisableCmd())
	cmd.AddCommand(skus.NewProductSKUsCmd())

	return cmd
}
