package output

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

const (
	injectionMarker     = "[redacted-injection]"
	firstPrintableASCII = 0x20
	deleteASCII         = 0x7f
	firstC1Control      = 0x80
	lastC1Control       = 0x9f
)

var bidiOverrides = map[rune]struct{}{
	0x202a: {}, 0x202b: {}, 0x202c: {}, 0x202d: {}, 0x202e: {},
	0x2066: {}, 0x2067: {}, 0x2068: {}, 0x2069: {},
}

var (
	ansiEscapeRE     = regexp.MustCompile(`\x1b\[[0-9;?]*[A-Za-z]`)
	injectionPhrases = []*regexp.Regexp{
		regexp.MustCompile(`(?i)ignore (?:all )?previous instructions?`),
		regexp.MustCompile(`(?i)disregard (?:all )?previous instructions?`),
		regexp.MustCompile(`(?i)you are now (?:a |an )?[a-z]+ assistant`),
	}
)

func SanitizeJSONBytes(data []byte) ([]byte, error) {
	var v any
	if err := json.Unmarshal(data, &v); err != nil {
		return nil, fmt.Errorf("could not parse JSON to sanitize: %w", err)
	}
	cleaned := sanitizeValue(v)
	out, err := json.Marshal(cleaned)
	if err != nil {
		return nil, fmt.Errorf("could not re-encode sanitized JSON: %w", err)
	}
	return out, nil
}

func sanitizeValue(v any) any {
	switch x := v.(type) {
	case string:
		return sanitizeString(x)
	case map[string]any:
		out := make(map[string]any, len(x))
		for k, val := range x {
			out[k] = sanitizeValue(val)
		}
		return out
	case []any:
		out := make([]any, len(x))
		for i, item := range x {
			out[i] = sanitizeValue(item)
		}
		return out
	default:
		return v
	}
}

func sanitizeString(s string) string {
	s = ansiEscapeRE.ReplaceAllString(s, "")
	s = stripDisallowedControlChars(s)
	for _, re := range injectionPhrases {
		s = re.ReplaceAllString(s, injectionMarker)
	}
	return s
}

func stripDisallowedControlChars(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		if r == '\t' || r == '\n' || r == '\r' {
			b.WriteRune(r)
			continue
		}
		if r < firstPrintableASCII || r == deleteASCII {
			continue
		}
		if r >= firstC1Control && r <= lastC1Control {
			continue
		}
		if _, isBidi := bidiOverrides[r]; isBidi {
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}
