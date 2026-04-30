package output

import (
	"fmt"
	"io"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/mattn/go-runewidth"
)

type Table struct {
	headers []string
	rows    [][]string
	styler  Styler
	styled  bool
}

var ansiSequencePattern = regexp.MustCompile(`\x1b\[[0-9;]*m`)
var numericCellPattern = regexp.MustCompile(`^-?\$?\d[\d,]*(\.\d+)?%?$`)

const (
	defaultTableWidth       = 120
	minTableColumnWidth     = 4
	maxHeaderMinimumWidth   = 16
	tableColumnSeparator    = "  "
	tableTruncationEllipsis = "…"
	tableSeparatorRune      = "─"
)

func NewTable(headers ...string) *Table {
	return &Table{headers: headers}
}

// NewStyledTable binds a command-scoped styler up front. Command handlers should
// prefer this constructor so explicit flags like --no-color propagate
// consistently instead of falling back to writer auto-detection.
func NewStyledTable(styler Styler, headers ...string) *Table {
	tbl := NewTable(headers...)
	tbl.SetStyler(styler)
	return tbl
}

// SetStyler overrides the table's header styling. Command handlers should pass
// opts.Style() here (or use NewStyledTable) so explicit output flags keep
// working even when stdout still looks like a capable terminal.
func (t *Table) SetStyler(styler Styler) {
	t.styler = styler
	t.styled = true
}

func (t *Table) AddRow(values ...string) {
	t.rows = append(t.rows, values)
}

func (t *Table) Render(w io.Writer) error {
	if len(t.rows) == 0 {
		return nil
	}

	widths := make([]int, len(t.headers))
	for i, h := range t.headers {
		widths[i] = visibleWidth(h)
	}
	for _, row := range t.rows {
		for i, cell := range row {
			if i >= len(widths) {
				break
			}

			cellWidth := visibleWidth(cell)
			if cellWidth > widths[i] {
				widths[i] = cellWidth
			}
		}
	}

	if width, ok := terminalWidth(w); ok {
		widths = clampTableWidths(t.headers, widths, width)
	}
	styler := NewStylerForWriter(w, false)
	if t.styled {
		styler = t.styler
	}
	theme := NewTheme(styler.Enabled())
	rightAlign := detectNumericColumns(t.rows, len(widths))

	// Header
	for i, h := range t.headers {
		if i > 0 {
			if _, err := fmt.Fprint(w, tableColumnSeparator); err != nil {
				return err
			}
		}
		if _, err := fmt.Fprint(w, styler.Bold(fitCellAligned(h, widths[i], false))); err != nil {
			return err
		}
	}
	if _, err := fmt.Fprintln(w); err != nil {
		return err
	}

	// Subtle separator line under the header.
	for i, width := range widths {
		if i > 0 {
			if _, err := fmt.Fprint(w, tableColumnSeparator); err != nil {
				return err
			}
		}
		seg := strings.Repeat(tableSeparatorRune, width)
		if styler.Enabled() {
			seg = theme.Muted(seg)
		}
		if _, err := fmt.Fprint(w, seg); err != nil {
			return err
		}
	}
	if _, err := fmt.Fprintln(w); err != nil {
		return err
	}

	// Rows
	for _, row := range t.rows {
		for i, cell := range row {
			if i >= len(widths) {
				break
			}
			if i > 0 {
				if _, err := fmt.Fprint(w, tableColumnSeparator); err != nil {
					return err
				}
			}
			if _, err := fmt.Fprint(w, fitCellAligned(cell, widths[i], rightAlign[i])); err != nil {
				return err
			}
		}
		if _, err := fmt.Fprintln(w); err != nil {
			return err
		}
	}
	return nil
}

func detectNumericColumns(rows [][]string, columns int) []bool {
	right := make([]bool, columns)
	for col := 0; col < columns; col++ {
		hasValue := false
		allNumeric := true
		for _, row := range rows {
			if col >= len(row) {
				continue
			}
			cell := strings.TrimSpace(stripANSI(row[col]))
			if cell == "" || cell == "-" {
				continue
			}
			hasValue = true
			if !numericCellPattern.MatchString(cell) {
				allNumeric = false
				break
			}
		}
		right[col] = hasValue && allNumeric
	}
	return right
}

func clampTableWidths(headers []string, widths []int, maxWidth int) []int {
	if len(widths) == 0 {
		return widths
	}

	available := maxWidth - len(tableColumnSeparator)*(len(widths)-1)
	if available < len(widths) {
		available = len(widths)
	}
	if sumWidths(widths) <= available {
		return widths
	}

	minWidths := make([]int, len(widths))
	for i, header := range headers {
		headerWidth := visibleWidth(header)
		if headerWidth < minTableColumnWidth {
			headerWidth = minTableColumnWidth
		}
		if headerWidth > maxHeaderMinimumWidth {
			headerWidth = maxHeaderMinimumWidth
		}
		minWidths[i] = headerWidth
	}

	if sumWidths(minWidths) > available {
		return distributeWidths(len(widths), available)
	}

	clamped := append([]int(nil), widths...)
	for sumWidths(clamped) > available {
		idx := widestColumn(clamped, minWidths)
		if idx < 0 {
			break
		}
		clamped[idx]--
	}
	return clamped
}

func visibleWidth(s string) int {
	return runewidth.StringWidth(stripANSI(s))
}

func stripANSI(s string) string {
	return ansiSequencePattern.ReplaceAllString(s, "")
}

func fitCell(s string, width int) string {
	return fitCellAligned(s, width, false)
}

func fitCellAligned(s string, width int, right bool) string {
	if width <= 0 {
		return ""
	}

	value := truncateCell(s, width)
	currentWidth := visibleWidth(value)
	if currentWidth >= width {
		return value
	}
	pad := strings.Repeat(" ", width-currentWidth)
	if right {
		return pad + value
	}
	return value + pad
}

func truncateCell(s string, width int) string {
	if width <= 0 {
		return ""
	}
	if visibleWidth(s) <= width {
		return s
	}
	if width <= visibleWidth(tableTruncationEllipsis) {
		return tableTruncationEllipsis
	}

	remaining := width - visibleWidth(tableTruncationEllipsis)
	var b strings.Builder
	hasANSI := false

	for i := 0; i < len(s); {
		if seq, ok := ansiSequenceAt(s, i); ok {
			b.WriteString(seq)
			hasANSI = true
			i += len(seq)
			continue
		}

		r, size := utf8.DecodeRuneInString(s[i:])
		runeWidth := runewidth.RuneWidth(r)
		if runeWidth > remaining {
			break
		}
		b.WriteString(s[i : i+size])
		remaining -= runeWidth
		i += size
	}

	b.WriteString(tableTruncationEllipsis)
	if hasANSI && !strings.HasSuffix(b.String(), "\033[0m") {
		b.WriteString("\033[0m")
	}
	return b.String()
}

func ansiSequenceAt(s string, start int) (string, bool) {
	if start+2 > len(s) || s[start] != '\x1b' || s[start+1] != '[' {
		return "", false
	}
	for i := start + 2; i < len(s); i++ {
		if s[i] == 'm' {
			return s[start : i+1], true
		}
	}
	return "", false
}

func sumWidths(widths []int) int {
	total := 0
	for _, width := range widths {
		total += width
	}
	return total
}

func widestColumn(widths, minWidths []int) int {
	idx := -1
	for i, width := range widths {
		if width <= minWidths[i] {
			continue
		}
		if idx < 0 || width > widths[idx] {
			idx = i
		}
	}
	return idx
}

func distributeWidths(columns, available int) []int {
	widths := make([]int, columns)
	for i := range widths {
		widths[i] = 1
	}
	remaining := available - columns
	for i := 0; remaining > 0; i = (i + 1) % columns {
		widths[i]++
		remaining--
	}
	return widths
}
