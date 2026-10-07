//go:build windows

package core

import (
	"io"
	"os"
	"strings"
	"syscall"
	"testing"
	"unsafe"

	"github.com/chainreactors/tui/readline/internal/term"
)

func TestStreamCursorDoesNotQueryHostConsole(t *testing.T) {
	old := kernel.GetConsoleScreenBufferInfo
	t.Cleanup(func() { kernel.GetConsoleScreenBufferInfo = old })
	kernel.GetConsoleScreenBufferInfo = func(...uintptr) error {
		t.Fatal("stream terminal queried the host console")
		return nil
	}
	k := new(Keys)
	k.SetInput(strings.NewReader(""))
	restore := term.Activate(io.Discard, nil)
	defer restore()
	if x, y := k.GetCursorPos(); x != -1 || y != -1 {
		t.Fatalf("stream cursor = %d,%d, want unavailable", x, y)
	}
}

func TestNativeCursorHandlesQueryFailure(t *testing.T) {
	old := kernel.GetConsoleScreenBufferInfo
	t.Cleanup(func() { kernel.GetConsoleScreenBufferInfo = old })
	k := new(Keys)
	k.SetInput(os.Stdin)
	restore := term.Activate(os.Stdout, nil)
	defer restore()
	kernel.GetConsoleScreenBufferInfo = func(...uintptr) error { return syscall.EINVAL }
	if x, y := k.GetCursorPos(); x != -1 || y != -1 {
		t.Fatalf("failed cursor query = %d,%d, want unavailable", x, y)
	}
	kernel.GetConsoleScreenBufferInfo = func(args ...uintptr) error {
		info := (*_CONSOLE_SCREEN_BUFFER_INFO)(unsafe.Pointer(args[1]))
		info.dwCursorPosition = _COORD{x: 8, y: 3}
		return nil
	}
	if x, y := k.GetCursorPos(); x != 9 || y != 3 {
		t.Fatalf("native cursor = %d,%d, want 9,3", x, y)
	}
}
