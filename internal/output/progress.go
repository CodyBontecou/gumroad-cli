package output

import (
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/charmbracelet/lipgloss"
)

var isProgressTerminal = func(w io.Writer) bool {
	return isTerminalWriter(w)
}

// ProgressBar is a thin, line-redrawing progress bar for known-length tasks
// (file uploads, paginated bulk fetches). Render is a no-op on non-TTY writers
// and TERM=dumb so output stays clean when piped or logged.
type ProgressBar struct {
	writer  io.Writer
	label   string
	total   int64
	current int64
	width   int
	started time.Time
	mu      sync.Mutex
	theme   Theme
	dirty   bool
}

const (
	progressFilledRune = "━"
	progressEmptyRune  = "─"
	progressBarWidth   = 28
)

// NewProgressBar creates a progress bar that writes to w. total is the target
// value; pass 0 if unknown to render an indeterminate label without a bar.
func NewProgressBar(w io.Writer, label string, total int64) *ProgressBar {
	if w == nil {
		w = os.Stderr
	}
	colorful := NewStylerForWriter(w, false).Enabled()
	return &ProgressBar{
		writer:  w,
		label:   label,
		total:   total,
		width:   progressBarWidth,
		started: time.Now(),
		theme:   NewTheme(colorful),
	}
}

// Set updates the current value and redraws the bar.
func (p *ProgressBar) Set(current int64) {
	p.mu.Lock()
	p.current = current
	p.dirty = true
	p.mu.Unlock()
	p.render()
}

// Add increments the current value by delta and redraws.
func (p *ProgressBar) Add(delta int64) {
	p.mu.Lock()
	p.current += delta
	p.dirty = true
	p.mu.Unlock()
	p.render()
}

// SetLabel changes the bar label. Useful for multi-stage operations.
func (p *ProgressBar) SetLabel(label string) {
	p.mu.Lock()
	p.label = label
	p.dirty = true
	p.mu.Unlock()
	p.render()
}

// Done clears the bar line and prints a final success status.
func (p *ProgressBar) Done(message string) {
	p.finish(StatusOK, message)
}

// Fail clears the bar line and prints a final error status.
func (p *ProgressBar) Fail(message string) {
	p.finish(StatusErr, message)
}

func (p *ProgressBar) finish(kind StatusKind, message string) {
	if !isProgressTerminal(p.writer) || isDumbTerminal() {
		if message != "" {
			fmt.Fprintln(p.writer, message)
		}
		return
	}
	fmt.Fprint(p.writer, "\r\033[K")
	if message != "" {
		fmt.Fprintln(p.writer, p.theme.Status(kind, message))
	}
}

func (p *ProgressBar) render() {
	if !isProgressTerminal(p.writer) || isDumbTerminal() {
		return
	}
	p.mu.Lock()
	current := p.current
	total := p.total
	label := p.label
	width := p.width
	elapsed := time.Since(p.started)
	p.mu.Unlock()

	var line string
	if total <= 0 {
		spinner := spinnerFrames[int(elapsed/spinnerFrameRate)%len(spinnerFrames)]
		line = fmt.Sprintf("%s %s %s",
			p.theme.Accent(spinner),
			p.theme.Soft(label),
			p.theme.Muted("("+formatElapsed(elapsed)+")"),
		)
	} else {
		ratio := float64(current) / float64(total)
		if ratio < 0 {
			ratio = 0
		}
		if ratio > 1 {
			ratio = 1
		}
		filled := int(float64(width) * ratio)
		bar := renderBar(p.theme, width, filled)
		percent := fmt.Sprintf("%3.0f%%", ratio*100)
		line = fmt.Sprintf("%s  %s  %s  %s",
			p.theme.Soft(label),
			bar,
			p.theme.AccentBold(percent),
			p.theme.Muted(formatProgressEta(current, total, elapsed)),
		)
	}
	fmt.Fprintf(p.writer, "\r\033[K%s", line)
}

func renderBar(theme Theme, width, filled int) string {
	if !theme.Enabled() {
		return "[" + strings.Repeat("=", filled) + strings.Repeat(" ", width-filled) + "]"
	}
	filledStr := theme.style().
		Foreground(lipgloss.Color(colorAccent)).
		Render(strings.Repeat(progressFilledRune, filled))
	emptyStr := theme.style().
		Foreground(lipgloss.Color(colorBorder)).
		Render(strings.Repeat(progressEmptyRune, width-filled))
	return filledStr + emptyStr
}

func formatProgressEta(current, total int64, elapsed time.Duration) string {
	if current <= 0 || elapsed <= 0 {
		return "—"
	}
	rate := float64(current) / elapsed.Seconds()
	if rate <= 0 {
		return "—"
	}
	remaining := time.Duration(float64(total-current)/rate) * time.Second
	if remaining < 0 {
		remaining = 0
	}
	return formatElapsed(remaining) + " left"
}
