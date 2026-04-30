package output

import (
	"fmt"
	"io"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
)

const (
	colorAccent   = "#FF90E8"
	colorAccent2  = "#C4528E"
	colorSuccess  = "#00C896"
	colorWarning  = "#FFC23C"
	colorError    = "#FF5A5F"
	colorInfo     = "#5B8DEF"
	colorMuted    = "#8B8D98"
	colorSurface  = "#1F1F23"
	colorBorder   = "#3A3A42"
	colorTextDim  = "#6F7079"
	colorTextSoft = "#C7C9D1"
)

const (
	iconSuccess = "✓"
	iconWarning = "⚠"
	iconError   = "✗"
	iconInfo    = "›"
	iconBullet  = "•"
	iconArrow   = "→"
)

type Theme struct {
	enabled  bool
	renderer *lipgloss.Renderer
}

func NewTheme(enabled bool) Theme {
	r := lipgloss.NewRenderer(io.Discard)
	if enabled {
		r.SetColorProfile(termenv.TrueColor)
	} else {
		r.SetColorProfile(termenv.Ascii)
	}
	return Theme{enabled: enabled, renderer: r}
}

func NewThemeForWriter(w io.Writer, noColor bool) Theme {
	return NewTheme(NewStylerForWriter(w, noColor).Enabled())
}

func (t Theme) Enabled() bool { return t.enabled }

func (t Theme) style() lipgloss.Style {
	return t.renderer.NewStyle()
}

func (t Theme) color(s, color string) string {
	if !t.enabled {
		return s
	}
	return t.style().Foreground(lipgloss.Color(color)).Render(s)
}

func (t Theme) Accent(s string) string  { return t.color(s, colorAccent) }
func (t Theme) Success(s string) string { return t.color(s, colorSuccess) }
func (t Theme) Warning(s string) string { return t.color(s, colorWarning) }
func (t Theme) Error(s string) string   { return t.color(s, colorError) }
func (t Theme) Info(s string) string    { return t.color(s, colorInfo) }
func (t Theme) Muted(s string) string   { return t.color(s, colorMuted) }
func (t Theme) Soft(s string) string    { return t.color(s, colorTextSoft) }

func (t Theme) AccentBold(s string) string {
	if !t.enabled {
		return s
	}
	return t.style().Foreground(lipgloss.Color(colorAccent)).Bold(true).Render(s)
}

func (t Theme) Bold(s string) string {
	if !t.enabled {
		return s
	}
	return t.style().Bold(true).Render(s)
}

func (t Theme) Pill(label, color string) string {
	if !t.enabled {
		return " " + label + " "
	}
	return t.style().
		Foreground(lipgloss.Color("#0E0E11")).
		Background(lipgloss.Color(color)).
		Bold(true).
		Padding(0, 1).
		Render(label)
}

func (t Theme) AccentPill(label string) string  { return t.Pill(label, colorAccent) }
func (t Theme) SuccessPill(label string) string { return t.Pill(label, colorSuccess) }
func (t Theme) WarningPill(label string) string { return t.Pill(label, colorWarning) }
func (t Theme) ErrorPill(label string) string   { return t.Pill(label, colorError) }
func (t Theme) InfoPill(label string) string    { return t.Pill(label, colorInfo) }

func (t Theme) StatusBadge(kind StatusKind, label string) string {
	switch kind {
	case StatusOK:
		return t.SuccessPill(strings.ToUpper(label))
	case StatusWarn:
		return t.WarningPill(strings.ToUpper(label))
	case StatusErr:
		return t.ErrorPill(strings.ToUpper(label))
	case StatusInfo:
		return t.InfoPill(strings.ToUpper(label))
	default:
		return t.style().Foreground(lipgloss.Color(colorMuted)).Render(label)
	}
}

type StatusKind int

const (
	StatusNeutral StatusKind = iota
	StatusOK
	StatusWarn
	StatusErr
	StatusInfo
)

// Brand renders the gumroad wordmark as a single styled token. Callers can
// drop it into help text or banners without having to reach for theme colors.
func (t Theme) Brand() string {
	return t.AccentBold("gumroad")
}

// Banner renders a one-line title block with an accent pipe on the left.
// Use it to mark the start of a long-running command's output.
func (t Theme) Banner(title, subtitle string) string {
	if !t.enabled {
		if subtitle == "" {
			return "| " + title
		}
		return "| " + title + "  " + subtitle
	}
	bar := t.Accent("┃ ")
	head := t.AccentBold(title)
	if subtitle == "" {
		return bar + head
	}
	return bar + head + t.Muted("  "+subtitle)
}

// Card renders a rounded-border block with a title row and key-value rows.
// Falls back to a flat layout when color is disabled so logs stay readable.
func (t Theme) Card(title string, rows [][2]string) string {
	if !t.enabled {
		return cardPlain(title, rows)
	}
	keyW := 0
	for _, r := range rows {
		if w := lipgloss.Width(r[0]); w > keyW {
			keyW = w
		}
	}
	var b strings.Builder
	b.WriteString(t.AccentBold(title))
	b.WriteString("\n")
	for i, r := range rows {
		key := t.Muted(padRight(r[0], keyW))
		val := t.Soft(r[1])
		b.WriteString(key + "  " + val)
		if i < len(rows)-1 {
			b.WriteString("\n")
		}
	}
	return t.style().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(colorBorder)).
		Padding(0, 2).
		Render(b.String())
}

func cardPlain(title string, rows [][2]string) string {
	var b strings.Builder
	b.WriteString(title + "\n")
	for _, r := range rows {
		fmt.Fprintf(&b, "  %s: %s\n", r[0], r[1])
	}
	return strings.TrimRight(b.String(), "\n")
}

// ErrorBox renders a multi-line error block with a red side bar, error icon,
// the headline message, and a dim hint line. The hint is optional.
func (t Theme) ErrorBox(headline, hint string) string {
	if !t.enabled {
		if hint == "" {
			return iconError + " " + headline
		}
		return iconError + " " + headline + "\n  " + hint
	}
	bar := t.Error("▌")
	icon := t.style().Foreground(lipgloss.Color(colorError)).Bold(true).Render(iconError)
	head := t.style().Foreground(lipgloss.Color(colorTextSoft)).Bold(true).Render(headline)
	out := bar + " " + icon + " " + head
	if hint != "" {
		out += "\n" + bar + "   " + t.Muted(hint)
	}
	return out
}

// Status renders a single line `<icon> <message>` colored by kind. Mutations
// should call this after the spinner stops to give users a clear final state.
func (t Theme) Status(kind StatusKind, message string) string {
	icon := iconBullet
	color := colorMuted
	switch kind {
	case StatusOK:
		icon = iconSuccess
		color = colorSuccess
	case StatusWarn:
		icon = iconWarning
		color = colorWarning
	case StatusErr:
		icon = iconError
		color = colorError
	case StatusInfo:
		icon = iconInfo
		color = colorInfo
	}
	if !t.enabled {
		return icon + " " + message
	}
	iconStr := t.style().Foreground(lipgloss.Color(color)).Bold(true).Render(icon)
	return iconStr + " " + t.Soft(message)
}

// PrintBanner writes a Banner to w followed by a newline.
func (t Theme) PrintBanner(w io.Writer, title, subtitle string) error {
	_, err := fmt.Fprintln(w, t.Banner(title, subtitle))
	return err
}

// PrintStatus writes a Status line to w followed by a newline.
func (t Theme) PrintStatus(w io.Writer, kind StatusKind, message string) error {
	_, err := fmt.Fprintln(w, t.Status(kind, message))
	return err
}

// PrintCard writes a Card to w followed by a newline.
func (t Theme) PrintCard(w io.Writer, title string, rows [][2]string) error {
	_, err := fmt.Fprintln(w, t.Card(title, rows))
	return err
}

// PrintErrorBox writes an ErrorBox to w followed by a newline.
func (t Theme) PrintErrorBox(w io.Writer, headline, hint string) error {
	_, err := fmt.Fprintln(w, t.ErrorBox(headline, hint))
	return err
}

func padRight(s string, w int) string {
	if vw := lipgloss.Width(s); vw < w {
		return s + strings.Repeat(" ", w-vw)
	}
	return s
}
