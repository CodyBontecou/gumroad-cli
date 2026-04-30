package output

import (
	"bytes"
	"strings"
	"testing"
)

func TestTheme_DisabledEmitsNoANSI(t *testing.T) {
	th := NewTheme(false)
	for _, got := range []string{
		th.Accent("x"),
		th.AccentBold("x"),
		th.Success("x"),
		th.Warning("x"),
		th.Error("x"),
		th.Info("x"),
		th.Muted("x"),
		th.Soft("x"),
		th.Bold("x"),
		th.AccentPill("LBL"),
		th.SuccessPill("LBL"),
		th.WarningPill("LBL"),
		th.ErrorPill("LBL"),
		th.InfoPill("LBL"),
		th.StatusBadge(StatusOK, "ok"),
		th.StatusBadge(StatusWarn, "warn"),
		th.StatusBadge(StatusErr, "err"),
		th.StatusBadge(StatusInfo, "info"),
		th.StatusBadge(StatusNeutral, "n"),
		th.Brand(),
		th.Banner("title", "subtitle"),
		th.Banner("title", ""),
		th.Status(StatusOK, "ok"),
		th.Status(StatusWarn, "warn"),
		th.Status(StatusErr, "err"),
		th.Status(StatusInfo, "info"),
		th.Status(StatusNeutral, "n"),
		th.ErrorBox("oops", "try again"),
		th.ErrorBox("oops", ""),
		th.Card("title", [][2]string{{"k", "v"}}),
	} {
		if strings.Contains(got, "\x1b[") {
			t.Fatalf("disabled theme leaked ANSI: %q", got)
		}
	}
}

func TestTheme_EnabledEmitsANSI(t *testing.T) {
	th := NewTheme(true)
	for name, got := range map[string]string{
		"Accent":   th.Accent("x"),
		"Success":  th.Success("x"),
		"Warning":  th.Warning("x"),
		"Error":    th.Error("x"),
		"Info":     th.Info("x"),
		"Muted":    th.Muted("x"),
		"Soft":     th.Soft("x"),
		"Bold":     th.Bold("x"),
		"Pill":     th.AccentPill("LBL"),
		"Banner":   th.Banner("hello", ""),
		"Status":   th.Status(StatusOK, "done"),
		"Card":     th.Card("title", [][2]string{{"k", "v"}}),
		"ErrorBox": th.ErrorBox("oops", "hint"),
	} {
		if !strings.Contains(got, "\x1b[") {
			t.Fatalf("enabled theme produced no ANSI for %s: %q", name, got)
		}
	}
}

func TestTheme_StatusKindFallback(t *testing.T) {
	th := NewTheme(false)
	if !strings.Contains(th.Status(StatusKind(99), "x"), iconBullet) {
		t.Fatal("expected unknown StatusKind to fall back to bullet")
	}
}

func TestTheme_PrintHelpersWriteToBuffer(t *testing.T) {
	th := NewTheme(false)
	var buf bytes.Buffer

	if err := th.PrintBanner(&buf, "title", "sub"); err != nil {
		t.Fatalf("PrintBanner: %v", err)
	}
	if err := th.PrintStatus(&buf, StatusOK, "ok"); err != nil {
		t.Fatalf("PrintStatus: %v", err)
	}
	if err := th.PrintCard(&buf, "title", [][2]string{{"k", "v"}}); err != nil {
		t.Fatalf("PrintCard: %v", err)
	}
	if err := th.PrintErrorBox(&buf, "oops", "hint"); err != nil {
		t.Fatalf("PrintErrorBox: %v", err)
	}

	for _, want := range []string{"title", "ok", "k", "v", "oops", "hint"} {
		if !strings.Contains(buf.String(), want) {
			t.Fatalf("expected %q in output: %q", want, buf.String())
		}
	}
}

func TestTheme_NewThemeForWriter(t *testing.T) {
	th := NewThemeForWriter(&bytes.Buffer{}, false)
	if th.Enabled() {
		t.Fatal("expected non-file writer to disable theme color")
	}
}

func TestTheme_BannerNoSubtitleEnabledAndDisabled(t *testing.T) {
	off := NewTheme(false).Banner("title", "")
	if off != "| title" {
		t.Fatalf("disabled no-sub banner: %q", off)
	}
	on := NewTheme(true).Banner("title", "")
	if !strings.Contains(on, "title") || !strings.Contains(on, "\x1b[") {
		t.Fatalf("enabled no-sub banner missing title or color: %q", on)
	}
}

func TestTheme_StatusBadgeAllKinds(t *testing.T) {
	th := NewTheme(true)
	for _, k := range []StatusKind{StatusOK, StatusWarn, StatusErr, StatusInfo, StatusNeutral} {
		if th.StatusBadge(k, "label") == "" {
			t.Fatalf("empty badge for kind %v", k)
		}
	}
}

func TestTheme_BrandHasAccentWhenEnabled(t *testing.T) {
	if !strings.Contains(NewTheme(true).Brand(), "\x1b[") {
		t.Fatal("expected enabled Brand to be styled")
	}
	if NewTheme(false).Brand() != "gumroad" {
		t.Fatal("expected disabled Brand to be plain")
	}
}

func TestPadRight(t *testing.T) {
	if got := padRight("ab", 4); got != "ab  " {
		t.Fatalf("padRight short: got %q", got)
	}
	if got := padRight("abcd", 2); got != "abcd" {
		t.Fatalf("padRight already wider: got %q", got)
	}
}
