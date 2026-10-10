//go:build unix

package core

import (
	"io"
	"testing"
	"time"

	"github.com/chainreactors/tui/readline/internal/term"
)

func TestCursorQueryTimeoutPreservesLateStreamInput(t *testing.T) {
	reader, writer := io.Pipe()
	defer reader.Close()
	defer writer.Close()
	keys := new(Keys)
	keys.SetInput(reader)
	defer term.Activate(io.Discard, nil)()

	if x, y := keys.GetCursorPos(); x != -1 || y != -1 {
		t.Fatalf("cursor without response = %d,%d", x, y)
	}
	read := make(chan string, 1)
	go func() {
		data, _ := keys.readInputFiltered()
		read <- string(data)
	}()
	if _, err := writer.Write([]byte("late input")); err != nil {
		t.Fatal(err)
	}
	select {
	case got := <-read:
		if got != "late input" {
			t.Fatalf("input after cursor timeout = %q", got)
		}
	case <-time.After(time.Second):
		t.Fatal("the expired cursor query consumed the next keyboard input")
	}
}

func TestCursorQueryTimeoutPreservesEscapeInput(t *testing.T) {
	reader, writer := io.Pipe()
	defer reader.Close()
	defer writer.Close()
	keys := new(Keys)
	keys.SetInput(reader)
	defer term.Activate(io.Discard, nil)()
	go func() { _, _ = writer.Write([]byte("\x1b[D")) }()
	keys.GetCursorPos()
	if got := string(keys.Read()); got != "\x1b[D" {
		t.Fatalf("direction key during cursor query = %q", got)
	}
}

func TestCursorQueryPreservesInputAroundFragmentedReply(t *testing.T) {
	reader, writer := io.Pipe()
	defer reader.Close()
	defer writer.Close()
	keys := new(Keys)
	keys.SetInput(reader)
	defer term.Activate(io.Discard, nil)()
	go func() {
		_, _ = writer.Write([]byte("before\x1b[D\x1b[2;"))
		_, _ = writer.Write([]byte("9Rafter"))
	}()
	if x, y := keys.GetCursorPos(); x != 9 || y != 2 {
		t.Fatalf("cursor = %d,%d, want 9,2", x, y)
	}
	if got := string(keys.Read()); got != "before\x1b[Dafter" {
		t.Fatalf("input around cursor reply = %q", got)
	}
}

func TestStreamRefreshWakesEditorWithoutConsumingInput(t *testing.T) {
	reader, writer := io.Pipe()
	defer reader.Close()
	defer writer.Close()
	keys := new(Keys)
	keys.SetInput(reader)
	refreshed := make(chan bool, 1)
	go func() { refreshed <- WaitAvailableKeys(keys, nil) }()
	keys.RequestRefresh()
	select {
	case got := <-refreshed:
		if !got {
			t.Fatal("resize did not wake the editor for a redraw")
		}
	case <-time.After(time.Second):
		t.Fatal("resize left the editor blocked on input")
	}
	go func() { _, _ = writer.Write([]byte("keyboard")) }()
	if data, err := keys.readInputFiltered(); err != nil || string(data) != "keyboard" {
		t.Fatalf("input after resize = %q, %v", data, err)
	}
}
