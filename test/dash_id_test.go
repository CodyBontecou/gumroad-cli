package test

import (
	"bytes"
	"net/http"
	"strings"
	"testing"

	cmd "github.com/antiwork/gumroad-cli/internal/cmd"
	"github.com/antiwork/gumroad-cli/internal/testutil"
)

// TestDashPrefixedIDArgs verifies that every command accepting a positional ID
// argument correctly handles IDs starting with "-". Without
// ParseErrorsWhitelist.UnknownFlags, cobra interprets these as shorthand flags
// and rejects them. The setting is applied centrally by cmdutil.AllowDashIDs.
func TestDashPrefixedIDArgs(t *testing.T) {
	testutil.Setup(t, func(w http.ResponseWriter, r *http.Request) {
		testutil.JSON(t, w, map[string]any{
			"product":          map[string]any{"id": "p1", "name": "Test"},
			"offer_code":       map[string]any{"id": "oc1", "name": "TEST"},
			"variant_category": map[string]any{"id": "vc1", "title": "Size"},
			"variant":          map[string]any{"id": "v1", "name": "Small"},
			"sale":             map[string]any{"id": "s1", "email": "test@example.com", "product_name": "Test"},
			"subscriber":       map[string]any{"id": "sub1", "email_address": "test@example.com"},
			"payout":           map[string]any{"id": "pay1", "display_payout_period": "Jan 2026", "formatted_amount": "$100"},
			"skus":             []map[string]any{},
		})
	})

	dashID := "-cGksPcArAUU8j_XTYsrnQ=="

	tests := []struct {
		name string
		args []string
	}{
		// products
		{"products view", []string{"products", "view", dashID}},
		{"products update", []string{"products", "update", "--name", "x", dashID}},
		{"products delete", []string{"products", "delete", "--yes", dashID}},
		{"products publish", []string{"products", "publish", dashID}},
		{"products unpublish", []string{"products", "unpublish", dashID}},
		{"products skus", []string{"products", "skus", dashID}},

		// offer-codes
		{"offer-codes view", []string{"offer-codes", "view", "--product", "p1", dashID}},
		{"offer-codes update", []string{"offer-codes", "update", "--product", "p1", "--max-purchase-count", "5", dashID}},
		{"offer-codes delete", []string{"offer-codes", "delete", "--product", "p1", "--yes", dashID}},

		// variant-categories
		{"variant-categories view", []string{"variant-categories", "view", "--product", "p1", dashID}},
		{"variant-categories update", []string{"variant-categories", "update", "--product", "p1", "--title", "Color", dashID}},
		{"variant-categories delete", []string{"variant-categories", "delete", "--product", "p1", "--yes", dashID}},

		// variants
		{"variants view", []string{"variants", "view", "--product", "p1", "--category", "c1", dashID}},
		{"variants update", []string{"variants", "update", "--product", "p1", "--category", "c1", "--name", "Large", dashID}},
		{"variants delete", []string{"variants", "delete", "--product", "p1", "--category", "c1", "--yes", dashID}},

		// sales
		{"sales view", []string{"sales", "view", dashID}},
		{"sales refund", []string{"sales", "refund", "--yes", dashID}},
		{"sales resend-receipt", []string{"sales", "resend-receipt", dashID}},
		{"sales ship", []string{"sales", "ship", dashID}},

		// subscribers
		{"subscribers view", []string{"subscribers", "view", dashID}},

		// payouts
		{"payouts view", []string{"payouts", "view", dashID}},

		// webhooks
		{"webhooks delete", []string{"webhooks", "delete", "--yes", dashID}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := cmd.NewRootCmd()
			root.SetArgs(append(tt.args, "--no-color"))
			var out bytes.Buffer
			root.SetOut(&out)
			root.SetErr(&out)

			err := root.Execute()
			if err != nil {
				combined := out.String()
				if strings.Contains(combined, "unknown shorthand flag") || strings.Contains(err.Error(), "unknown shorthand flag") {
					t.Fatalf("dash-prefixed ID %q was parsed as a flag: %v", dashID, err)
				}
				// Other errors (e.g. mock server doesn't match exactly) are acceptable —
				// the point is that the ID was not mistaken for a flag.
			}
		})
	}
}
