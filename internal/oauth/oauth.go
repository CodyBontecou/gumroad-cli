package oauth

import "time"

const (
	// ClientID is the public OAuth application UID for the Gumroad CLI.
	// This is a public client (confidential: false) — the client ID is not secret.
	ClientID = "PLACEHOLDER_REPLACE_BEFORE_RELEASE"

	AuthorizeURL = "https://app.gumroad.com/oauth/authorize"
	TokenURL = "https://app.gumroad.com/oauth/token" //nolint:gosec // G101: not a credential

	Scopes = "edit_products view_sales mark_sales_as_shipped edit_sales view_payouts view_profile account"

	DefaultTimeout = 2 * time.Minute
)
