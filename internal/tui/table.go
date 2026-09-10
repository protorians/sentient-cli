package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Table renders a simple bordered align-left table.
type Table struct {
	Headers []string
	Rows    [][]string
	styles  *Styles
}

// NewTable allocates a table.
func NewTable(headers []string) *Table {
	return &Table{Headers: headers, styles: NewStyles()}
}

// AddRow appends a row (length must match len(headers)).
func (t *Table) AddRow(cells ...string) {
	if len(cells) != len(t.Headers) {
		return
	}
	row := make([]string, len(cells))
	copy(row, cells)
	t.Rows = append(t.Rows, row)
}

// Render produces the printable table.
func (t *Table) Render() string {
	widths := make([]int, len(t.Headers))
	for i, h := range t.Headers {
		widths[i] = lipgloss.Width(h)
	}
	for _, row := range t.Rows {
		for i, cell := range row {
			if w := lipgloss.Width(cell); w > widths[i] {
				widths[i] = w
			}
		}
	}

	var b strings.Builder
	writeRow := func(cells []string, style lipgloss.Style) {
		b.WriteString("│ ")
		for i, cell := range cells {
			b.WriteString(style.Render(lipgloss.NewStyle().Width(widths[i]).Render(cell)))
			if i < len(cells)-1 {
				b.WriteString(" │ ")
			}
		}
		b.WriteString(" │\n")
	}
	writeSep := func() {
		b.WriteString("├")
		for i := range t.Headers {
			b.WriteString(strings.Repeat("─", widths[i]+2))
			if i < len(t.Headers)-1 {
				b.WriteString("┼")
			}
		}
		b.WriteString("┤\n")
	}

	b.WriteString("┌")
	for i := range t.Headers {
		b.WriteString(strings.Repeat("─", widths[i]+2))
		if i < len(t.Headers)-1 {
			b.WriteString("┬")
		}
	}
	b.WriteString("┐\n")
	writeRow(t.Headers, t.styles.TableHeader)
	writeSep()
	for _, row := range t.Rows {
		styled := make([]string, len(row))
		copy(styled, row)
		writeRow(styled, t.styles.TableRow)
	}
	b.WriteString("└")
	for i := range t.Headers {
		b.WriteString(strings.Repeat("─", widths[i]+2))
		if i < len(t.Headers)-1 {
			b.WriteString("┴")
		}
	}
	b.WriteString("┘\n")
	return b.String()
}
