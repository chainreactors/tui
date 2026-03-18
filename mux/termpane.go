// Package mux provides a tmux-like terminal multiplexer library built on
// Bubble Tea. Each pane runs a subprocess in its own PTY, with a VT terminal
// emulator maintaining the screen buffer.
package mux

import (
	"context"
	"io"
	"os/exec"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/charmbracelet/x/vt"
	"github.com/charmbracelet/x/xpty"
)

// MuxCmd is a command sent from a child process to the multiplexer via OSC title.
type MuxCmd struct {
	PaneID int
	Action string // e.g. "MuxOpen"
	Arg    string // e.g. session ID
}

// TermPane wraps a subprocess running in a PTY with a VT terminal emulator.
// It maintains a screen buffer that can be rendered as a string at any time.
type TermPane struct {
	id   int
	name string

	pty xpty.Pty
	vt  *vt.Emulator
	cmd *exec.Cmd

	width  int
	height int

	focused atomic.Bool
	dead    atomic.Bool

	ctx    context.Context
	cancel context.CancelFunc

	mu sync.Mutex // serialises all vt reads and writes

	// renderCache holds the last successfully rendered screen as a string.
	// Updated by readLoop after each vt.Write; read by Render without locking
	// so that View() in the Bubble Tea event loop never blocks on the mutex.
	renderCache atomic.Value // type: string

	// scrollOffset > 0 means we are viewing scrollback history.
	// 0 = live (showing current screen), N = N lines scrolled up.
	scrollOffset int

	// inputCh receives bytes to write to the PTY. A dedicated writeLoop
	// goroutine drains it so that WriteInput never blocks the caller.
	inputCh chan []byte

	// MuxCmds receives commands from the child process via OSC title sequences.
	// The mux model should drain this channel in its Update loop.
	MuxCmds chan MuxCmd
}

// NewTermPane creates a new terminal pane that runs the given command in a PTY.
// The pane maintains a VT terminal emulator screen buffer of the specified
// dimensions.
func NewTermPane(id int, name string, exe string, args []string, width, height int) (*TermPane, error) {
	p, err := xpty.NewPty(width, height)
	if err != nil {
		return nil, err
	}

	em := vt.NewEmulator(width, height)
	ctx, cancel := context.WithCancel(context.Background())

	cmd := exec.Command(exe, args...)

	if err := p.Start(cmd); err != nil {
		p.Close()
		em.Close()
		cancel()
		return nil, err
	}

	tp := &TermPane{
		id:      id,
		name:    name,
		pty:     p,
		vt:      em,
		cmd:     cmd,
		width:   width,
		height:  height,
		ctx:     ctx,
		cancel:  cancel,
		inputCh: make(chan []byte, 64),
		MuxCmds: make(chan MuxCmd, 8),
	}

	// Register VT callbacks for inter-process communication via OSC title.
	// Child processes write "\x1b]0;MuxOpen=<session-id>\x07" to request
	// the multiplexer to open a new pane.
	em.SetCallbacks(vt.Callbacks{
		Title: func(title string) {
			if action, arg, ok := parseMuxTitle(title); ok {
				select {
				case tp.MuxCmds <- MuxCmd{PaneID: id, Action: action, Arg: arg}:
				default:
				}
			}
			// Non-Mux titles (shell PS1, PROMPT_COMMAND, etc.) are ignored.
			// Pane name is only set at creation or via explicit SetName().
		},
	})

	go tp.readLoop()
	go tp.writeLoop()
	go tp.vtResponseLoop()
	go tp.waitExit()

	return tp, nil
}

// readLoop continuously reads PTY output and writes it to the VT emulator.
// After each write it atomically updates renderCache so that Render() in the
// Bubble Tea event loop can read the latest screen content without blocking.
func (tp *TermPane) readLoop() {
	buf := make([]byte, 32*1024)
	for {
		select {
		case <-tp.ctx.Done():
			return
		default:
		}

		n, err := tp.pty.Read(buf)
		if n > 0 {
			tp.mu.Lock()
			tp.vt.Write(buf[:n])
			rendered := tp.vt.Render()
			tp.mu.Unlock()
			tp.renderCache.Store(rendered)
		}
		if err != nil {
			return
		}
	}
}

// writeLoop drains inputCh and writes bytes to the PTY. Running in its own
// goroutine ensures that PTY writes never block the Bubble Tea event loop.
func (tp *TermPane) writeLoop() {
	for {
		select {
		case <-tp.ctx.Done():
			return
		case data := <-tp.inputCh:
			tp.pty.Write(data)
		}
	}
}

// vtResponseLoop forwards VT emulator responses back to the PTY.
// The VT emulator processes certain terminal queries (e.g. \x1b[6n cursor
// position request) by writing a response to its internal io.Pipe write end.
// If nobody drains that pipe, vt.Write() blocks permanently while holding
// tp.mu, deadlocking the event loop. This goroutine drains the pipe and
// forwards the bytes to the PTY so the child process receives the reply.
func (tp *TermPane) vtResponseLoop() {
	buf := make([]byte, 256)
	for {
		n, err := tp.vt.Read(buf)
		if n > 0 {
			tp.pty.Write(buf[:n])
		}
		if err != nil {
			return
		}
		select {
		case <-tp.ctx.Done():
			return
		default:
		}
	}
}

// waitExit monitors subprocess termination.
func (tp *TermPane) waitExit() {
	xpty.WaitProcess(tp.ctx, tp.cmd)
	tp.dead.Store(true)
}

// ID returns the pane's unique identifier.
func (tp *TermPane) ID() int { return tp.id }

// Name returns the pane's display name.
func (tp *TermPane) Name() string { return tp.name }

// SetName explicitly updates the pane's display name.
func (tp *TermPane) SetName(name string) { tp.name = name }

// IsDead returns true if the subprocess has exited.
func (tp *TermPane) IsDead() bool { return tp.dead.Load() }

// IsFocused returns true if this pane currently has input focus.
func (tp *TermPane) IsFocused() bool { return tp.focused.Load() }

// Focus gives this pane input focus.
func (tp *TermPane) Focus() { tp.focused.Store(true) }

// Blur removes input focus from this pane.
func (tp *TermPane) Blur() { tp.focused.Store(false) }

// WriteInput sends raw bytes to the subprocess via the PTY. It is safe to call
// from any goroutine, including the Bubble Tea event loop, and never blocks.
func (tp *TermPane) WriteInput(p []byte) (int, error) {
	if tp.dead.Load() {
		return 0, io.ErrClosedPipe
	}
	// Copy the slice before handing it off: the caller may reuse the buffer.
	buf := make([]byte, len(p))
	copy(buf, p)
	select {
	case tp.inputCh <- buf:
	default:
		// Channel full: drop to avoid blocking the event loop.
	}
	return len(p), nil
}

// ScrollUp scrolls the viewport up by n lines into the scrollback buffer.
func (tp *TermPane) ScrollUp(n int) {
	tp.mu.Lock()
	maxOffset := tp.vt.ScrollbackLen()
	tp.mu.Unlock()
	tp.scrollOffset += n
	if tp.scrollOffset > maxOffset {
		tp.scrollOffset = maxOffset
	}
}

// ScrollDown scrolls the viewport down by n lines toward the live screen.
func (tp *TermPane) ScrollDown(n int) {
	tp.scrollOffset -= n
	if tp.scrollOffset < 0 {
		tp.scrollOffset = 0
	}
}

// IsScrolled returns true when viewing scrollback (not live).
func (tp *TermPane) IsScrolled() bool {
	return tp.scrollOffset > 0
}

// Render returns the current screen content as an ANSI-encoded string.
// For the live view (scrollOffset == 0) it reads from the atomic renderCache,
// which is updated by readLoop after every vt.Write. This path never blocks.
// For scrollback it acquires the mutex briefly to read from the vt emulator.
func (tp *TermPane) Render() string {
	if tp.scrollOffset == 0 {
		// Fast non-blocking path: use the atomically cached render.
		if v := tp.renderCache.Load(); v != nil {
			return v.(string)
		}
		return ""
	}

	// Scrollback path: needs consistent vt state. vtResponseLoop drains the
	// VT's internal pipe so vt.Write() never blocks permanently; this lock
	// is held only for the brief duration of the scrollback cell reads.
	tp.mu.Lock()
	defer tp.mu.Unlock()

	// Build view from scrollback + screen content.
	h := tp.vt.Height()
	w := tp.vt.Width()
	sbLen := tp.vt.ScrollbackLen()

	if tp.scrollOffset > sbLen {
		tp.scrollOffset = sbLen
	}

	var lines []string
	sbStart := sbLen - tp.scrollOffset
	sbLines := tp.scrollOffset
	if sbLines > h {
		sbLines = h
		sbStart = sbLen - h
	}
	for y := sbStart; y < sbStart+sbLines && y < sbLen; y++ {
		line := ""
		for x := 0; x < w; x++ {
			cell := tp.vt.ScrollbackCellAt(x, y)
			if cell != nil && cell.Content != "" {
				line += cell.Content
			} else {
				line += " "
			}
		}
		lines = append(lines, line)
	}
	screenLines := h - len(lines)
	for y := 0; y < screenLines; y++ {
		line := ""
		for x := 0; x < w; x++ {
			c := tp.vt.CellAt(x, y)
			if c != nil && c.Content != "" {
				line += c.Content
			} else {
				line += " "
			}
		}
		lines = append(lines, line)
	}

	return strings.Join(lines, "\n")
}

// Resize changes the pane dimensions, propagating to both PTY and VT emulator.
func (tp *TermPane) Resize(width, height int) error {
	if width <= 0 || height <= 0 {
		return nil
	}

	tp.width = width
	tp.height = height

	tp.mu.Lock()
	tp.vt.Resize(width, height)
	tp.mu.Unlock()

	err := tp.pty.Resize(width, height)
	return err
}

// Width returns the current pane width.
func (tp *TermPane) Width() int { return tp.width }

// Height returns the current pane height.
func (tp *TermPane) Height() int { return tp.height }

// parseMuxTitle checks if a title string is a mux command (e.g. "MuxOpen=<sid>").
// Returns (action, arg, true) if it matches, or ("", "", false) otherwise.
func parseMuxTitle(title string) (action, arg string, ok bool) {
	const prefix = "Mux"
	if !strings.HasPrefix(title, prefix) {
		return "", "", false
	}
	eq := strings.IndexByte(title, '=')
	if eq < 0 {
		return title, "", true
	}
	return title[:eq], title[eq+1:], true
}

// Close shuts down the pane: closes PTY, VT emulator, and kills the subprocess.
func (tp *TermPane) Close() error {
	tp.cancel()
	tp.vt.Close()

	if tp.cmd.Process != nil && !tp.dead.Load() {
		tp.cmd.Process.Kill()
	}

	return tp.pty.Close()
}
