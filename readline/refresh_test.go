package readline

import (
	"bytes"
	"errors"
	"io"
	"regexp"
	"strings"
	"testing"

	"github.com/chainreactors/tui/readline/internal/display"
	rlterm "github.com/chainreactors/tui/readline/terminal"
)

func TestPrimaryRedrawKeepsWrappedDraftBelowOutput(t *testing.T) {
	for name, redraw := range map[string]func(*Shell){
		"output":           func(rl *Shell) { _, _ = rl.Printf("committed") },
		"transient output": func(rl *Shell) { _, _ = rl.PrintTransientf("committed") },
		"status":           func(rl *Shell) { rl.RefreshPrimaryWithoutAutocomplete() },
	} {
		t.Run(name, func(t *testing.T) {
			var output bytes.Buffer
			terminal := rlterm.Stream(strings.NewReader(""), &output, &output, rlterm.NewControl(false, 12, 24))
			rl := NewShellWithTerminal(terminal)
			rl.Prompt.Primary(func() string { return "draft> " })
			const draft = "keep this wrapped draft"
			rl.Line().Set([]rune(draft)...)
			rl.Cursor().Set(len(draft))
			display.Init(rl.Display, nil)
			rl.RefreshWithoutAutocomplete()
			output.Reset()

			redraw(rl)

			// Once the full prompt has been printed at its new position, the
			// editor must not backtrack into the output above it to print the draft.
			_, refresh, ok := strings.Cut(output.String(), "\x1b[?25l")
			beforePrompt, _, hasPrompt := strings.Cut(refresh, "draft> ")
			if !ok || !hasPrompt {
				t.Fatalf("missing prompt redraw: %q", output.String())
			}
			if regexp.MustCompile("\x1b\\[[0-9]+A").MatchString(beforePrompt) {
				t.Fatalf("draft redraw moved above the new prompt: %q", beforePrompt)
			}
			if string(*rl.Line()) != draft || rl.Cursor().Pos() != len(draft) {
				t.Fatal("redraw changed the draft or editing cursor")
			}
		})
	}
}

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
