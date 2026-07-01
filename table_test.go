package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/evertras/bubble-table/table"
)

func TestTableEnterDefersHandlerUntilRunCompletes(t *testing.T) {
	model := NewTable([]table.Column{
		table.NewColumn("ID", "ID", 10),
	}, false)
	model.SetRows([]table.Row{
		table.NewRow(table.RowData{"ID": "row-1"}),
	})

	called := 0
	model.SetHandler(func() {
		called++
	})

	_, _ = model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if called != 0 {
		t.Fatalf("handler called during Update, want deferred execution")
	}
	if !model.handlerPending {
		t.Fatalf("handlerPending = false, want true after enter")
	}

	model.runPendingHandler()
	if called != 1 {
		t.Fatalf("handler call count = %d, want 1", called)
	}
	if model.handlerPending {
		t.Fatalf("handlerPending = true, want false after pending handler runs")
	}
}
