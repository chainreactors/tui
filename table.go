package tui

import (
	"bytes"
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/evertras/bubble-table/table"
	"golang.org/x/term"
)

var (
	styleBase = lipgloss.NewStyle().
			Align(lipgloss.Left)

	borderNone = table.Border{
		Top:            "",
		Bottom:         "─",
		Left:           "",
		Right:          "",
		TopLeft:        "",
		TopRight:       "",
		BottomLeft:     "",
		BottomRight:    "",
		TopJunction:    "",
		BottomJunction: " ",
		LeftJunction:   "",
		RightJunction:  "",
		InnerJunction:  " ",
		InnerDivider:   " ",
	}
)

const defaultTermWidth = 120

func getTerminalWidth() int {
	w, _, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil || w <= 0 {
		return defaultTermWidth
	}
	return w
}

func NewTable(columns []table.Column, isStatic bool) *TableModel {
	var newTable = table.Model{}
	termWidth := getTerminalWidth()
	if isStatic {
		newTable = table.New(columns).WithFooterVisibility(false).
			Border(borderNone).WithBaseStyle(styleBase).
			WithTargetWidth(termWidth)
	} else {
		newTable = table.New(columns).Filtered(true).
			Border(borderNone).Focused(true).WithPageSize(10).WithBaseStyle(styleBase).
			WithTargetWidth(termWidth)
	}
	keyMap := table.DefaultKeyMap()
	keyMap.RowSelectToggle = key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("enter", "select row"),
	)
	newTable = newTable.WithKeyMap(keyMap)
	newTable = newTable.HighlightStyle(SelectStyle)
	t := &TableModel{
		table:       newTable,
		Columns:     columns,
		rowsPerPage: 10,
		isStatic:    isStatic,
	}
	return t
}

type TableModel struct {
	table   table.Model
	Columns []table.Column
	Rows    []table.Row
	AllRows []table.Row
	*bytes.Buffer
	currentPage    int
	totalPages     int
	rowsPerPage    int
	isStatic       bool
	filtered       bool
	handle         func()
	Title          string
	selected       table.Row
	highlightRows  []int
	searchString   string
	highlightStyle lipgloss.Style
}

func (t *TableModel) UpdatePagination() {
	t.totalPages = (len(t.Rows) + t.rowsPerPage - 1) / t.rowsPerPage
	if t.currentPage > t.totalPages {
		t.currentPage = t.totalPages
	}
	if t.currentPage < 1 {
		t.currentPage = 1
	}
}

func (t *TableModel) Init() tea.Cmd { return nil }

func (t *TableModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		t.table = t.table.WithTargetWidth(msg.Width)
		return t, nil
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEsc, tea.KeyCtrlQ:
			if len(t.highlightRows) > 0 {
				t.CleanHighlight()
				t.SetRows(t.AllRows)
				return t, nil
			}
			return t, tea.Quit
		case tea.KeyEnter:
			t.selected = t.GetHighlightedRow()
			if t.handle == nil {
				return t, tea.Quit
			}
			t.handleSelectedRow()
			return t, tea.Quit
		}
	}
	t.table, cmd = t.table.Update(msg)
	return t, tea.Batch(cmd)
}

func (t *TableModel) View() string {
	if t.isStatic {
		t.table.WithPageSize(len(t.Rows))
		return fmt.Sprintf("%s\n", t.Title) + "\n" + t.table.View() + "\n"
	}
	return fmt.Sprintf("%s\n", t.Title) + "\n" + t.table.View() + "\n"
}

func (t *TableModel) SetRows(rows []table.Row) {
	t.Rows = rows
	t.autoFitColumns(rows)
	t.table = t.table.WithRows(rows)
}

// autoFitColumns calculates optimal column widths based on actual cell content,
// then replaces columns with fixed-width columns sized to fit.
// Long content is capped per-column and left to multiline word-wrap.
func (t *TableModel) autoFitColumns(rows []table.Row) {
	numCols := len(t.Columns)
	if numCols == 0 || len(rows) == 0 {
		return
	}

	termWidth := getTerminalWidth()
	const padding = 1
	const minWidth = 6
	// InnerDivider is 1 char between columns
	dividerOverhead := numCols - 1
	availableWidth := termWidth - dividerOverhead

	// Per-column max: adaptive based on column count
	// Allow up to 2x average width, but at least 20% of terminal
	maxColWidth := max(termWidth/numCols*2, termWidth*20/100)

	// Calculate optimal width for each column: max(header, max_cell_content) + padding
	widths := make([]int, numCols)
	capped := make([]bool, numCols)
	for i, col := range t.Columns {
		widths[i] = lipgloss.Width(col.Title()) + padding

		key := col.Key()
		for _, row := range rows {
			if capped[i] {
				break
			}
			val, ok := row.Data[key]
			if !ok {
				continue
			}
			s := fmt.Sprint(val)
			// Handle multiline cells: use the widest line
			for _, line := range strings.Split(s, "\n") {
				if w := lipgloss.Width(line) + padding; w > widths[i] {
					widths[i] = w
				}
			}
			// Early termination: cap reached, no need to scan more rows
			if widths[i] >= maxColWidth {
				widths[i] = maxColWidth
				capped[i] = true
			}
		}
	}

	total := 0
	for _, w := range widths {
		total += w
	}

	if total > availableWidth {
		// Proportionally compress, respecting a minimum width
		ratio := float64(availableWidth) / float64(total)
		newTotal := 0
		for i := range widths {
			widths[i] = max(int(float64(widths[i])*ratio), minWidth)
			newTotal += widths[i]
		}
		// Distribute leftover space to the columns that lost the most
		remaining := availableWidth - newTotal
		for remaining > 0 {
			bestIdx, bestLoss := 0, 0
			for i := range widths {
				orig := int(float64(widths[i]) / ratio)
				loss := orig - widths[i]
				if loss > bestLoss {
					bestLoss = loss
					bestIdx = i
				}
			}
			if bestLoss == 0 {
				break
			}
			widths[bestIdx]++
			remaining--
		}
	}

	// Rebuild columns with calculated widths
	newCols := make([]table.Column, numCols)
	for i, col := range t.Columns {
		newCols[i] = table.NewColumn(col.Key(), col.Title(), widths[i])
		if col.Filterable() {
			newCols[i] = newCols[i].WithFiltered(true)
		}
	}

	t.Columns = newCols
	t.table = t.table.WithColumns(newCols).WithTargetWidth(termWidth)
}

func (t *TableModel) handleSelectedRow() {
	t.handle()
}

func (t *TableModel) SetHandle(handle func()) {
	t.handle = handle
}

func (t *TableModel) CleanHighlight() {
	t.highlightRows = []int{}
	t.searchString = ""
}

func (t *TableModel) SetMultiline() {
	t.table = t.table.WithMultiline(true)
}

func (t *TableModel) GetSelectedRow() table.Row {
	selectedRow := t.selected
	return selectedRow
}

func (t *TableModel) GetHighlightedRow() table.Row {
	selectedRow := t.table.HighlightedRow()
	return selectedRow
}
func (t *TableModel) pageView(s string) string {
	style := lipgloss.NewStyle().Inline(true).Align(lipgloss.Left)
	return style.Render(s)
}

func (t *TableModel) SetAscSort(s string) {
	t.table = t.table.SortByAsc(s)
}

func (t *TableModel) SetDescSort(s string) {
	t.table = t.table.SortByDesc(s)
}

func (t *TableModel) Run() error {
	p := tea.NewProgram(t)
	_, err := p.Run()
	if err != nil {
		return err
	}
	fmt.Printf(HelpStyle("<Press enter to exit>\n"))
	os.Stdin.Write([]byte("\n"))
	ClearLines(1)
	return nil
}
