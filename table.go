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
	AllRows        []table.Row
	currentPage    int
	totalPages     int
	rowsPerPage    int
	isStatic       bool
	isConsole      bool
	handle         func()
	Title          string
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
			if len(t.highlightRows) > 0 {
				t.CleanHighlight()
				t.SetRows(t.AllRows)
				return t, nil
			}
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
		return fmt.Sprintf("%s\n", t.Title) + "\n" + HeaderStyle.Render(t.table.View()) + "\n" +
			t.pageView(fmt.Sprintf("\nPage %d of %d", t.currentPage, t.totalPages)+
				t.searchView(t.searchString+"\n"))
	}
	return fmt.Sprintf("%s\n", t.Title) + "\n" + HeaderStyle.Render(t.table.View()) + "\n" +
		t.pageView(fmt.Sprintf("\nPage %d of %d\n", t.currentPage, t.totalPages))
}

func (t *TableModel) SetRows(rows []table.Row) {
	t.Rows = rows
	if t.isStatic {
		t.table.SetHeight(len(t.Rows) + 1)
		t.rowsPerPage = len(t.Rows)
		t.handle = func() {
			t.Update(tea.Quit())
		}
	}
	t.table.SetRows(t.Rows)
	t.totalPages = len(t.Rows) / t.rowsPerPage
	t.table.SetHeight(t.rowsPerPage + 1)
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
	t.searchString = ""
}

func (t *TableModel) GetSelectedRow() table.Row {
	selectedRow := t.table.SelectedRow()
	return selectedRow
}

func (t *TableModel) ConsoleHandler(value string) {
	if strings.HasPrefix(value, "/") {
		searchString := strings.TrimPrefix(value, "/")
		t.highlightRows = t.searchRows(searchString)
		var rows []table.Row
		for _, i := range t.highlightRows {
			rows = append(rows, t.Rows[i])
		}
		t.AllRows = t.Rows
		t.searchString = "searchString: " + searchString
		t.SetRows(rows)
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

func (t *TableModel) searchView(s string) string {
	var width int
	for i := range t.Columns {
		width = width + t.Columns[i].Width
	}
	style := lipgloss.NewStyle().Width(width).MaxWidth(width).Inline(true).Align(lipgloss.Right)
	return t.Style.Cell.Render(style.Render(runewidth.Truncate(s, width, "…")))
}

func (t *TableModel) pageView(s string) string {
	style := lipgloss.NewStyle().Inline(true).Align(lipgloss.Left)
	return style.Render(s)
}
