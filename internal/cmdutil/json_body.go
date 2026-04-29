package cmdutil

import (
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
)

func RejectFlagsWithJSONBody(cmd *cobra.Command, bodyFlags ...string) error {
	if !cmd.Flags().Changed("json-body") {
		return nil
	}
	var conflicting []string
	for _, name := range bodyFlags {
		if cmd.Flags().Changed(name) {
			conflicting = append(conflicting, "--"+name)
		}
	}
	if len(conflicting) == 0 {
		return nil
	}
	return UsageErrorf(cmd, "--json-body cannot be combined with %s", strings.Join(conflicting, ", "))
}

func ParseJSONBody(raw string, stdin io.Reader) (url.Values, error) {
	body, err := readJSONBody(raw, stdin)
	if err != nil {
		return nil, err
	}
	return convertJSONObject(body)
}

func readJSONBody(raw string, stdin io.Reader) ([]byte, error) {
	if raw != "-" {
		return []byte(raw), nil
	}
	if stdin == nil {
		return nil, fmt.Errorf("--json-body=- requires stdin to be available")
	}
	data, err := io.ReadAll(stdin)
	if err != nil {
		return nil, fmt.Errorf("could not read --json-body from stdin: %w", err)
	}
	return data, nil
}

func convertJSONObject(data []byte) (url.Values, error) {
	dec := json.NewDecoder(strings.NewReader(string(data)))
	dec.UseNumber()
	var raw any
	if err := dec.Decode(&raw); err != nil {
		return nil, fmt.Errorf("invalid --json-body: %w", err)
	}

	obj, ok := raw.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("--json-body must be a JSON object")
	}

	values := url.Values{}
	for key, val := range obj {
		if err := setJSONValue(values, key, val); err != nil {
			return nil, err
		}
	}
	return values, nil
}

func setJSONValue(values url.Values, key string, val any) error {
	switch v := val.(type) {
	case nil:
		return nil
	case string:
		values.Set(key, v)
	case bool:
		values.Set(key, strconv.FormatBool(v))
	case json.Number:
		values.Set(key, v.String())
	case []any:
		bracketKey := key + "[]"
		for i, item := range v {
			s, err := scalarToString(item)
			if err != nil {
				return fmt.Errorf("--json-body: %s[%d]: %w", key, i, err)
			}
			values.Add(bracketKey, s)
		}
	case map[string]any:
		return fmt.Errorf("--json-body: nested object at %q not supported; use bracket notation in keys", key)
	default:
		return fmt.Errorf("--json-body: unsupported value type for %q", key)
	}
	return nil
}

func scalarToString(v any) (string, error) {
	switch s := v.(type) {
	case nil:
		return "", fmt.Errorf("null is not allowed inside an array")
	case string:
		return s, nil
	case bool:
		return strconv.FormatBool(s), nil
	case json.Number:
		return s.String(), nil
	default:
		return "", fmt.Errorf("array elements must be string, number, or boolean")
	}
}
