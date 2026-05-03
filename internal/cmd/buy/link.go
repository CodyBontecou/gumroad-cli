package buy

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

const (
	envStripePublishableKey = "GUMROAD_STRIPE_PUBLISHABLE_KEY"
	stripePaymentMethodsURL = "https://api.stripe.com/v1/payment_methods"
	linkMerchantName        = "Gumroad"
	linkMerchantURL         = "https://gumroad.com"
	linkContextMinLen       = 100
	linkPlaceholderPM       = "pm-pending-link-mint"
)

type linkBuyParams struct {
	Permalink  string
	PriceCents int
	Quantity   int
	TipCents   int
}

type linkSpender interface {
	Mint(ctx context.Context, params linkBuyParams) (string, error)
}

var newLinkSpender = func() linkSpender { return newCliLinkSpender() }

type cliLinkSpender struct {
	binary       string
	stripeURL    string
	httpClient   *http.Client
	publishKeyEv string
}

func newCliLinkSpender() *cliLinkSpender {
	return &cliLinkSpender{
		binary:       "link-cli",
		stripeURL:    stripePaymentMethodsURL,
		httpClient:   http.DefaultClient,
		publishKeyEv: envStripePublishableKey,
	}
}

type linkPaymentMethod struct {
	ID   string `json:"id"`
	Type string `json:"type"`
}

type linkSpendRequest struct {
	ID     string         `json:"id"`
	Status string         `json:"status"`
	Card   *linkCardCreds `json:"card,omitempty"`
}

type linkCardCreds struct {
	Number   string `json:"number"`
	CVC      string `json:"cvc"`
	ExpMonth int    `json:"exp_month"`
	ExpYear  int    `json:"exp_year"`
}

func (c *cliLinkSpender) Mint(ctx context.Context, params linkBuyParams) (string, error) {
	if _, err := exec.LookPath(c.binary); err != nil {
		return "", fmt.Errorf("link-cli not found on PATH; install with `npm i -g @stripe/link-cli`, then run `link-cli auth login`")
	}
	pk := strings.TrimSpace(os.Getenv(c.publishKeyEv))
	if pk == "" {
		return "", fmt.Errorf("%s is not set; export Gumroad's Stripe test publishable key (pk_test_…) so the CLI can tokenize the Link card", c.publishKeyEv)
	}
	if !strings.HasPrefix(pk, "pk_test_") {
		return "", fmt.Errorf("%s must be a Stripe test publishable key (pk_test_…)", c.publishKeyEv)
	}

	pmID, err := c.firstPaymentMethod(ctx)
	if err != nil {
		return "", err
	}
	sr, err := c.createSpendRequest(ctx, pmID, params)
	if err != nil {
		return "", err
	}
	if sr.Status != "approved" {
		sr, err = c.waitForApproval(ctx, sr.ID)
		if err != nil {
			return "", err
		}
	}
	if sr.Status != "approved" {
		return "", fmt.Errorf("spend request %s ended with status %q; approve in your Link app and retry", sr.ID, sr.Status)
	}
	if sr.Card == nil || sr.Card.Number == "" {
		card, err := c.retrieveCard(ctx, sr.ID)
		if err != nil {
			return "", err
		}
		sr.Card = card
	}
	return c.tokenize(ctx, pk, sr.Card)
}

func (c *cliLinkSpender) waitForApproval(ctx context.Context, id string) (*linkSpendRequest, error) {
	out, err := c.runLink(ctx, "spend-request", "retrieve", id, "--include=card", "--interval", "2", "--max-attempts", "150", "--format", "json")
	if err != nil {
		return nil, fmt.Errorf("link-cli spend-request retrieve (poll): %w", err)
	}
	return parseSpendRequest(out)
}

func (c *cliLinkSpender) firstPaymentMethod(ctx context.Context) (string, error) {
	out, err := c.runLink(ctx, "payment-methods", "list", "--format", "json")
	if err != nil {
		return "", fmt.Errorf("link-cli payment-methods list: %w", err)
	}
	var pms []linkPaymentMethod
	if err := json.Unmarshal(out, &pms); err != nil {
		return "", fmt.Errorf("parse link-cli payment-methods list: %w (raw: %s)", err, truncate(out, 200))
	}
	if len(pms) == 0 {
		return "", errors.New("no payment methods in your Link wallet; add one at https://app.link.com/wallet")
	}
	return pms[0].ID, nil
}

func (c *cliLinkSpender) createSpendRequest(ctx context.Context, pmID string, p linkBuyParams) (*linkSpendRequest, error) {
	total := p.PriceCents*p.Quantity + p.TipCents
	args := []string{
		"spend-request", "create",
		"--test",
		"--request-approval",
		"--format", "json",
		"--payment-method-id", pmID,
		"--merchant-name", linkMerchantName,
		"--merchant-url", linkMerchantURL,
		"--context", buildLinkContext(p, total),
		"--amount", strconv.Itoa(total),
		"--currency", "usd",
		"--line-item", fmt.Sprintf("name:%s,unit_amount:%d,quantity:%d", sanitizeLinkValue(p.Permalink), p.PriceCents, p.Quantity),
		"--total", fmt.Sprintf("type:total,display_text:Total,amount:%d", total),
	}
	out, err := c.runLink(ctx, args...)
	if err != nil {
		return nil, fmt.Errorf("link-cli spend-request create: %w", err)
	}
	return parseSpendRequest(out)
}

func (c *cliLinkSpender) retrieveCard(ctx context.Context, id string) (*linkCardCreds, error) {
	out, err := c.runLink(ctx, "spend-request", "retrieve", id, "--include=card", "--format", "json")
	if err != nil {
		return nil, fmt.Errorf("link-cli spend-request retrieve: %w", err)
	}
	sr, err := parseSpendRequest(out)
	if err != nil {
		return nil, err
	}
	if sr.Card == nil || sr.Card.Number == "" {
		return nil, fmt.Errorf("spend request %s did not include card credentials (status %q)", sr.ID, sr.Status)
	}
	return sr.Card, nil
}

func (c *cliLinkSpender) tokenize(ctx context.Context, pk string, card *linkCardCreds) (string, error) {
	form := url.Values{}
	form.Set("type", "card")
	form.Set("card[number]", card.Number)
	form.Set("card[cvc]", card.CVC)
	form.Set("card[exp_month]", strconv.Itoa(card.ExpMonth))
	form.Set("card[exp_year]", strconv.Itoa(card.ExpYear))

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.stripeURL, strings.NewReader(form.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+pk)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("stripe tokenize request: %w", err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)

	var parsed struct {
		ID    string `json:"id"`
		Error struct {
			Message string `json:"message"`
			Code    string `json:"code"`
		} `json:"error"`
	}
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return "", fmt.Errorf("parse stripe response (status %d): %w; raw: %s", resp.StatusCode, err, truncate(raw, 200))
	}
	if resp.StatusCode != http.StatusOK || parsed.ID == "" {
		msg := parsed.Error.Message
		if msg == "" {
			msg = strings.TrimSpace(string(raw))
		}
		return "", fmt.Errorf("stripe tokenize failed (HTTP %d): %s", resp.StatusCode, msg)
	}
	return parsed.ID, nil
}

func (c *cliLinkSpender) runLink(ctx context.Context, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, c.binary, args...) //nolint:gosec // G204: link-cli binary path is operator-controlled, args are constructed from validated buy flags
	cmd.Env = append(os.Environ(), "NO_UPDATE_NOTIFIER=1")
	out, err := cmd.Output()
	if err != nil {
		var ee *exec.ExitError
		if errors.As(err, &ee) {
			stderr := strings.TrimSpace(string(ee.Stderr))
			if stderr != "" {
				return nil, fmt.Errorf("%s", stderr)
			}
		}
		return nil, err
	}
	return out, nil
}

func parseSpendRequest(raw []byte) (*linkSpendRequest, error) {
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) > 0 && trimmed[0] == '[' {
		var srs []linkSpendRequest
		if err := json.Unmarshal(raw, &srs); err != nil {
			return nil, fmt.Errorf("parse spend request: %w (raw: %s)", err, truncate(raw, 200))
		}
		if len(srs) == 0 {
			return nil, fmt.Errorf("link-cli returned empty spend request array (raw: %s)", truncate(raw, 200))
		}
		return &srs[0], nil
	}
	var sr linkSpendRequest
	if err := json.Unmarshal(raw, &sr); err != nil {
		return nil, fmt.Errorf("parse spend request: %w (raw: %s)", err, truncate(raw, 200))
	}
	return &sr, nil
}

func buildLinkContext(p linkBuyParams, total int) string {
	s := fmt.Sprintf("Purchasing Gumroad product %q via the gumroad-cli `buy --link` command in development mode against a local Rails instance. Total %d cents (%d × %d).", p.Permalink, total, p.PriceCents, p.Quantity)
	for len(s) < linkContextMinLen {
		s += " (development)"
	}
	return s
}

func sanitizeLinkValue(s string) string {
	s = strings.ReplaceAll(s, ",", " ")
	s = strings.ReplaceAll(s, ":", " ")
	return s
}

func truncate(b []byte, n int) string {
	if len(b) <= n {
		return string(b)
	}
	return string(b[:n]) + "…"
}
