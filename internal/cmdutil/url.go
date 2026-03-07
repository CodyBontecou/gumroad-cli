package cmdutil

import (
	"net/url"
	"strings"
)

// CloneValues copies URL values so callers can safely mutate request params.
func CloneValues(values url.Values) url.Values {
	cloned := make(url.Values, len(values))
	for key, current := range values {
		cloned[key] = append([]string(nil), current...)
	}
	return cloned
}

// JoinPath escapes path segments and joins them into an absolute API path.
func JoinPath(segments ...string) string {
	escaped := make([]string, len(segments))
	for i, segment := range segments {
		escaped[i] = url.PathEscape(segment)
	}
	return "/" + strings.Join(escaped, "/")
}
