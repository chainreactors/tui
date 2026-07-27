package readline

import (
	"bytes"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/chainreactors/tui/readline/internal/display"
	rlterm "github.com/chainreactors/tui/readline/terminal"
)

func TestRefreshWithoutAutocompleteDoesNotGenerateMenu(t *testing.T) {
	var output bytes.Buffer
	terminal := rlterm.Stream(strings.NewReader(""), &output, &output, rlterm.NewControl(false, 80, 24))
	rl := NewShellWithTerminal(terminal)
	_ = rl.Config.Set("autocomplete", true)
	calls := 0
	rl.Completer = func(_ []rune, _ int) Completions {
		calls++
		return CompleteValues("/exit", "/help")
	}
	rl.Line().Set([]rune("/")...)
	rl.Cursor().Set(1)
	display.Init(rl.Display, nil)

	rl.RefreshWithoutAutocomplete()
	if calls != 0 {
		t.Fatalf("footer refresh generated autocomplete %d times", calls)
	}

	rl.Refresh()
	if calls != 1 {
		t.Fatalf("normal refresh generated autocomplete %d times, want 1", calls)
	}
}

func TestOnReadlineReadyRunsAfterFirstDisplayRefresh(t *testing.T) {
	var output bytes.Buffer
	terminal := rlterm.Stream(strings.NewReader(""), &output, &output, rlterm.NewControl(false, 80, 24))
	rl := NewShellWithTerminal(terminal)
	readyAfterRefresh := false
	doneAfterReady := false
	rl.OnReadlineReady = func() {
		readyAfterRefresh = strings.Contains(output.String(), "\x1b[?25l")
	}
	rl.OnReadlineDone = func() {
		doneAfterReady = readyAfterRefresh
	}

	_, err := rl.Readline()
	if !errors.Is(err, io.EOF) {
		t.Fatalf("Readline() error = %v, want EOF", err)
	}
	if !readyAfterRefresh {
		t.Fatal("OnReadlineReady ran before the first display refresh")
	}
	if !doneAfterReady {
		t.Fatal("OnReadlineDone did not close the active ready lifecycle")
	}
}
