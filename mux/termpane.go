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

	mu sync.RWMutex // protects vt access during concurrent read/render

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
	go tp.waitExit()

	return tp, nil
}

// readLoop continuously reads PTY output and writes it to the VT emulator.
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
			tp.mu.Unlock()
		}
		if err != nil {
			return
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

// WriteInput sends raw bytes to the subprocess via the PTY.
func (tp *TermPane) WriteInput(p []byte) (int, error) {
	if tp.dead.Load() {
		return 0, io.ErrClosedPipe
	}
	return tp.pty.Write(p)
}

// Render returns the current screen content as an ANSI-encoded string.
// This can be directly used in Bubble Tea's View().
func (tp *TermPane) Render() string {
	tp.mu.RLock()
	defer tp.mu.RUnlock()
	return tp.vt.Render()
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

	return tp.pty.Resize(width, height)
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
