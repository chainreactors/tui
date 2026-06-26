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

// Those variables are very important to realine low-level code: all virtual terminal
// escape sequences should always be sent and read through the raw terminal file, even
// if people start using io.MultiWriters and os.Pipes involving basic IO.
var (
	stdoutTerm *os.File
	stdinTerm  *os.File
	stderrTerm *os.File

	outputs  sync.Map
	controls sync.Map
)

func init() {
	stdoutTerm = os.Stdout
	stdinTerm = os.Stdin
	stderrTerm = os.Stderr
}

// fallback terminal width when we can't get it through query.
var defaultTermWidth = 80

// Control is the terminal capability subset readline internals need.
type Control interface {
	Size() (cols, rows int)
	OnResize(func(cols, rows int)) func()
}

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
	return stdoutTerm
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

// Printf writes to the active terminal output.
func Printf(format string, a ...interface{}) (int, error) {
	return fmt.Fprintf(Output(), format, a...)
}

// Println writes to the active terminal output.
func Println(a ...interface{}) (int, error) {
	return fmt.Fprintln(Output(), a...)
}

// GetWidth returns the width of Stdout or 80 if the width cannot be established.
func GetWidth() (termWidth int) {
	if control, ok := controls.Load(goid()); ok {
		if c, ok := control.(Control); ok && c != nil {
			width, _ := c.Size()
			if width > 0 {
				return width
			}
		}
	}
	var err error
	fd := int(stdoutTerm.Fd())
	termWidth, _, err = GetSize(fd)

	if err != nil || termWidth == 0 {
		termWidth = defaultTermWidth
	}

	return
}

// GetLength returns the length of the terminal
// (Y length), or 80 if it cannot be established.
func GetLength() int {
	if control, ok := controls.Load(goid()); ok {
		if c, ok := control.(Control); ok && c != nil {
			_, length := c.Size()
			if length > 0 {
				return length
			}
		}
	}
	termFd := int(stdoutTerm.Fd())

	_, length, err := GetSize(termFd)
	if err != nil || length == 0 {
		return defaultTermWidth
	}

	return length
}

// OnResize registers a resize callback on the active terminal control.
func OnResize(fn func(cols, rows int)) func() {
	if control, ok := controls.Load(goid()); ok {
		if c, ok := control.(Control); ok && c != nil {
			return c.OnResize(fn)
		}
	}
	return func() {}
}

func printf(format string, a ...interface{}) {
	s := fmt.Sprintf(format, a...)
	Print(s)
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
