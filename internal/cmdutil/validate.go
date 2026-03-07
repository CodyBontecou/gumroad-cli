package cmdutil

import (
	"net/url"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

// RequirePositiveIntFlag rejects explicitly provided integer flags that must be > 0.
func RequirePositiveIntFlag(cmd *cobra.Command, flag string, value int) error {
	if cmd.Flags().Changed(flag) && value <= 0 {
		return UsageErrorf(cmd, "--%s must be greater than 0", flag)
	}
	return nil
}

// RequirePercentFlag rejects explicitly provided percentage flags that must be
// between 1 and 100 inclusive.
func RequirePercentFlag(cmd *cobra.Command, flag string, value int) error {
	if !cmd.Flags().Changed(flag) {
		return nil
	}
	if value < 1 || value > 100 {
		return UsageErrorf(cmd, "--%s must be between 1 and 100", flag)
	}
	return nil
}

// RequireNonNegativeIntFlag rejects explicitly provided integer flags that must be >= 0.
func RequireNonNegativeIntFlag(cmd *cobra.Command, flag string, value int) error {
	if cmd.Flags().Changed(flag) && value < 0 {
		return UsageErrorf(cmd, "--%s cannot be negative", flag)
	}
	return nil
}

// RequireNonNegativeDurationFlag rejects explicitly provided duration flags
// that must be >= 0.
func RequireNonNegativeDurationFlag(cmd *cobra.Command, flag string, value time.Duration) error {
	if cmd.Flags().Changed(flag) && value < 0 {
		return UsageErrorf(cmd, "--%s cannot be negative", flag)
	}
	return nil
}

// RequireDateFlag rejects explicitly provided date flags that are not strict
// YYYY-MM-DD values.
func RequireDateFlag(cmd *cobra.Command, flag, value string) error {
	if !cmd.Flags().Changed(flag) {
		return nil
	}
	if _, err := time.Parse("2006-01-02", value); err != nil {
		return UsageErrorf(cmd, "--%s must be a valid date in YYYY-MM-DD format", flag)
	}
	return nil
}

// RequireHTTPURLFlag rejects explicitly provided URL flags that are not
// absolute http(s) URLs.
func RequireHTTPURLFlag(cmd *cobra.Command, flag, value string) error {
	if !cmd.Flags().Changed(flag) {
		return nil
	}

	parsed, err := url.ParseRequestURI(value)
	if err != nil || parsed == nil || parsed.Host == "" {
		return UsageErrorf(cmd, "--%s must be a valid absolute URL", flag)
	}

	switch strings.ToLower(parsed.Scheme) {
	case "http", "https":
		return nil
	default:
		return UsageErrorf(cmd, "--%s must use http or https", flag)
	}
}
