package term

import (
	"fmt"
	"io"
	"os"
	"runtime"
	"strconv"
	"strings"
	"sync"
)

// termFile is the file descriptor used for all low-level terminal queries (size,
// cursor position) and escape sequences. We deliberately use stderr rather than
// stdout: stdout is the stream most likely to be redirected (e.g. `app | other`),
// whereas stderr usually stays attached to the controlling terminal, giving a
// reliable terminal size even when the program's output is piped.
var termFile = os.Stderr

// fallback terminal width when we can't get it through query.
var defaultTermWidth = 80

// Control is the terminal capability subset readline internals need.
type Control interface {
	IsTerminal() bool
	Size() (cols, rows int)
	OnResize(func(cols, rows int)) func()
}

var (
	outputs  sync.Map
	controls sync.Map
)

// Activate binds output/control to the current goroutine until the returned
// restore function is called.
func Activate(output io.Writer, control Control) func() {
	id := goid()
	oldOutput, hadOutput := outputs.Load(id)
	oldControl, hadControl := controls.Load(id)

	if output != nil {
		outputs.Store(id, output)
	}
	if control != nil {
		controls.Store(id, control)
	}

	return func() {
		if hadOutput {
			outputs.Store(id, oldOutput)
		} else {
			outputs.Delete(id)
		}

		if hadControl {
			controls.Store(id, oldControl)
		} else {
			controls.Delete(id)
		}
	}
}

// Output returns the writer bound to the current readline session.
func Output() io.Writer {
	if output, ok := outputs.Load(goid()); ok {
		if w, ok := output.(io.Writer); ok && w != nil {
			return w
		}
	}

	return os.Stdout
}

// CurrentControl returns the control bound to the current readline session.
func CurrentControl() Control {
	if control, ok := controls.Load(goid()); ok {
		if c, ok := control.(Control); ok {
			return c
		}
	}

	return nil
}

// Print writes to the active terminal output.
func Print(a ...interface{}) (int, error) {
	return fmt.Fprint(Output(), a...)
}

// Println writes to the active terminal output.
func Println(a ...interface{}) (int, error) {
	return fmt.Fprintln(Output(), a...)
}

// GetWidth returns the width of the terminal or 80 if it cannot be established.
func GetWidth() (termWidth int) {
	if control := CurrentControl(); control != nil {
		width, _ := control.Size()
		if width > 0 {
			return width
		}
	}

	var err error
	fd := int(termFile.Fd())
	termWidth, _, err = GetSize(fd)

	if err != nil || termWidth == 0 {
		termWidth = defaultTermWidth
	}

	return
}

// GetLength returns the length of the terminal
// (Y length), or 80 if it cannot be established.
func GetLength() int {
	if control := CurrentControl(); control != nil {
		_, length := control.Size()
		if length > 0 {
			return length
		}
	}

	termFd := int(termFile.Fd())

	_, length, err := GetSize(termFd)
	if err != nil || length == 0 {
		return defaultTermWidth
	}

	return length
}

func printf(format string, a ...interface{}) {
	WriteString(fmt.Sprintf(format, a...))
}

// OnResize registers a resize callback on the active terminal control.
func OnResize(fn func(cols, rows int)) func() {
	if control := CurrentControl(); control != nil {
		return control.OnResize(fn)
	}

	return func() {}
}

// EnableBracketedPaste enables bracketed paste mode.
func EnableBracketedPaste() {
	WriteString(BracketedPasteStart)
}

// DisableBracketedPaste disables bracketed paste mode.
func DisableBracketedPaste() {
	WriteString(BracketedPasteEnd)
}

func goid() uint64 {
	var buf [64]byte
	n := runtime.Stack(buf[:], false)
	fields := strings.Fields(string(buf[:n]))
	if len(fields) < 2 {
		return 0
	}

	id, err := strconv.ParseUint(fields[1], 10, 64)
	if err != nil {
		return 0
	}

	return id
}
