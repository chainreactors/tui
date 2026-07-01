package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/evertras/bubble-table/table"
)

func TestTableEnterDefersHandleUntilRunCompletes(t *testing.T) {
	model := NewTable([]table.Column{
		table.NewColumn("ID", "ID", 10),
	}, false)
	model.SetRows([]table.Row{
		table.NewRow(table.RowData{"ID": "row-1"}),
	})

	called := 0
	model.SetHandle(func() {
		called++
	})

	_, _ = model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if called != 0 {
		t.Fatalf("handle called during Update, want deferred execution")
	}
	if !model.handlePending {
		t.Fatalf("handlePending = false, want true after enter")
	}

	model.runPendingHandle()
	if called != 1 {
		t.Fatalf("handle call count = %d, want 1", called)
	}
	if model.handlePending {
		t.Fatalf("handlePending = true, want false after pending handle runs")
	}
}
