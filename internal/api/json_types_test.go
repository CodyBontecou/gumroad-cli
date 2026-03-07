package api

import (
	"encoding/json"
	"strconv"
	"strings"
	"testing"
)

func assertJSONIntValue(t *testing.T, input string, initial, want JSONInt) {
	t.Helper()

	got := initial
	if err := json.Unmarshal([]byte(input), &got); err != nil {
		t.Fatalf("unmarshal %q: %v", input, err)
	}
	if got != want {
		t.Fatalf("input %q: got %d, want %d", input, got, want)
	}
}

func assertJSONIntErrorContains(t *testing.T, input, want string) {
	t.Helper()

	var got JSONInt
	err := json.Unmarshal([]byte(input), &got)
	if err == nil {
		t.Fatalf("expected error for %q", input)
	}
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("unexpected error for %q: %v", input, err)
	}
}

func TestJSONIntAcceptsIntegerLikeNumbers(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		initial JSONInt
		want    JSONInt
	}{
		{name: "int", input: "7", want: 7},
		{name: "float", input: "7.0", want: 7},
		{name: "null", input: "null", initial: 9, want: 0},
		{name: "exponent", input: "12e2", want: 1200},
		{name: "float exponent", input: "12.0e2", want: 1200},
		{name: "uppercase exponent", input: "12E2", want: 1200},
		{name: "negative exponent result", input: "-12e2", want: -1200},
		{name: "negative zero", input: "-0.0e3", initial: 9, want: 0},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assertJSONIntValue(t, test.input, test.initial, test.want)
		})
	}
}

func TestJSONIntRejectsFractionalNumbers(t *testing.T) {
	for _, input := range []string{"7.5", "12e-1", "12.05e1"} {
		assertJSONIntErrorContains(t, input, "expected integer JSON number")
	}
}

func TestJSONIntPreservesLargeIntegerPrecision(t *testing.T) {
	if strconv.IntSize < 64 {
		t.Skip("precision check requires 64-bit ints")
	}

	var got JSONInt
	if err := json.Unmarshal([]byte("9007199254740993"), &got); err != nil {
		t.Fatalf("unmarshal large int: %v", err)
	}
	if got != JSONInt(9007199254740993) {
		t.Fatalf("got %d, want %d", got, JSONInt(9007199254740993))
	}
}

func TestJSONIntRejectsOutOfRangeValues(t *testing.T) {
	tooBig := "9223372036854775808"
	if strconv.IntSize < 64 {
		tooBig = "2147483648"
	}

	assertJSONIntErrorContains(t, tooBig, "out of range")
}

func TestNormalizeJSONInt(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr string
	}{
		{name: "trimmed integral decimal", input: "120.00", want: "120"},
		{name: "positive exponent", input: "12.0E2", want: "1200"},
		{name: "fractional result rejected", input: "12.05e1", wantErr: "expected integer JSON number"},
		{name: "overflow from exponent growth", input: "9e999", wantErr: "out of range"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := normalizeJSONInt(test.input)
			if test.wantErr != "" {
				if err == nil {
					t.Fatal("expected error")
				}
				if !strings.Contains(err.Error(), test.wantErr) {
					t.Fatalf("got err=%v, want substring %q", err, test.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != test.want {
				t.Fatalf("got %q, want %q", got, test.want)
			}
		})
	}
}

func TestParseJSONNumberParts(t *testing.T) {
	tests := []struct {
		input string
		ok    bool
	}{
		{input: "12.0E2", ok: true},
		{input: "-12e2", ok: true},
		{input: "", ok: false},
		{input: "-", ok: false},
		{input: "12e", ok: false},
		{input: "12.", ok: false},
		{input: ".12", ok: false},
		{input: "12.a", ok: false},
	}

	for _, test := range tests {
		_, ok := parseJSONNumberParts(test.input)
		if ok != test.ok {
			t.Fatalf("input %q: got ok=%v, want %v", test.input, ok, test.ok)
		}
	}
}

func TestDigitHelpers(t *testing.T) {
	if !isASCIIDigits("123456") {
		t.Fatal("expected ASCII digits to be accepted")
	}
	if isASCIIDigits("12a") {
		t.Fatal("did not expect mixed digits to be accepted")
	}
	if isASCIIDigits("") {
		t.Fatal("did not expect empty string to be accepted")
	}

	if got := trimLeadingZeros("000123"); got != "123" {
		t.Fatalf("got %q, want %q", got, "123")
	}
	if got := trimLeadingZeros("0000"); got != "" {
		t.Fatalf("got %q, want empty string", got)
	}
}
