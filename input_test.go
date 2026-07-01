package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestInputSetHandlerUpdatesOriginalModel(t *testing.T) {
	model := NewInput("title")
	returned := model.SetHandler(func() {})

	if returned != model {
		t.Fatalf("SetHandler returned a different model pointer")
	}
	if model.handler == nil {
		t.Fatalf("SetHandler did not update the original model")
	}
}

func TestInputEnterDefersHandlerUntilRunCompletes(t *testing.T) {
	model := NewInput("title")
	called := 0
	model = model.SetHandler(func() {
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
		t.Fatalf("handler call count = %d, want 1 after pending handler runs", called)
	}
	if model.handlerPending {
		t.Fatalf("handlerPending = true, want false after pending handler runs")
	}
}
