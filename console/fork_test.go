package console

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/chainreactors/tui/readline"
	rlterm "github.com/chainreactors/tui/readline/terminal"
)

func TestNewWithTerminalBindsShellAndConsoleOutput(t *testing.T) {
	var out bytes.Buffer
	term := rlterm.Stream(strings.NewReader(""), &out, &out, rlterm.NewControl(false, 80, 24))
	c := NewWithTerminal("testapp", term)

	if c.terminal != term {
		t.Fatal("console did not retain caller-provided terminal")
	}
	if c.Shell().Terminal != term {
		t.Fatal("readline shell did not retain caller-provided terminal")
	}

	c.isExecuting.Store(true)
	defer c.isExecuting.Store(false)

	if _, err := c.Printf("log %s", "message"); err != nil {
		t.Fatalf("Printf returned error: %v", err)
	}
	if got := out.String(); got != "log message" {
		t.Fatalf("terminal output = %q, want %q", got, "log message")
	}
}

func TestDisplayNewlinesUseTerminalOutput(t *testing.T) {
	var out bytes.Buffer
	term := rlterm.Stream(strings.NewReader(""), &out, &out, rlterm.NewControl(false, 80, 24))
	c := NewWithTerminal("testapp", term)

	c.NewlineBefore = true
	c.displayPreRun("cmd")
	if got := out.String(); got != "\n" {
		t.Fatalf("displayPreRun wrote %q, want newline", got)
	}

	out.Reset()
	c.NewlineAfter = true
	c.displayPostRun("cmd")
	if got := out.String(); got != "\n" {
		t.Fatalf("displayPostRun wrote %q, want newline", got)
	}
}

func TestExecuteExportedRunsCommands(t *testing.T) {
	c := NewWithTerminal("testapp", rlterm.Stream(strings.NewReader(""), ioDiscard{}, ioDiscard{}, rlterm.NewControl(false, 80, 24)))
	menu := c.ActiveMenu()

	ran := false
	menu.SetCommands(func() *cobra.Command {
		root := &cobra.Command{Use: "root"}
		root.AddCommand(&cobra.Command{
			Use: "run",
			Run: func(*cobra.Command, []string) {
				ran = true
			},
		})
		return root
	})
	menu.resetPreRun()

	if err := c.Execute(context.Background(), menu, []string{"run"}, false); err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	if !ran {
		t.Fatal("Execute did not run target command")
	}
	if c.isExecuting.Load() {
		t.Fatal("Execute left console in executing state")
	}
}

func TestCompletionInlineSuggestionBridge(t *testing.T) {
	c := NewWithTerminal("testapp", rlterm.Stream(strings.NewReader(""), ioDiscard{}, ioDiscard{}, rlterm.NewControl(false, 80, 24)))

	comps := readline.CompleteRaw([]readline.Completion{{Value: "status "}})
	comps.PREFIX = "sta"
	c.setInlineSuggestion([]rune("sta"), 3, comps)

	if got := c.Shell().GetInlineSuggestion(); got != "status" {
		t.Fatalf("inline suggestion = %q, want %q", got, "status")
	}

	c.setInlineSuggestion([]rune("sta"), 1, comps)
	if got := c.Shell().GetInlineSuggestion(); got != "" {
		t.Fatalf("inline suggestion after mid-line cursor = %q, want empty", got)
	}
}

type ioDiscard struct{}

func (ioDiscard) Write(p []byte) (int, error) { return len(p), nil }
