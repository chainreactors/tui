package tui

import (
	"fmt"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/mattn/go-runewidth"
	"strings"
)

func NewTable(columns []table.Column, isStatic bool) *TableModel {
	t := &TableModel{
		table: table.New(
			table.WithColumns(columns),
			table.WithFocused(true)),
		Style:          DefaultTableStyle,
		Columns:        columns,
		rowsPerPage:    10,
		isStatic:       isStatic,
		highlightStyle: DefaultTableHighlineStyle,
	}
	t.table.SetStyles(DefaultTableStyle)
	return t
}

// TODO tui: table 实现自适应width 并通过左右键查看无法一次性展示的属性
type TableModel struct {
	table          table.Model
	Style          table.Styles
	Columns        []table.Column
	Rows           []table.Row
	currentPage    int
	totalPages     int
	rowsPerPage    int
	isStatic       bool
	isConsole      bool
	handle         func()
	Title          string
	highlightRows  []int
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
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, DefaultKeys.Right): // Next page
			if t.currentPage < t.totalPages {
				t.currentPage++
			}
			return t, nil
		case key.Matches(msg, DefaultKeys.Left): // Previous page
			if t.currentPage > 1 {
				t.currentPage--
			}
		}
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEsc, tea.KeyCtrlQ:
			return t, tea.Quit
		case tea.KeyEnter:
			t.handleSelectedRow()
			return t, tea.Quit
		}
		t.UpdatePagination()
	}
	t.table, cmd = t.table.Update(msg)
	return t, tea.Batch(cmd)
}

func (t *TableModel) View() string {
	startIndex := (t.currentPage - 1) * t.rowsPerPage
	endIndex := startIndex + t.rowsPerPage
	if startIndex < 0 {
		startIndex = 0
	}
	if endIndex > len(t.Rows) {
		endIndex = len(t.Rows)
	}
	t.table.SetRows(t.Rows[startIndex:endIndex])
	if len(t.highlightRows) > 0 {
		t.table.SetCursor(t.highlightRows[0])
		defer t.CleanHighlight()
		return fmt.Sprintf("%s\n", t.Title) + HeaderStyle.Render(t.headersView()+"\n"+t.highView(t.Rows)) +
			fmt.Sprintf("\nPage %d of %d\n", t.currentPage, t.totalPages)
	}
	return fmt.Sprintf("%s\n", t.Title) + "\n" + HeaderStyle.Render(t.table.View()) +
		fmt.Sprintf("\nPage %d of %d\n", t.currentPage, t.totalPages)
}

func (t *TableModel) SetRows(rows []table.Row) {
	t.Rows = rows
	if t.isStatic {
		t.handle = func() {
			t.Update(tea.Quit())
		}
	}
	t.table.SetRows(t.Rows)
	t.totalPages = len(t.Rows) / t.rowsPerPage
	if len(t.Rows)%t.rowsPerPage != 0 {
		t.totalPages++
	}
	t.currentPage = 1
}

func (t *TableModel) handleSelectedRow() {
	t.handle()
}

func (t *TableModel) SetHandle(handle func()) {
	t.handle = handle
}

func (t *TableModel) CleanHighlight() {
	t.highlightRows = []int{}
}

func (t *TableModel) GetSelectedRow() table.Row {
	selectedRow := t.table.SelectedRow()
	return selectedRow
}

func (t *TableModel) ConsoleHandler(value string) {
	if strings.HasPrefix(value, "/") {
		searchString := strings.TrimPrefix(value, "/")
		t.highlightRows = t.searchRows(searchString)
		t.table.SetCursor(t.highlightRows[0])
		return
	}
}

func (t *TableModel) searchRows(searchString string) []int {
	var result []int
	for i, row := range t.Rows {
		for _, cell := range row {
			if strings.Contains(fmt.Sprintf("%v", cell), searchString) {
				result = append(result, i)
				break
			}
		}
	}
	return result
}

func contains(slice []int, item int) bool {
	for _, v := range slice {
		if v == item {
			return true
		}
	}
	return false
}

func (t *TableModel) highView(highRows []table.Row) string {
	renderedRows := make([]string, 0, len(t.Rows))
	for i := 0; i < len(highRows); i++ {
		renderedRows = append(renderedRows, t.renderRow(i, highRows))
	}

	strs := lipgloss.JoinVertical(lipgloss.Left, renderedRows...)
	strs = strings.ReplaceAll(strs, "\r\n", "\n") // normalize line endings
	lines := strings.Split(strs, "\n")

	w, h := t.table.Width(), t.table.Height()
	if sw := t.Style.Selected.GetWidth(); sw != 0 {
		w = min(w, sw)
	}
	if sh := t.Style.Selected.GetHeight(); sh != 0 {
		h = min(h, sh)
	}
	contentWidth := w - t.Style.Selected.GetHorizontalFrameSize()
	contentHeight := h - t.Style.Selected.GetVerticalFrameSize()
	contents := lipgloss.NewStyle().
		Width(contentWidth).      // pad to width.
		Height(contentHeight).    // pad to height.
		MaxHeight(contentHeight). // truncate height if taller.
		MaxWidth(contentWidth).   // truncate width if wider.
		Render(strings.Join(lines, "\n"))
	spiltContents := strings.Split(contents, "\n")
	for _, rowIndex := range t.highlightRows {
		spiltContents[rowIndex] = t.highlightStyle.Copy().
			UnsetWidth().UnsetHeight(). // Style size already applied in contents.
			Render(spiltContents[rowIndex])
	}
	contents = strings.Join(spiltContents, "\n")
	return contents
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func (t *TableModel) renderRow(rowID int, highRows []table.Row) string {
	var s = make([]string, 0, len(t.Columns))
	for i, value := range highRows[rowID] {
		style := lipgloss.NewStyle().Width(t.Columns[i].Width).MaxWidth(t.Columns[i].Width).Inline(true)
		renderedCell := t.Style.Cell.Render(style.Render(runewidth.Truncate(value, t.Columns[i].Width, "…")))
		s = append(s, renderedCell)
	}

	row := lipgloss.JoinHorizontal(lipgloss.Left, s...)

	return row
}

func (t *TableModel) headersView() string {
	var s = make([]string, 0, len(t.Columns))
	for _, col := range t.Columns {
		style := lipgloss.NewStyle().Width(col.Width).MaxWidth(col.Width).Inline(true)
		renderedCell := style.Render(runewidth.Truncate(col.Title, col.Width, "…"))
		s = append(s, t.Style.Header.Render(renderedCell))
	}
	return lipgloss.JoinHorizontal(lipgloss.Left, s...)
}
