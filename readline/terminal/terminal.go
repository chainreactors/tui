package terminal

import (
	"context"
	"errors"
	"io"
	"os"
	"sync"

	internalterm "github.com/chainreactors/tui/readline/internal/term"
)

// Control exposes optional terminal capabilities for a byte stream.
type Control interface {
	IsTerminal() bool
	Size() (cols, rows int)
	MakeRaw() (restore func(), err error)
	OnResize(func(cols, rows int)) func()
	Close() error
}

// Terminal is the transport-neutral IO boundary used by readline.
type Terminal struct {
	In      io.Reader
	Out     io.Writer
	Err     io.Writer
	Control Control
}

// Local returns a Terminal backed by process stdin/stdout/stderr.
func Local() *Terminal {
	return Stream(os.Stdin, os.Stdout, os.Stderr, localControl{})
}

// Stream returns a Terminal backed by arbitrary streams and optional control.
func Stream(in io.Reader, out, err io.Writer, control Control) *Terminal {
	if in == nil {
		in = eofReader{}
	}
	if out == nil {
		out = io.Discard
	}
	if err == nil {
		err = out
	}
	if control == nil {
		control = NewControl(true, 80, 24)
	}
	return &Terminal{In: in, Out: out, Err: err, Control: control}
}

// EventType is a transport-neutral terminal event.
type EventType string

const (
	EventData     EventType = "data"
	EventResize   EventType = "resize"
	EventRawEnter EventType = "raw.enter"
	EventRawExit  EventType = "raw.exit"
	EventClose    EventType = "close"
	EventError    EventType = "error"
)

// Event is the minimal frame shape for adapting arbitrary carriers.
type Event struct {
	Type    EventType
	Data    []byte
	Cols    int
	Rows    int
	Message string
}

// Carrier is implemented by WebSocket, TCP, stdio relay, or any other channel.
type Carrier interface {
	Send(context.Context, Event) error
	Recv(context.Context) (Event, error)
}

// Remote owns a Terminal backed by a Carrier.
type Remote struct {
	Terminal *Terminal

	ctx    context.Context
	cancel context.CancelFunc
	input  *io.PipeWriter
}

// RemoteOptions configures a carrier-backed terminal.
type RemoteOptions struct {
	IsTerminal bool
	Cols       int
	Rows       int
}

// NewRemote adapts any carrier into a Terminal.
func NewRemote(ctx context.Context, carrier Carrier, opts RemoteOptions) (*Remote, error) {
	if carrier == nil {
		return nil, errors.New("terminal carrier is nil")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	ctx, cancel := context.WithCancel(ctx)
	pr, pw := io.Pipe()
	control := &carrierControl{
		StreamControl: NewControl(opts.IsTerminal, opts.Cols, opts.Rows),
		ctx:           ctx,
		carrier:       carrier,
	}
	remote := &Remote{
		ctx:    ctx,
		cancel: cancel,
		input:  pw,
	}
	remote.Terminal = Stream(pr, carrierWriter{ctx: ctx, carrier: carrier}, nil, control)
	go remote.recvLoop(carrier)
	return remote, nil
}

// Close closes the remote terminal.
func (r *Remote) Close() error {
	if r == nil {
		return nil
	}
	r.cancel()
	_ = r.input.Close()
	if r.Terminal != nil && r.Terminal.Control != nil {
		return r.Terminal.Control.Close()
	}
	return nil
}

func (r *Remote) recvLoop(carrier Carrier) {
	defer r.cancel()
	defer r.input.Close()
	for {
		event, err := carrier.Recv(r.ctx)
		if err != nil {
			return
		}
		switch event.Type {
		case EventData:
			if len(event.Data) > 0 {
				_, _ = r.input.Write(event.Data)
			}
		case EventResize:
			if c, ok := r.Terminal.Control.(*carrierControl); ok {
				c.SetSize(event.Cols, event.Rows)
			}
		case EventClose:
			return
		}
	}
}

type carrierWriter struct {
	ctx     context.Context
	carrier Carrier
}

func (w carrierWriter) Write(p []byte) (int, error) {
	data := append([]byte(nil), p...)
	if err := w.carrier.Send(w.ctx, Event{Type: EventData, Data: data}); err != nil {
		return 0, err
	}
	return len(p), nil
}

type carrierControl struct {
	*StreamControl
	ctx     context.Context
	carrier Carrier
}

func (c *carrierControl) MakeRaw() (func(), error) {
	if err := c.carrier.Send(c.ctx, Event{Type: EventRawEnter}); err != nil {
		return nil, err
	}
	return func() {
		_ = c.carrier.Send(c.ctx, Event{Type: EventRawExit})
	}, nil
}

func (c *carrierControl) Close() error {
	return c.carrier.Send(c.ctx, Event{Type: EventClose})
}

// StreamControl is a reusable in-memory terminal control implementation.
type StreamControl struct {
	mu          sync.Mutex
	terminal    bool
	cols        int
	rows        int
	nextID      int
	callbacks   map[int]func(int, int)
	closeFunc   func() error
	makeRawFunc func() (func(), error)
}

// NewControl returns a mutable terminal control object.
func NewControl(isTerminal bool, cols, rows int) *StreamControl {
	if cols <= 0 {
		cols = 80
	}
	if rows <= 0 {
		rows = 24
	}
	return &StreamControl{
		terminal:  isTerminal,
		cols:      cols,
		rows:      rows,
		callbacks: make(map[int]func(int, int)),
	}
}

func (c *StreamControl) IsTerminal() bool {
	if c == nil {
		return false
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.terminal
}

func (c *StreamControl) Size() (int, int) {
	if c == nil {
		return 80, 24
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.cols, c.rows
}

func (c *StreamControl) SetSize(cols, rows int) {
	if c == nil || cols <= 0 || rows <= 0 {
		return
	}
	c.mu.Lock()
	c.cols = cols
	c.rows = rows
	callbacks := make([]func(int, int), 0, len(c.callbacks))
	for _, cb := range c.callbacks {
		callbacks = append(callbacks, cb)
	}
	c.mu.Unlock()
	for _, cb := range callbacks {
		cb(cols, rows)
	}
}

func (c *StreamControl) MakeRaw() (func(), error) {
	if c != nil && c.makeRawFunc != nil {
		return c.makeRawFunc()
	}
	return func() {}, nil
}

func (c *StreamControl) OnResize(fn func(int, int)) func() {
	if c == nil || fn == nil {
		return func() {}
	}
	c.mu.Lock()
	c.nextID++
	id := c.nextID
	c.callbacks[id] = fn
	c.mu.Unlock()
	return func() {
		c.mu.Lock()
		delete(c.callbacks, id)
		c.mu.Unlock()
	}
}

func (c *StreamControl) Close() error {
	if c != nil && c.closeFunc != nil {
		return c.closeFunc()
	}
	return nil
}

type localControl struct{}

func (localControl) IsTerminal() bool {
	return internalterm.IsTerminal(int(os.Stdin.Fd()))
}

func (localControl) Size() (int, int) {
	cols, rows, err := internalterm.GetSize(int(os.Stdout.Fd()))
	if err != nil || cols <= 0 || rows <= 0 {
		return 80, 24
	}
	return cols, rows
}

func (localControl) MakeRaw() (func(), error) {
	if !internalterm.IsTerminal(int(os.Stdin.Fd())) {
		return func() {}, nil
	}
	state, err := internalterm.MakeRaw(int(os.Stdin.Fd()))
	if err != nil {
		return nil, err
	}
	return func() { _ = internalterm.Restore(int(os.Stdin.Fd()), state) }, nil
}

func (localControl) OnResize(func(int, int)) func() {
	return func() {}
}

func (localControl) Close() error {
	return nil
}

type eofReader struct{}

func (eofReader) Read([]byte) (int, error) {
	return 0, io.EOF
}
