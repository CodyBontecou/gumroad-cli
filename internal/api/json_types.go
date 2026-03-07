package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

// JSONInt is a Gumroad API compatibility shim for integer fields that may
// arrive as 1, 1.0, null, or be omitted.
//
// Keep this type at the JSON decode boundary only. It exists to shield the CLI
// from upstream schema inconsistencies and should be easy to delete if the API
// becomes consistently typed.
type JSONInt int

const maxInt64Base10Digits = 19

var (
	maxPlatformInt = int64(^uint(0) >> 1)
	minPlatformInt = -maxPlatformInt - 1
)

func (n *JSONInt) UnmarshalJSON(data []byte) error {
	trimmed := bytes.TrimSpace(data)
	if bytes.Equal(trimmed, []byte("null")) {
		*n = 0
		return nil
	}

	var v json.Number
	if err := json.Unmarshal(trimmed, &v); err != nil {
		return err
	}

	normalized, err := normalizeJSONInt(v.String())
	if err != nil {
		return err
	}

	intValue, err := strconv.ParseInt(normalized, 10, 64)
	if err != nil {
		return fmt.Errorf("integer JSON number out of range: %s", data)
	}

	if intValue < minPlatformInt || intValue > maxPlatformInt {
		return fmt.Errorf("integer JSON number out of range: %s", data)
	}

	*n = JSONInt(intValue)
	return nil
}

type jsonNumberParts struct {
	negative bool
	integer  string
	fraction string
	exponent int
}

func normalizeJSONInt(value string) (string, error) {
	parts, ok := parseJSONNumberParts(value)
	if !ok {
		return "", fmt.Errorf("expected integer JSON number, got %s", value)
	}

	digits := parts.integer + parts.fraction
	if allZeros(digits) {
		return "0", nil
	}

	decimalDigits := len(parts.fraction) - parts.exponent
	switch {
	case decimalDigits > 0:
		if decimalDigits > len(digits) || !allZeros(digits[len(digits)-decimalDigits:]) {
			return "", fmt.Errorf("expected integer JSON number, got %s", value)
		}
		digits = digits[:len(digits)-decimalDigits]
	case decimalDigits < 0:
		extraZeros := -decimalDigits
		if len(trimLeadingZeros(digits))+extraZeros > maxInt64Base10Digits {
			return "", fmt.Errorf("integer JSON number out of range: %s", value)
		}
		digits += strings.Repeat("0", extraZeros)
	}

	digits = trimLeadingZeros(digits)
	if digits == "" {
		return "0", nil
	}
	if len(digits) > maxInt64Base10Digits {
		return "", fmt.Errorf("integer JSON number out of range: %s", value)
	}
	if parts.negative {
		return "-" + digits, nil
	}
	return digits, nil
}

func parseJSONNumberParts(value string) (jsonNumberParts, bool) {
	var parts jsonNumberParts
	if value == "" {
		return parts, false
	}

	if strings.HasPrefix(value, "-") {
		parts.negative = true
		value = value[1:]
	}
	if value == "" {
		return parts, false
	}

	mantissa, exponentPart, hasExponent := strings.Cut(value, "e")
	if !hasExponent {
		mantissa, exponentPart, hasExponent = strings.Cut(value, "E")
	}
	if hasExponent {
		if exponentPart == "" {
			return parts, false
		}
		exponent, err := strconv.Atoi(exponentPart)
		if err != nil {
			return parts, false
		}
		parts.exponent = exponent
	}

	integer, fraction, hasFraction := strings.Cut(mantissa, ".")
	if integer == "" || !isASCIIDigits(integer) {
		return parts, false
	}
	if hasFraction {
		if fraction == "" || !isASCIIDigits(fraction) {
			return parts, false
		}
	}

	parts.integer = integer
	if hasFraction {
		parts.fraction = fraction
	}

	return parts, true
}

func isASCIIDigits(value string) bool {
	for i := 0; i < len(value); i++ {
		if value[i] < '0' || value[i] > '9' {
			return false
		}
	}
	return value != ""
}

func allZeros(value string) bool {
	for i := 0; i < len(value); i++ {
		if value[i] != '0' {
			return false
		}
	}
	return value != ""
}

func trimLeadingZeros(value string) string {
	for i := 0; i < len(value); i++ {
		if value[i] != '0' {
			return value[i:]
		}
	}
	return ""
}
