package readline

import (
	"bytes"
	"strings"
	"testing"

	rlterm "github.com/chainreactors/tui/readline/terminal"
)

func TestNewShellWithTerminalUsesCustomInput(t *testing.T) {
	term := rlterm.Stream(strings.NewReader("alpha<<END>>beta"), ioDiscard{}, ioDiscard{}, rlterm.NewControl(false, 40, 10))
	rl := NewShellWithTerminal(term)

	got := string(rl.Keys.ReadUntilSequence([]byte("<<END>>")))
	if got != "alpha" {
		t.Fatalf("ReadUntilSequence read %q, want %q", got, "alpha")
	}

	rest := string(rl.Keys.Read())
	if rest != "beta" {
		t.Fatalf("remaining buffered input = %q, want %q", rest, "beta")
	}
}

func TestShellPrintfUsesTerminalOutput(t *testing.T) {
	var out bytes.Buffer
	term := rlterm.Stream(strings.NewReader(""), &out, &out, rlterm.NewControl(false, 40, 10))
	rl := NewShellWithTerminal(term)

	if _, err := rl.Printf("hello %s", "terminal"); err != nil {
		t.Fatalf("Printf returned error: %v", err)
	}

	if !strings.Contains(out.String(), "hello terminal") {
		t.Fatalf("terminal output %q does not contain formatted message", out.String())
	}
}

func TestInlineSuggestionAPI(t *testing.T) {
	rl := NewShellWithTerminal(rlterm.Stream(strings.NewReader(""), ioDiscard{}, ioDiscard{}, rlterm.NewControl(false, 40, 10)))

	rl.SetInlineSuggestion("status --verbose")
	if got := rl.GetInlineSuggestion(); got != "status --verbose" {
		t.Fatalf("inline suggestion = %q", got)
	}

	rl.ClearInlineSuggestion()
	if got := rl.GetInlineSuggestion(); got != "" {
		t.Fatalf("cleared inline suggestion = %q, want empty", got)
	}
}

type ioDiscard struct{}

func (ioDiscard) Write(p []byte) (int, error) { return len(p), nil }
