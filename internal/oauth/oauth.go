package oauth

import (
	"os"
	"time"
)

const (
	// DefaultClientID is the public OAuth application UID for the production Gumroad CLI.
	// This is a public client (confidential: false) — the client ID is not secret.
	DefaultClientID = "oljO5HmcOWvCZ5wbitpXPXk3u0LjAb5GdAEBBU5hwKA"

	DefaultAuthorizeURL = "https://app.gumroad.com/oauth/authorize"
	DefaultTokenURL     = "https://app.gumroad.com/oauth/token" //nolint:gosec // G101: not a credential

	Scopes = "edit_products view_sales mark_sales_as_shipped edit_sales view_payouts view_profile account"

	DefaultTimeout = 2 * time.Minute

	EnvClientID     = "GUMROAD_OAUTH_CLIENT_ID"
	EnvAuthorizeURL = "GUMROAD_OAUTH_AUTHORIZE_URL"
	EnvTokenURL     = "GUMROAD_OAUTH_TOKEN_URL" //nolint:gosec // G101: env var name, not a credential
)

func ClientID() string {
	if v := os.Getenv(EnvClientID); v != "" {
		return v
	}
	return DefaultClientID
}

func AuthorizeURL() string {
	if v := os.Getenv(EnvAuthorizeURL); v != "" {
		return v
	}
	return DefaultAuthorizeURL
}

func TokenURL() string {
	if v := os.Getenv(EnvTokenURL); v != "" {
		return v
	}
	return DefaultTokenURL
}
