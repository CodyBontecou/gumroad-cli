package cmdutil

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

const (
	firstPrintableASCII = 0x20
	deleteASCII         = 0x7f
)

func ValidateResourceID(id string) error {
	if id == "" {
		return fmt.Errorf("resource ID cannot be empty")
	}
	if id == "." || id == ".." {
		return fmt.Errorf("resource ID %q is a path traversal segment", id)
	}
	for i := 0; i < len(id); i++ {
		c := id[i]
		if c < firstPrintableASCII || c == deleteASCII {
			return fmt.Errorf("resource ID contains control character at position %d", i)
		}
		switch c {
		case '?', '#', '%', '/', '\\', ' ':
			return fmt.Errorf("resource ID contains forbidden character %q", string(c))
		}
	}
	return nil
}

func SafeIDArgs(n int) cobra.PositionalArgs {
	exact := ExactArgs(n)
	return func(cmd *cobra.Command, args []string) error {
		if err := exact(cmd, args); err != nil {
			return err
		}
		for _, arg := range args {
			if err := ValidateResourceID(arg); err != nil {
				return UsageErrorf(cmd, "invalid resource ID: %s", strings.TrimSpace(err.Error()))
			}
		}
		return nil
	}
}

func RequireSafeIDFlag(cmd *cobra.Command, flag, value string) error {
	if !cmd.Flags().Changed(flag) {
		return nil
	}
	if err := ValidateResourceID(value); err != nil {
		return UsageErrorf(cmd, "--%s: %s", flag, strings.TrimSpace(err.Error()))
	}
	return nil
}
