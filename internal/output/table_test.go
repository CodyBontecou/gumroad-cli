package output

import (
	"bytes"
	"errors"
	"strings"
	"testing"
)

type widthAwareBuffer struct {
	bytes.Buffer
	width int
}

func (b *widthAwareBuffer) terminalWidth() (int, bool) {
	return b.width, true
}

func TestTable_Empty(t *testing.T) {
	var buf bytes.Buffer
	tbl := NewTable("A", "B")
	if err := tbl.Render(&buf); err != nil {
		t.Fatalf("Render failed: %v", err)
	}
	if buf.Len() != 0 {
		t.Errorf("expected empty output for no rows, got %q", buf.String())
	}
}

func TestTable_Render(t *testing.T) {
	var buf bytes.Buffer
	tbl := NewTable("ID", "NAME")
	tbl.AddRow("1", "Alpha")
	tbl.AddRow("22", "Beta")
	if err := tbl.Render(&buf); err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	lines := strings.Split(strings.TrimRight(buf.String(), "\n"), "\n")
	if len(lines) != 3 {
		t.Fatalf("expected 3 lines (header + 2 rows), got %d: %q", len(lines), buf.String())
	}

	// Header should have both column names
	if !strings.Contains(lines[0], "ID") || !strings.Contains(lines[0], "NAME") {
		t.Errorf("header missing columns: %q", lines[0])
	}

	// Data rows
	if !strings.Contains(lines[1], "1") || !strings.Contains(lines[1], "Alpha") {
		t.Errorf("row 1 missing data: %q", lines[1])
	}
	if !strings.Contains(lines[2], "22") || !strings.Contains(lines[2], "Beta") {
		t.Errorf("row 2 missing data: %q", lines[2])
	}
}

func TestTable_Alignment(t *testing.T) {
	var buf bytes.Buffer
	tbl := NewTable("ID", "NAME")
	tbl.AddRow("x", "short")
	tbl.AddRow("xxx", "longer name")
	if err := tbl.Render(&buf); err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	lines := strings.Split(strings.TrimRight(buf.String(), "\n"), "\n")
	// All rows should have same structure (columns padded)
	for _, line := range lines {
		if !strings.Contains(line, "  ") {
			t.Errorf("expected column separator in line %q", line)
		}
	}
}

func TestTable_AlignmentWithANSIColor(t *testing.T) {
	style := Styler{enabled: true}

	var buf bytes.Buffer
	tbl := NewTable("NAME", "STATUS", "PRICE")
	tbl.SetStyler(style)
	tbl.AddRow("Testing", style.Yellow("draft"), "$10")
	if err := tbl.Render(&buf); err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	lines := strings.Split(strings.TrimRight(buf.String(), "\n"), "\n")
	if len(lines) != 2 {
		t.Fatalf("expected 2 lines, got %d: %q", len(lines), buf.String())
	}

	header := stripANSI(lines[0])
	row := stripANSI(lines[1])
	if strings.Index(header, "PRICE") != strings.Index(row, "$10") {
		t.Fatalf("expected PRICE and $10 to share a column start:\nheader: %q\nrow:    %q", header, row)
	}
}

func TestNewStyledTable_UsesProvidedStyler(t *testing.T) {
	style := Styler{enabled: true}

	var buf bytes.Buffer
	tbl := NewStyledTable(style, "ID")
	tbl.AddRow("one")
	if err := tbl.Render(&buf); err != nil {
		t.Fatalf("Render failed: %v", err)
	}
	if !strings.Contains(buf.String(), "\033[1mID ") || !strings.Contains(buf.String(), "\033[0m") {
		t.Fatalf("expected styled header output, got %q", buf.String())
	}
}

func TestTable_AlignmentWithWideCharacters(t *testing.T) {
	var buf bytes.Buffer
	tbl := NewTable("NAME", "STATUS")
	tbl.AddRow("商品", "ok")
	if err := tbl.Render(&buf); err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	lines := strings.Split(strings.TrimRight(buf.String(), "\n"), "\n")
	if len(lines) != 2 {
		t.Fatalf("expected 2 lines, got %d: %q", len(lines), buf.String())
	}

	header := stripANSI(lines[0])
	row := stripANSI(lines[1])
	headerStart := strings.Index(header, "STATUS")
	rowStart := strings.Index(row, "ok")
	if headerStart < 0 || rowStart < 0 {
		t.Fatalf("expected STATUS and ok in output:\nheader: %q\nrow:    %q", header, row)
	}
	if visibleWidth(header[:headerStart]) != visibleWidth(row[:rowStart]) {
		t.Fatalf("expected STATUS and ok to share a display column start:\nheader: %q\nrow:    %q", header, row)
	}
}

func TestTable_ExtraCellsTruncated(t *testing.T) {
	var buf bytes.Buffer
	tbl := NewTable("A")
	tbl.AddRow("val1", "extra-ignored")
	if err := tbl.Render(&buf); err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	out := buf.String()
	if strings.Contains(out, "extra-ignored") {
		t.Errorf("extra cells should be truncated: %q", out)
	}
	if !strings.Contains(out, "val1") {
		t.Errorf("first cell should be present: %q", out)
	}
}

func TestTable_SingleColumn(t *testing.T) {
	var buf bytes.Buffer
	tbl := NewTable("NAME")
	tbl.AddRow("alpha")
	tbl.AddRow("beta")
	if err := tbl.Render(&buf); err != nil {
		t.Fatalf("Render failed: %v", err)
	}

	lines := strings.Split(strings.TrimRight(buf.String(), "\n"), "\n")
	if len(lines) != 3 {
		t.Fatalf("expected 3 lines, got %d", len(lines))
	}
}

func TestTable_RenderWriteError(t *testing.T) {
	tbl := NewTable("A")
	tbl.AddRow("value")

	if err := tbl.Render(errWriter{}); !errors.Is(err, errTestWrite) {
		t.Fatalf("got %v, want %v", err, errTestWrite)
	}
}

func TestClampTableWidths_ReducesWideColumns(t *testing.T) {
	headers := []string{"ID", "NAME", "URL"}
	widths := []int{6, 24, 80}

	got := clampTableWidths(headers, widths, 40)
	total := sumWidths(got) + len(tableColumnSeparator)*(len(got)-1)
	if total > 40 {
		t.Fatalf("got table width %d, want <= 40", total)
	}
	if got[2] >= widths[2] {
		t.Fatalf("expected wide column to shrink, got %v from %v", got, widths)
	}
}

func TestTruncateCell_PreservesVisibleWidth(t *testing.T) {
	got := truncateCell("this is a very long value", 10)
	if visibleWidth(got) != 10 {
		t.Fatalf("got visible width %d, want 10: %q", visibleWidth(got), got)
	}
	if !strings.Contains(got, tableTruncationEllipsis) {
		t.Fatalf("expected ellipsis in %q", got)
	}
}

func TestTruncateCell_KeepsANSIReset(t *testing.T) {
	got := truncateCell(Styler{enabled: true}.Yellow("this status is very long"), 8)
	if visibleWidth(got) != 8 {
		t.Fatalf("got visible width %d, want 8: %q", visibleWidth(got), got)
	}
	if !strings.HasSuffix(got, "\033[0m") {
		t.Fatalf("expected ANSI reset suffix in %q", got)
	}
}

func TestClampTableWidths_DistributesTinyWidth(t *testing.T) {
	headers := []string{"IDENTIFIER", "LONG HEADER", "URL"}
	widths := []int{20, 20, 20}

	got := clampTableWidths(headers, widths, 8)
	total := sumWidths(got) + len(tableColumnSeparator)*(len(got)-1)
	if total > 8 {
		t.Fatalf("got table width %d, want <= 8", total)
	}
	for _, width := range got {
		if width < 1 {
			t.Fatalf("expected each width to be at least 1, got %v", got)
		}
	}
}

func TestANSISequenceAt(t *testing.T) {
	seq, ok := ansiSequenceAt("\033[33mtext", 0)
	if !ok || seq != "\033[33m" {
		t.Fatalf("got seq=%q ok=%v", seq, ok)
	}
	if _, ok := ansiSequenceAt("plain", 0); ok {
		t.Fatal("expected plain text to have no ANSI sequence")
	}
	if _, ok := ansiSequenceAt("\033[33", 0); ok {
		t.Fatal("expected unterminated ANSI sequence to be rejected")
	}
}

func TestFitCell_PadsShortValues(t *testing.T) {
	got := fitCell("ok", 4)
	if got != "ok  " {
		t.Fatalf("got %q, want %q", got, "ok  ")
	}
}

func TestFitCell_ZeroWidth(t *testing.T) {
	if got := fitCell("value", 0); got != "" {
		t.Fatalf("got %q, want empty string", got)
	}
}

func TestClampTableWidths_EmptyInput(t *testing.T) {
	if got := clampTableWidths(nil, nil, 20); len(got) != 0 {
		t.Fatalf("got %v, want empty widths", got)
	}
}

func TestTable_RenderDoesNotClampNonTerminalWriter(t *testing.T) {
	t.Setenv("COLUMNS", "12")

	var buf bytes.Buffer
	value := "this value should not be truncated when redirected"
	tbl := NewTable("NAME")
	tbl.AddRow(value)

	if err := tbl.Render(&buf); err != nil {
		t.Fatalf("Render failed: %v", err)
	}
	if !strings.Contains(buf.String(), value) {
		t.Fatalf("expected full value in redirected output, got %q", buf.String())
	}
	if strings.Contains(buf.String(), tableTruncationEllipsis) {
		t.Fatalf("did not expect truncation ellipsis in %q", buf.String())
	}
}

func TestTable_RenderClampsWidthAwareWriter(t *testing.T) {
	buf := &widthAwareBuffer{width: 12}
	tbl := NewTable("NAME")
	tbl.AddRow("this value should be truncated")

	if err := tbl.Render(buf); err != nil {
		t.Fatalf("Render failed: %v", err)
	}
	if !strings.Contains(buf.String(), tableTruncationEllipsis) {
		t.Fatalf("expected truncated output, got %q", buf.String())
	}
}
