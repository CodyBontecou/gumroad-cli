package cmdutil

import (
	"os"
	"runtime"
	"strings"
)

type CommandArg struct {
	Flag    string
	Value   string
	Boolean bool // true for valueless flags like --no-upcoming
}

var hintGOOS = runtime.GOOS
var hintGetenv = os.Getenv

// ReplayCommand formats a copy-pasteable command hint that preserves active flags.
// Valued args are included when Value is non-empty. Boolean args are included when Boolean is true.
func ReplayCommand(base string, args ...CommandArg) string {
	parts := []string{base}
	quote := commandQuoter()
	for _, arg := range args {
		if arg.Boolean {
			parts = append(parts, arg.Flag)
			continue
		}
		if arg.Value == "" {
			continue
		}

		parts = append(parts, arg.Flag)
		parts = append(parts, quote(arg.Value))
	}
	return strings.Join(parts, " ")
}

func commandQuoter() func(string) string {
	if hintGOOS != "windows" {
		return shellQuotePOSIX
	}
	if looksLikePowerShell() {
		return shellQuotePowerShell
	}
	if looksLikePOSIXShell() {
		return shellQuotePOSIX
	}
	return shellQuoteCmd
}

func looksLikePOSIXShell() bool {
	shell := strings.ToLower(hintGetenv("SHELL"))
	return shell != "" || hintGetenv("MSYSTEM") != ""
}

func looksLikePowerShell() bool {
	shell := strings.ToLower(hintGetenv("SHELL"))
	if strings.Contains(shell, "pwsh") || strings.Contains(shell, "powershell") {
		return true
	}
	return hintGetenv("PSModulePath") != ""
}

func shellQuotePOSIX(value string) string {
	if strings.ContainsAny(value, " \t\n\"'`$&;|<>*?()[]{}!") {
		return "'" + strings.ReplaceAll(value, "'", `'"'"'`) + "'"
	}
	return value
}

func shellQuotePowerShell(value string) string {
	if strings.ContainsAny(value, " \t\n\"'`$&;|<>*?()[]{}!") {
		return "'" + strings.ReplaceAll(value, "'", "''") + "'"
	}
	return value
}

func shellQuoteCmd(value string) string {
	if strings.ContainsAny(value, " \t\n\"&|<>^") {
		return `"` + strings.ReplaceAll(value, `"`, `""`) + `"`
	}
	return value
}
