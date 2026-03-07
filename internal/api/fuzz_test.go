package api

import (
	"encoding/json"
	"errors"
	"math"
	"testing"
)

func FuzzJSONIntUnmarshal(f *testing.F) {
	for _, seed := range []string{"0", "7", "7.0", "7.5", "null", `{}`, `[]`, `"x"`, "", "18446744073709551616"} {
		f.Add(seed)
	}

	maxInt := int(^uint(0) >> 1)
	minInt := -maxInt - 1

	f.Fuzz(func(t *testing.T, input string) {
		var got JSONInt
		err := json.Unmarshal([]byte(input), &got)
		if err != nil {
			return
		}

		var decoded any
		if err := json.Unmarshal([]byte(input), &decoded); err != nil {
			t.Fatalf("JSONInt accepted input that generic JSON rejected: %q (%v)", input, err)
		}

		switch v := decoded.(type) {
		case nil:
			if got != 0 {
				t.Fatalf("null should decode to 0, got %d", got)
			}
		case float64:
			if math.Trunc(v) != v {
				t.Fatalf("accepted non-integer number %v for input %q", v, input)
			}

			if v >= float64(minInt) && v <= float64(maxInt) && got != JSONInt(v) {
				t.Fatalf("got %d, want %d for input %q", got, JSONInt(v), input)
			}
		default:
			t.Fatalf("unexpected successful decode type %T for input %q", decoded, input)
		}
	})
}

func FuzzParseAPIError(f *testing.F) {
	f.Add(401, []byte(`{"success":false}`))
	f.Add(403, []byte(`{"success":false,"message":"Insufficient scope"}`))
	f.Add(404, []byte(`{"success":false,"message":"Missing"}`))
	f.Add(500, []byte(`not json`))

	f.Fuzz(func(t *testing.T, status int, body []byte) {
		if status < 0 {
			status = -status
		}
		status %= 1000

		err := parseAPIError(status, body)
		var apiErr *APIError
		if !errors.As(err, &apiErr) {
			t.Fatalf("expected *APIError, got %T", err)
		}
		if apiErr.StatusCode != status {
			t.Fatalf("got status %d, want %d", apiErr.StatusCode, status)
		}
		if apiErr.Message == "" {
			t.Fatalf("expected non-empty message for status=%d body=%q", status, string(body))
		}
	})
}
