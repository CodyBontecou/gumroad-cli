package buy

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/antiwork/gumroad-cli/internal/cmdutil"
	"github.com/antiwork/gumroad-cli/internal/testutil"
)

func decodeOrderBody(t *testing.T, r *http.Request) map[string]any {
	t.Helper()
	raw, err := io.ReadAll(r.Body)
	if err != nil {
		t.Fatalf("read body: %v", err)
	}
	var body map[string]any
	if err := json.Unmarshal(raw, &body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	return body
}

func rejectGET(t *testing.T) http.HandlerFunc {
	t.Helper()
	return func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("unexpected GET %s — buy must POST /orders directly without any pre-lookup", r.URL.Path)
		http.Error(w, "no GETs allowed in buy tests", http.StatusInternalServerError)
	}
}

func ordersHandler(t *testing.T, body map[string]any) http.HandlerFunc {
	t.Helper()
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			rejectGET(t)(w, r)
			return
		}
		if r.URL.Path != "/orders" {
			t.Errorf("unexpected request to %s; want /orders", r.URL.Path)
			http.NotFound(w, r)
			return
		}
		testutil.JSON(t, w, body)
	}
}

func TestBuy_RequiresPermalinkArg(t *testing.T) {
	cmd := testutil.Command(NewBuyCmd())
	cmd.SetArgs([]string{})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error when permalink missing")
	}
	if !strings.Contains(err.Error(), "missing required argument") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestBuy_RequiresPaymentMethod(t *testing.T) {
	cmd := testutil.Command(NewBuyCmd())
	cmd.SetArgs([]string{"abc123", "--price-cents", "500"})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error when --payment-method-id missing")
	}
	if !strings.Contains(err.Error(), "--payment-method-id") {
		t.Fatalf("unexpected error: %v", err)
	}
	var usageErr *cmdutil.UsageError
	if !errors.As(err, &usageErr) {
		t.Fatalf("expected *cmdutil.UsageError, got %T", err)
	}
}

func TestBuy_RequiresPriceCents(t *testing.T) {
	cmd := testutil.Command(NewBuyCmd())
	cmd.SetArgs([]string{"abc123", "--pm", "pm_card_visa"})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error when --price-cents missing")
	}
	if !strings.Contains(err.Error(), "--price-cents") {
		t.Fatalf("unexpected error: %v", err)
	}
	var usageErr *cmdutil.UsageError
	if !errors.As(err, &usageErr) {
		t.Fatalf("expected *cmdutil.UsageError, got %T", err)
	}
}

func TestBuy_DryRunPrintsBodyAndSkipsPost(t *testing.T) {
	var sawAnyRequest atomic.Bool
	testutil.Setup(t, func(w http.ResponseWriter, r *http.Request) {
		sawAnyRequest.Store(true)
		t.Errorf("dry-run made unexpected %s %s — must not touch the network", r.Method, r.URL.Path)
		http.Error(w, "dry-run should not call the API", http.StatusInternalServerError)
	})

	cmd := testutil.Command(NewBuyCmd(), testutil.DryRun(true), testutil.Quiet(false))
	cmd.SetArgs([]string{"abc123", "--pm", "pm_card_visa", "--price-cents", "500"})
	out := testutil.CaptureStdout(func() { testutil.MustExecute(t, cmd) })

	if sawAnyRequest.Load() {
		t.Fatal("dry-run must not make any HTTP request")
	}
	for _, want := range []string{"Dry run", "POST /orders", `"permalink": "abc123"`, `"perceived_price_cents": 500`, `"stripe_payment_method_id": "pm_card_visa"`} {
		if !strings.Contains(out, want) {
			t.Fatalf("dry-run output missing %q in %q", want, out)
		}
	}
}

func TestBuy_DryRun_DoesNotRequireConfig(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("GUMROAD_ACCESS_TOKEN", "")
	t.Setenv("GUMROAD_API_BASE_URL", "http://127.0.0.1:1/v2")

	cmd := testutil.Command(NewBuyCmd(), testutil.DryRun(true))
	cmd.SetArgs([]string{"abc123", "--pm", "pm_card_visa", "--price-cents", "500"})
	out := testutil.CaptureStdout(func() { testutil.MustExecute(t, cmd) })

	if !strings.Contains(out, "POST /orders") {
		t.Fatalf("dry-run output missing POST /orders; got %q", out)
	}
}

func TestBuy_HappyPath(t *testing.T) {
	var gotMethod, gotPath, gotContentType string
	var gotBody map[string]any
	testutil.Setup(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			rejectGET(t)(w, r)
			return
		}
		gotMethod = r.Method
		gotPath = r.URL.Path
		gotContentType = r.Header.Get("Content-Type")
		gotBody = decodeOrderBody(t, r)
		testutil.JSON(t, w, map[string]any{
			"line_items": map[string]any{
				"li-0": map[string]any{
					"uid":         "li-0",
					"success":     true,
					"name":        "Art Pack",
					"permalink":   "abc123",
					"content_url": "https://gumroad.com/library/abc123",
				},
			},
		})
	})

	cmd := testutil.Command(NewBuyCmd(), testutil.Yes(true), testutil.Quiet(false))
	cmd.SetArgs([]string{"abc123", "--pm", "pm_card_visa", "--price-cents", "500"})
	out := testutil.CaptureStdout(func() { testutil.MustExecute(t, cmd) })

	if gotMethod != "POST" || gotPath != "/orders" {
		t.Fatalf("got %s %s, want POST /orders", gotMethod, gotPath)
	}
	if !strings.HasPrefix(gotContentType, "application/json") {
		t.Fatalf("got Content-Type %q, want application/json", gotContentType)
	}
	if gotBody["stripe_payment_method_id"] != "pm_card_visa" {
		t.Fatalf("got pm %v, want pm_card_visa", gotBody["stripe_payment_method_id"])
	}
	items, ok := gotBody["line_items"].([]any)
	if !ok || len(items) != 1 {
		t.Fatalf("expected 1 line item, got %v", gotBody["line_items"])
	}
	item := items[0].(map[string]any)
	if item["permalink"] != "abc123" {
		t.Fatalf("got permalink %v, want abc123", item["permalink"])
	}
	if int(item["perceived_price_cents"].(float64)) != 500 {
		t.Fatalf("got perceived_price_cents %v, want 500", item["perceived_price_cents"])
	}
	if int(item["quantity"].(float64)) != 1 {
		t.Fatalf("got quantity %v, want 1", item["quantity"])
	}
	if item["uid"] != "li-0" {
		t.Fatalf("got uid %v, want li-0", item["uid"])
	}
	for _, want := range []string{"Purchased Art Pack", "https://gumroad.com/library/abc123"} {
		if !strings.Contains(out, want) {
			t.Fatalf("output missing %q in %q", want, out)
		}
	}
}

func TestBuy_RequiresConfirmation(t *testing.T) {
	testutil.Setup(t, func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("should not POST /orders without confirmation; got %s %s", r.Method, r.URL.Path)
		http.Error(w, "no API call expected", http.StatusInternalServerError)
	})

	cmd := testutil.Command(NewBuyCmd(), testutil.NoInput(true))
	cmd.SetArgs([]string{"abc123", "--pm", "pm_card_visa", "--price-cents", "500"})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error without --yes and --no-input")
	}
}

func TestBuy_CardDeclined(t *testing.T) {
	testutil.Setup(t, ordersHandler(t, map[string]any{
		"line_items": map[string]any{
			"li-0": map[string]any{
				"uid":           "li-0",
				"success":       false,
				"error_message": "Your card was declined.",
			},
		},
	}))

	var stderr strings.Builder
	cmd := testutil.Command(NewBuyCmd(), testutil.Yes(true), testutil.Quiet(false), testutil.Stderr(&stderr))
	cmd.SetArgs([]string{"abc123", "--pm", "pm_card_visa", "--price-cents", "500"})
	err := cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "Your card was declined") {
		t.Fatalf("expected decline error, got %v", err)
	}
	if !strings.Contains(stderr.String(), "Your card was declined") {
		t.Fatalf("expected decline message on stderr, got %q", stderr.String())
	}
}

func TestBuy_RequiresAction3DS(t *testing.T) {
	testutil.Setup(t, ordersHandler(t, map[string]any{
		"line_items": map[string]any{
			"li-0": map[string]any{
				"uid":              "li-0",
				"requires_action":  true,
				"client_secret":    "pi_xxx_secret_yyy",
				"confirmation_url": "https://gumroad.com/l/abc123",
			},
		},
	}))

	var stderr strings.Builder
	cmd := testutil.Command(NewBuyCmd(), testutil.Yes(true), testutil.Quiet(false), testutil.Stderr(&stderr))
	cmd.SetArgs([]string{"abc123", "--pm", "pm_card_visa", "--price-cents", "500"})
	err := cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "3DS") {
		t.Fatalf("expected 3DS error, got %v", err)
	}
	for _, want := range []string{"3DS verification required", "https://gumroad.com/l/abc123"} {
		if !strings.Contains(stderr.String(), want) {
			t.Fatalf("expected %q on stderr, got %q", want, stderr.String())
		}
	}
}

func TestBuy_JSONOutputPreservesEnvelope(t *testing.T) {
	testutil.Setup(t, ordersHandler(t, map[string]any{
		"line_items": map[string]any{
			"li-0": map[string]any{
				"uid":         "li-0",
				"success":     true,
				"name":        "Art Pack",
				"permalink":   "abc123",
				"content_url": "https://gumroad.com/library/abc123",
			},
		},
	}))

	cmd := testutil.Command(NewBuyCmd(), testutil.Yes(true), testutil.JSONOutput())
	cmd.SetArgs([]string{"abc123", "--pm", "pm_card_visa", "--price-cents", "500"})
	out := testutil.CaptureStdout(func() { testutil.MustExecute(t, cmd) })

	var resp map[string]any
	if err := json.Unmarshal([]byte(out), &resp); err != nil {
		t.Fatalf("output is not valid JSON: %v\n%s", err, out)
	}
	items, ok := resp["line_items"].(map[string]any)
	if !ok {
		t.Fatalf("expected line_items in JSON envelope, got %v", resp)
	}
	li, ok := items["li-0"].(map[string]any)
	if !ok {
		t.Fatalf("expected li-0 in JSON envelope, got %v", items)
	}
	if li["content_url"] != "https://gumroad.com/library/abc123" {
		t.Fatalf("got content_url %v, want https://gumroad.com/library/abc123", li["content_url"])
	}
}

func TestBuy_PWYWSendsPriceVerbatim(t *testing.T) {
	var gotBody map[string]any
	testutil.Setup(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			rejectGET(t)(w, r)
			return
		}
		gotBody = decodeOrderBody(t, r)
		testutil.JSON(t, w, map[string]any{
			"line_items": map[string]any{
				"li-0": map[string]any{
					"uid":         "li-0",
					"success":     true,
					"name":        "PWYW Pack",
					"permalink":   "pwyw1",
					"content_url": "https://gumroad.com/library/pwyw1",
				},
			},
		})
	})

	cmd := testutil.Command(NewBuyCmd(), testutil.Yes(true), testutil.Quiet(true))
	cmd.SetArgs([]string{"pwyw1", "--pm", "pm_card_visa", "--price-cents", "1234"})
	testutil.MustExecute(t, cmd)

	items := gotBody["line_items"].([]any)
	item := items[0].(map[string]any)
	if int(item["perceived_price_cents"].(float64)) != 1234 {
		t.Fatalf("got perceived_price_cents %v, want 1234 (CLI must send the flag verbatim)", item["perceived_price_cents"])
	}
}

func TestBuy_PermalinkNotFound(t *testing.T) {
	testutil.Setup(t, ordersHandler(t, map[string]any{
		"line_items": map[string]any{
			"li-0": map[string]any{
				"uid":           "li-0",
				"success":       false,
				"error_message": "Product missing does not exist",
			},
		},
	}))

	var stderr strings.Builder
	cmd := testutil.Command(NewBuyCmd(), testutil.Yes(true), testutil.Stderr(&stderr))
	cmd.SetArgs([]string{"missing", "--pm", "pm_card_visa", "--price-cents", "500"})
	err := cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "Product missing does not exist") {
		t.Fatalf("expected unknown-permalink error, got %v", err)
	}
	if !strings.Contains(stderr.String(), "Product missing does not exist") {
		t.Fatalf("expected error on stderr, got %q", stderr.String())
	}
}

func TestBuy_OptionalFieldsSentWhenSet(t *testing.T) {
	var gotBody map[string]any
	testutil.Setup(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			rejectGET(t)(w, r)
			return
		}
		gotBody = decodeOrderBody(t, r)
		testutil.JSON(t, w, map[string]any{
			"line_items": map[string]any{
				"li-0": map[string]any{"uid": "li-0", "success": true, "name": "Art Pack", "permalink": "abc123"},
			},
		})
	})

	cmd := testutil.Command(NewBuyCmd(), testutil.Yes(true), testutil.Quiet(true))
	cmd.SetArgs([]string{
		"abc123",
		"--pm", "pm_card_visa",
		"--price-cents", "500",
		"--customer-id", "cus_xyz",
		"--email", "buyer@example.com",
		"--quantity", "2",
		"--variant", "var_id",
		"--offer-code", "SAVE10",
		"--tip-cents", "100",
	})
	testutil.MustExecute(t, cmd)

	if gotBody["stripe_customer_id"] != "cus_xyz" {
		t.Errorf("missing stripe_customer_id, got %v", gotBody["stripe_customer_id"])
	}
	if gotBody["email"] != "buyer@example.com" {
		t.Errorf("missing email, got %v", gotBody["email"])
	}
	item := gotBody["line_items"].([]any)[0].(map[string]any)
	if int(item["quantity"].(float64)) != 2 {
		t.Errorf("got quantity %v, want 2", item["quantity"])
	}
	variants, ok := item["variants"].([]any)
	if !ok || len(variants) != 1 || variants[0] != "var_id" {
		t.Errorf("got variants %v, want [var_id]", item["variants"])
	}
	if item["discount_code"] != "SAVE10" {
		t.Errorf("got discount_code %v, want SAVE10", item["discount_code"])
	}
	if int(item["tip_cents"].(float64)) != 100 {
		t.Errorf("got tip_cents %v, want 100", item["tip_cents"])
	}
}

func TestBuy_DryRun_JSONOutput(t *testing.T) {
	testutil.Setup(t, func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("dry-run should not POST; got %s %s", r.Method, r.URL.Path)
		http.Error(w, "no API call expected", http.StatusInternalServerError)
	})

	cmd := testutil.Command(NewBuyCmd(), testutil.DryRun(true), testutil.JSONOutput())
	cmd.SetArgs([]string{"abc123", "--pm", "pm_card_visa", "--price-cents", "500"})
	out := testutil.CaptureStdout(func() { testutil.MustExecute(t, cmd) })

	var payload struct {
		DryRun  bool `json:"dry_run"`
		Request struct {
			Method string         `json:"method"`
			Path   string         `json:"path"`
			Body   map[string]any `json:"body"`
		} `json:"request"`
	}
	if err := json.Unmarshal([]byte(out), &payload); err != nil {
		t.Fatalf("not valid JSON: %v\n%s", err, out)
	}
	if !payload.DryRun || payload.Request.Method != "POST" || payload.Request.Path != "/orders" {
		t.Fatalf("unexpected dry-run JSON envelope: %+v", payload)
	}
	if payload.Request.Body["stripe_payment_method_id"] != "pm_card_visa" {
		t.Fatalf("dry-run body missing payment method, got %v", payload.Request.Body)
	}
}

func TestBuy_DryRun_PlainOutput(t *testing.T) {
	testutil.Setup(t, func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("dry-run should not POST; got %s %s", r.Method, r.URL.Path)
		http.Error(w, "no API call expected", http.StatusInternalServerError)
	})

	cmd := testutil.Command(NewBuyCmd(), testutil.DryRun(true), testutil.PlainOutput())
	cmd.SetArgs([]string{"abc123", "--pm", "pm_card_visa", "--price-cents", "500"})
	out := testutil.CaptureStdout(func() { testutil.MustExecute(t, cmd) })

	if !strings.Contains(out, "POST\t/orders\t") {
		t.Fatalf("expected plain dry-run row, got %q", out)
	}
}

func TestBuy_PlainOutput_Success(t *testing.T) {
	testutil.Setup(t, ordersHandler(t, map[string]any{
		"line_items": map[string]any{
			"li-0": map[string]any{
				"uid":            "li-0",
				"success":        true,
				"name":           "Art Pack",
				"permalink":      "abc123",
				"content_url":    "https://gumroad.com/library/abc123",
				"redirect_token": "tok-1",
			},
		},
	}))

	cmd := testutil.Command(NewBuyCmd(), testutil.Yes(true), testutil.PlainOutput())
	cmd.SetArgs([]string{"abc123", "--pm", "pm_card_visa", "--price-cents", "500"})
	out := testutil.CaptureStdout(func() { testutil.MustExecute(t, cmd) })

	cols := strings.Split(strings.TrimRight(out, "\n"), "\t")
	if len(cols) != 4 {
		t.Fatalf("expected 4 tab-separated columns, got %d: %q", len(cols), out)
	}
	if cols[0] != "abc123" || cols[1] != "Art Pack" || cols[2] != "https://gumroad.com/library/abc123" || cols[3] != "tok-1" {
		t.Fatalf("unexpected plain row %v", cols)
	}
}

func TestBuy_QuietSuppressesOutput(t *testing.T) {
	testutil.Setup(t, ordersHandler(t, map[string]any{
		"line_items": map[string]any{
			"li-0": map[string]any{"uid": "li-0", "success": true, "name": "Art Pack", "permalink": "abc123"},
		},
	}))

	cmd := testutil.Command(NewBuyCmd(), testutil.Yes(true), testutil.Quiet(true))
	cmd.SetArgs([]string{"abc123", "--pm", "pm_card_visa", "--price-cents", "500"})
	out := testutil.CaptureStdout(func() { testutil.MustExecute(t, cmd) })

	if strings.TrimSpace(out) != "" {
		t.Fatalf("expected no output in quiet mode, got %q", out)
	}
}

func TestBuy_FallsBackToFirstLineItem(t *testing.T) {
	testutil.Setup(t, ordersHandler(t, map[string]any{
		"line_items": map[string]any{
			"other-uid": map[string]any{
				"uid": "other-uid", "success": true, "name": "Art Pack", "permalink": "abc123",
			},
		},
	}))

	cmd := testutil.Command(NewBuyCmd(), testutil.Yes(true), testutil.Quiet(false))
	cmd.SetArgs([]string{"abc123", "--pm", "pm_card_visa", "--price-cents", "500"})
	out := testutil.CaptureStdout(func() { testutil.MustExecute(t, cmd) })

	if !strings.Contains(out, "Purchased Art Pack") {
		t.Fatalf("expected success render from fallback line item, got %q", out)
	}
}

func TestBuy_EmptyLineItemsErrors(t *testing.T) {
	testutil.Setup(t, ordersHandler(t, map[string]any{"line_items": map[string]any{}}))

	cmd := testutil.Command(NewBuyCmd(), testutil.Yes(true))
	cmd.SetArgs([]string{"abc123", "--pm", "pm_card_visa", "--price-cents", "500"})
	err := cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "missing line items") {
		t.Fatalf("expected missing line items error, got %v", err)
	}
}

func TestBuy_QuantityRejectsZero(t *testing.T) {
	testutil.Setup(t, func(w http.ResponseWriter, r *http.Request) {
		t.Error("should not reach API with invalid --quantity")
	})

	cmd := testutil.Command(NewBuyCmd(), testutil.Yes(true))
	cmd.SetArgs([]string{"abc123", "--pm", "pm_card_visa", "--price-cents", "500", "--quantity", "0"})
	err := cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "--quantity must be greater than 0") {
		t.Fatalf("expected quantity error, got %v", err)
	}
}

func TestBuy_TipCentsRejectsNegative(t *testing.T) {
	testutil.Setup(t, func(w http.ResponseWriter, r *http.Request) {
		t.Error("should not reach API with negative --tip-cents")
	})

	cmd := testutil.Command(NewBuyCmd(), testutil.Yes(true))
	cmd.SetArgs([]string{"abc123", "--pm", "pm_card_visa", "--price-cents", "500", "--tip-cents", "-1"})
	err := cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "--tip-cents cannot be negative") {
		t.Fatalf("expected tip-cents error, got %v", err)
	}
}

func TestBuy_DeclineWithoutMessage(t *testing.T) {
	testutil.Setup(t, ordersHandler(t, map[string]any{
		"line_items": map[string]any{
			"li-0": map[string]any{"uid": "li-0", "success": false},
		},
	}))

	var stderr strings.Builder
	cmd := testutil.Command(NewBuyCmd(), testutil.Yes(true), testutil.Stderr(&stderr))
	cmd.SetArgs([]string{"abc123", "--pm", "pm_card_visa", "--price-cents", "500"})
	err := cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "purchase failed") {
		t.Fatalf("expected fallback purchase failed error, got %v", err)
	}
}

func TestBuy_OptionalFieldsOmittedWhenUnset(t *testing.T) {
	var gotBody map[string]any
	testutil.Setup(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			rejectGET(t)(w, r)
			return
		}
		gotBody = decodeOrderBody(t, r)
		testutil.JSON(t, w, map[string]any{
			"line_items": map[string]any{
				"li-0": map[string]any{"uid": "li-0", "success": true, "name": "Art Pack", "permalink": "abc123"},
			},
		})
	})

	cmd := testutil.Command(NewBuyCmd(), testutil.Yes(true), testutil.Quiet(true))
	cmd.SetArgs([]string{"abc123", "--pm", "pm_card_visa", "--price-cents", "500"})
	testutil.MustExecute(t, cmd)

	for _, key := range []string{"stripe_customer_id", "email"} {
		if _, ok := gotBody[key]; ok {
			t.Errorf("expected %s omitted when unset, got %v", key, gotBody[key])
		}
	}
	item := gotBody["line_items"].([]any)[0].(map[string]any)
	for _, key := range []string{"variants", "discount_code", "tip_cents"} {
		if _, ok := item[key]; ok {
			t.Errorf("expected line item %s omitted when unset, got %v", key, item[key])
		}
	}
}
