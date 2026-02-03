package tableprinter

import (
	"fmt"
	"io"
	"strings"

	"github.com/triptechtravel/clickup-cli/internal/iostreams"
)

// TablePrinter renders TTY-aware table output.
type TablePrinter struct {
	out        io.Writer
	maxWidth   int
	isTTY      bool
	rows       [][]string
	colWidths  []int
	truncCols  map[int]bool // columns to truncate
}

// New creates a new TablePrinter for the given IOStreams.
func New(ios *iostreams.IOStreams) *TablePrinter {
	return &TablePrinter{
		out:       ios.Out,
		maxWidth:  ios.TerminalWidth(),
		isTTY:     ios.IsTerminal(),
		truncCols: map[int]bool{},
	}
}

// AddField adds a field to the current row.
func (t *TablePrinter) AddField(text string) {
	if len(t.rows) == 0 {
		t.rows = append(t.rows, []string{})
	}
	row := len(t.rows) - 1
	col := len(t.rows[row])

	t.rows[row] = append(t.rows[row], text)

	// Track max column width
	for len(t.colWidths) <= col {
		t.colWidths = append(t.colWidths, 0)
	}
	if len(text) > t.colWidths[col] {
		t.colWidths[col] = len(text)
	}
}

// EndRow finishes the current row and starts a new one.
func (t *TablePrinter) EndRow() {
	t.rows = append(t.rows, []string{})
}

// SetTruncateColumn marks a column to be truncated if the table is too wide.
func (t *TablePrinter) SetTruncateColumn(col int) {
	t.truncCols[col] = true
}

// Render outputs the table.
func (t *TablePrinter) Render() error {
	// Remove empty trailing row
	if len(t.rows) > 0 && len(t.rows[len(t.rows)-1]) == 0 {
		t.rows = t.rows[:len(t.rows)-1]
	}

	if len(t.rows) == 0 {
		return nil
	}

	if !t.isTTY {
		return t.renderTSV()
	}

	return t.renderAligned()
}

func (t *TablePrinter) renderTSV() error {
	for _, row := range t.rows {
		fmt.Fprintln(t.out, strings.Join(row, "\t"))
	}
	return nil
}

func (t *TablePrinter) renderAligned() error {
	sep := "  "
	sepWidth := len(sep)

	// Calculate total width
	totalWidth := 0
	for i, w := range t.colWidths {
		totalWidth += w
		if i < len(t.colWidths)-1 {
			totalWidth += sepWidth
		}
	}

	// Truncate marked columns if table is too wide
	if totalWidth > t.maxWidth {
		excess := totalWidth - t.maxWidth
		for col := range t.truncCols {
			if col < len(t.colWidths) && t.colWidths[col] > 10 {
				reduce := excess
				if reduce > t.colWidths[col]-10 {
					reduce = t.colWidths[col] - 10
				}
				t.colWidths[col] -= reduce
				excess -= reduce
				if excess <= 0 {
					break
				}
			}
		}
	}

	for _, row := range t.rows {
		fields := make([]string, len(row))
		for i, field := range row {
			if i < len(t.colWidths) {
				if t.truncCols[i] && len(field) > t.colWidths[i] {
					field = field[:t.colWidths[i]-3] + "..."
				}
				if i < len(row)-1 {
					fields[i] = fmt.Sprintf("%-*s", t.colWidths[i], field)
				} else {
					fields[i] = field
				}
			} else {
				fields[i] = field
			}
		}
		fmt.Fprintln(t.out, strings.Join(fields, sep))
	}
	return nil
}
