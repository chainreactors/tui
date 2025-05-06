package tui

import (
	"fmt"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/evertras/bubble-table/table"
)

var (
	styleBase = lipgloss.NewStyle().
		Align(lipgloss.Left)
)

func NewTable(columns []table.Column, isStatic bool) *TableModel {
	var newTable = table.Model{}
	if isStatic {
		newTable = table.New(columns).WithFooterVisibility(false).
			BorderRounded().WithBaseStyle(styleBase)
	} else {
		newTable = table.New(columns).Filtered(true).
			BorderRounded().Focused(true).WithPageSize(10).WithBaseStyle(styleBase)
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
	table          table.Model
	Columns        []table.Column
	Rows           []table.Row
	AllRows        []table.Row
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
	t.table = t.table.WithRows(rows)
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
