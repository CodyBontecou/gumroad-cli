package buy

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/antiwork/gumroad-cli/internal/cmdutil"
	"github.com/antiwork/gumroad-cli/internal/testutil"
)

func fakeLinkBinary(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "fake-link-cli")
	const script = `#!/bin/sh
case "$1 $2" in
  "payment-methods list")
    printf '[{"id":"csmrpd_test","type":"card"}]'
    ;;
  "spend-request create")
    printf '{"id":"lsrq_test","status":"approved"}'
    ;;
  "spend-request retrieve")
    printf '{"id":"lsrq_test","status":"approved","card":{"number":"4242424242424242","cvc":"123","exp_month":12,"exp_year":2030}}'
    ;;
  *)
    echo "unexpected: $@" >&2
    exit 1
    ;;
esac
`
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil { //nolint:gosec // G306: test fixture must be executable
		t.Fatalf("write fake link-cli: %v", err)
	}
	return path
}

type stubSpender struct {
	pm        string
	err       error
	gotParams linkBuyParams
	calls     atomic.Int32
}

func (s *stubSpender) Mint(_ context.Context, p linkBuyParams) (string, error) {
	s.calls.Add(1)
	s.gotParams = p
	if s.err != nil {
		return "", s.err
	}
	return s.pm, nil
}

func withSpender(t *testing.T, sp linkSpender) {
	t.Helper()
	prev := newLinkSpender
	newLinkSpender = func() linkSpender { return sp }
	t.Cleanup(func() { newLinkSpender = prev })
}

func TestBuy_LinkAndPMAreMutuallyExclusive(t *testing.T) {
	cmd := testutil.Command(NewBuyCmd())
	cmd.SetArgs([]string{"abc123", "--link", "--pm", "pm_card_visa", "--price-cents", "500"})
	err := cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "mutually exclusive") {
		t.Fatalf("expected mutually-exclusive error, got %v", err)
	}
	var usageErr *cmdutil.UsageError
	if !errors.As(err, &usageErr) {
		t.Fatalf("expected UsageError, got %T", err)
	}
}

func TestBuy_LinkRequiresPriceCents(t *testing.T) {
	cmd := testutil.Command(NewBuyCmd())
	cmd.SetArgs([]string{"abc123", "--link"})
	err := cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "--price-cents") {
		t.Fatalf("expected --price-cents error, got %v", err)
	}
}

func TestBuy_LinkSkipsPaymentMethodRequirement(t *testing.T) {
	sp := &stubSpender{pm: "pm_link_minted"}
	withSpender(t, sp)

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
	cmd.SetArgs([]string{"abc123", "--link", "--price-cents", "500", "--quantity", "2", "--tip-cents", "100"})
	testutil.MustExecute(t, cmd)

	if sp.calls.Load() != 1 {
		t.Fatalf("expected linkSpender.Mint called once, got %d", sp.calls.Load())
	}
	if sp.gotParams.Permalink != "abc123" || sp.gotParams.PriceCents != 500 || sp.gotParams.Quantity != 2 || sp.gotParams.TipCents != 100 {
		t.Fatalf("got params %+v, want permalink/price/qty/tip 'abc123'/500/2/100", sp.gotParams)
	}
	if gotBody["stripe_payment_method_id"] != "pm_link_minted" {
		t.Fatalf("got stripe_payment_method_id %v, want pm_link_minted", gotBody["stripe_payment_method_id"])
	}
}

func TestBuy_LinkMintError_DoesNotPostOrder(t *testing.T) {
	sp := &stubSpender{err: errors.New("link-cli not found on PATH")}
	withSpender(t, sp)

	testutil.Setup(t, func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("must not POST when Link mint fails; got %s %s", r.Method, r.URL.Path)
		http.Error(w, "no API call expected", http.StatusInternalServerError)
	})

	cmd := testutil.Command(NewBuyCmd(), testutil.Yes(true), testutil.Quiet(true))
	cmd.SetArgs([]string{"abc123", "--link", "--price-cents", "500"})
	err := cmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "link-cli not found") {
		t.Fatalf("expected mint error to surface, got %v", err)
	}
}

func TestBuy_LinkDryRunDoesNotMint(t *testing.T) {
	sp := &stubSpender{pm: "pm_should_not_mint"}
	withSpender(t, sp)

	testutil.Setup(t, func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("dry-run must not touch the network; got %s %s", r.Method, r.URL.Path)
		http.Error(w, "no API call expected", http.StatusInternalServerError)
	})

	cmd := testutil.Command(NewBuyCmd(), testutil.DryRun(true), testutil.Quiet(false))
	cmd.SetArgs([]string{"abc123", "--link", "--price-cents", "500"})
	out := testutil.CaptureStdout(func() { testutil.MustExecute(t, cmd) })

	if sp.calls.Load() != 0 {
		t.Fatalf("dry-run must not call Mint; got %d calls", sp.calls.Load())
	}
	if !strings.Contains(out, linkPlaceholderPM) {
		t.Fatalf("dry-run should show placeholder PM %q for --link; got %q", linkPlaceholderPM, out)
	}
}

func TestCliLinkSpender_RequiresLinkCLI(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	t.Setenv(envStripePublishableKey, "pk_test_dummy")

	sp := newCliLinkSpender()
	sp.binary = "link-cli-not-on-path-xyz"
	_, err := sp.Mint(context.Background(), linkBuyParams{Permalink: "p", PriceCents: 500, Quantity: 1})
	if err == nil || !strings.Contains(err.Error(), "link-cli not found") {
		t.Fatalf("expected 'link-cli not found' error, got %v", err)
	}
}

func TestCliLinkSpender_RequiresPublishableKey(t *testing.T) {
	t.Setenv(envStripePublishableKey, "")

	sp := newCliLinkSpender()
	sp.binary = fakeLinkBinary(t)
	_, err := sp.Mint(context.Background(), linkBuyParams{Permalink: "p", PriceCents: 500, Quantity: 1})
	if err == nil || !strings.Contains(err.Error(), envStripePublishableKey) {
		t.Fatalf("expected publishable-key error, got %v", err)
	}
}

func TestCliLinkSpender_RejectsLiveKey(t *testing.T) {
	t.Setenv(envStripePublishableKey, "pk_live_xxx")

	sp := newCliLinkSpender()
	sp.binary = fakeLinkBinary(t)
	_, err := sp.Mint(context.Background(), linkBuyParams{Permalink: "p", PriceCents: 500, Quantity: 1})
	if err == nil || !strings.Contains(err.Error(), "test publishable key") {
		t.Fatalf("expected live-key rejection, got %v", err)
	}
}

func TestCliLinkSpender_TokenizesAndReturnsPM(t *testing.T) {
	stripe := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer pk_test_dummy" {
			t.Errorf("got auth %q, want Bearer pk_test_dummy", got)
		}
		if err := r.ParseForm(); err != nil {
			t.Fatalf("parse form: %v", err)
		}
		want := url.Values{
			"type":            []string{"card"},
			"card[number]":    []string{"4242424242424242"},
			"card[cvc]":       []string{"123"},
			"card[exp_month]": []string{"12"},
			"card[exp_year]":  []string{"2030"},
		}
		for k, v := range want {
			if got := r.Form.Get(k); got != v[0] {
				t.Errorf("form[%s] = %q, want %q", k, got, v[0])
			}
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"id": "pm_stripe_minted"})
	}))
	t.Cleanup(stripe.Close)

	t.Setenv(envStripePublishableKey, "pk_test_dummy")
	sp := newCliLinkSpender()
	sp.binary = fakeLinkBinary(t)
	sp.stripeURL = stripe.URL
	pm, err := sp.Mint(context.Background(), linkBuyParams{Permalink: "p", PriceCents: 500, Quantity: 1})
	if err != nil {
		t.Fatalf("mint: %v", err)
	}
	if pm != "pm_stripe_minted" {
		t.Fatalf("got pm %q, want pm_stripe_minted", pm)
	}
}

func TestCliLinkSpender_TokenizationFailureSurfaces(t *testing.T) {
	stripe := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"error": map[string]any{"message": "Your card number is invalid.", "code": "incorrect_number"},
		})
	}))
	t.Cleanup(stripe.Close)

	t.Setenv(envStripePublishableKey, "pk_test_dummy")
	sp := newCliLinkSpender()
	sp.binary = fakeLinkBinary(t)
	sp.stripeURL = stripe.URL
	_, err := sp.Mint(context.Background(), linkBuyParams{Permalink: "p", PriceCents: 500, Quantity: 1})
	if err == nil || !strings.Contains(err.Error(), "Your card number is invalid") {
		t.Fatalf("expected stripe error to surface, got %v", err)
	}
}

func TestBuildLinkContext_MeetsMinLength(t *testing.T) {
	ctx := buildLinkContext(linkBuyParams{Permalink: "x", PriceCents: 1, Quantity: 1}, 1)
	if len(ctx) < linkContextMinLen {
		t.Fatalf("context %d chars, want >= %d: %q", len(ctx), linkContextMinLen, ctx)
	}
}

func TestSanitizeLinkValue_StripsCommasAndColons(t *testing.T) {
	if got := sanitizeLinkValue("a,b:c"); strings.ContainsAny(got, ",:") {
		t.Fatalf("got %q, expected no comma or colon", got)
	}
}

func writeFakeLinkScript(t *testing.T, body string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "fake-link-cli")
	script := "#!/bin/sh\n" + body + "\n"
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil { //nolint:gosec // G306: test fixture must be executable
		t.Fatalf("write fake link-cli: %v", err)
	}
	return path
}

func TestCliLinkSpender_NoLinkPaymentMethodsErrors(t *testing.T) {
	t.Setenv(envStripePublishableKey, "pk_test_dummy")
	sp := newCliLinkSpender()
	sp.binary = writeFakeLinkScript(t, `printf '[]'`)
	_, err := sp.Mint(context.Background(), linkBuyParams{Permalink: "p", PriceCents: 500, Quantity: 1})
	if err == nil || !strings.Contains(err.Error(), "no payment methods") {
		t.Fatalf("expected 'no payment methods' error, got %v", err)
	}
}

func TestCliLinkSpender_LinkCLIExitErrorSurfacesStderr(t *testing.T) {
	t.Setenv(envStripePublishableKey, "pk_test_dummy")
	sp := newCliLinkSpender()
	sp.binary = writeFakeLinkScript(t, `echo "link auth required" >&2; exit 2`)
	_, err := sp.Mint(context.Background(), linkBuyParams{Permalink: "p", PriceCents: 500, Quantity: 1})
	if err == nil || !strings.Contains(err.Error(), "link auth required") {
		t.Fatalf("expected stderr to surface, got %v", err)
	}
}

func TestCliLinkSpender_NotApprovedStatusErrors(t *testing.T) {
	t.Setenv(envStripePublishableKey, "pk_test_dummy")
	sp := newCliLinkSpender()
	sp.binary = writeFakeLinkScript(t, `case "$1 $2" in
"payment-methods list") printf '[{"id":"csmrpd_x","type":"card"}]' ;;
"spend-request create") printf '{"id":"lsrq_x","status":"denied"}' ;;
"spend-request retrieve") printf '{"id":"lsrq_x","status":"denied"}' ;;
esac`)
	_, err := sp.Mint(context.Background(), linkBuyParams{Permalink: "p", PriceCents: 500, Quantity: 1})
	if err == nil || !strings.Contains(err.Error(), "denied") {
		t.Fatalf("expected denied-status error, got %v", err)
	}
}

func TestCliLinkSpender_RetrieveWithoutCardErrors(t *testing.T) {
	stripe := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"id": "pm_x"})
	}))
	t.Cleanup(stripe.Close)
	t.Setenv(envStripePublishableKey, "pk_test_dummy")
	sp := newCliLinkSpender()
	sp.stripeURL = stripe.URL
	sp.binary = writeFakeLinkScript(t, `case "$1 $2" in
"payment-methods list") printf '[{"id":"csmrpd_x","type":"card"}]' ;;
"spend-request create") printf '{"id":"lsrq_x","status":"approved"}' ;;
"spend-request retrieve") printf '{"id":"lsrq_x","status":"approved"}' ;;
esac`)
	_, err := sp.Mint(context.Background(), linkBuyParams{Permalink: "p", PriceCents: 500, Quantity: 1})
	if err == nil || !strings.Contains(err.Error(), "did not include card credentials") {
		t.Fatalf("expected missing-card error, got %v", err)
	}
}

func TestCliLinkSpender_StripeReturnsInvalidJSON(t *testing.T) {
	stripe := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("not json"))
	}))
	t.Cleanup(stripe.Close)
	t.Setenv(envStripePublishableKey, "pk_test_dummy")
	sp := newCliLinkSpender()
	sp.binary = fakeLinkBinary(t)
	sp.stripeURL = stripe.URL
	_, err := sp.Mint(context.Background(), linkBuyParams{Permalink: "p", PriceCents: 500, Quantity: 1})
	if err == nil || !strings.Contains(err.Error(), "parse stripe response") {
		t.Fatalf("expected parse-error, got %v", err)
	}
}

func TestParseSpendRequest_InvalidJSONErrors(t *testing.T) {
	if _, err := parseSpendRequest([]byte("not json")); err == nil {
		t.Fatal("expected parse error")
	}
}

func TestTruncate(t *testing.T) {
	if got := truncate([]byte("short"), 10); got != "short" {
		t.Fatalf("got %q, want short", got)
	}
	if got := truncate([]byte("0123456789"), 5); got != "01234…" {
		t.Fatalf("got %q, want 01234…", got)
	}
}
