package term

import (
	"fmt"
	"io"
	"strings"
	"sync"
	"time"

	bspinner "github.com/charmbracelet/bubbles/spinner"
)

const SpinnerSentinel = "\x00"

var defaultFrames = bspinner.Dot

type LiveView struct {
	w      io.Writer
	accent string

	mu       sync.Mutex
	lines    []string
	running  bool
	hidden   bool
	frame    string
	rendered int
	stop     chan struct{}
	done     chan struct{}
}

func NewLiveView(w io.Writer, accent string) *LiveView {
	return &LiveView{w: w, accent: accent}
}

func (v *LiveView) Update(lines []string) {
	if v == nil {
		return
	}
	v.mu.Lock()
	defer v.mu.Unlock()
	v.lines = make([]string, len(lines))
	copy(v.lines, lines)
	if v.running && !v.hidden {
		v.renderLocked(v.currentFrame())
	}
}

func (v *LiveView) Start() {
	if v == nil || v.w == nil {
		return
	}
	v.mu.Lock()
	defer v.mu.Unlock()
	if v.running {
		return
	}
	v.stop = make(chan struct{})
	v.done = make(chan struct{})
	v.running = true
	v.frame = defaultFrames.Frames[0]
	v.renderLocked(v.frame)
	go v.tick()
}

func (v *LiveView) tick() {
	defer close(v.done)
	frames := defaultFrames.Frames
	t := time.NewTicker(defaultFrames.FPS)
	defer t.Stop()
	idx := 0
	for {
		v.render(frames[idx])
		idx = (idx + 1) % len(frames)
		select {
		case <-v.stop:
			return
		case <-t.C:
		}
	}
}

func (v *LiveView) render(frame string) {
	v.mu.Lock()
	v.renderLocked(frame)
	v.mu.Unlock()
}

func (v *LiveView) renderLocked(frame string) {
	v.frame = frame
	if v.hidden {
		return
	}
	lines := make([]string, len(v.lines))
	copy(lines, v.lines)
	prev := v.rendered

	if len(lines) == 0 {
		if prev > 0 {
			WriteSynced(v.w, func() {
				EraseLines(v.w, prev)
			})
			v.rendered = 0
		}
		return
	}

	marker := v.accent + frame + "\x1b[0m"
	WriteSynced(v.w, func() {
		EraseLines(v.w, prev)
		for i, line := range lines {
			replaced := strings.Replace(line, SpinnerSentinel, marker, 1)
			if i < len(lines)-1 {
				fmt.Fprintf(v.w, "%s\n", replaced)
			} else {
				fmt.Fprint(v.w, replaced)
			}
		}
	})

	v.rendered = len(lines)
}

func (v *LiveView) WithHidden(fn func()) {
	if v == nil {
		if fn != nil {
			fn()
		}
		return
	}
	v.mu.Lock()
	if !v.running {
		v.mu.Unlock()
		if fn != nil {
			fn()
		}
		return
	}
	if v.rendered > 0 {
		WriteSynced(v.w, func() {
			EraseLines(v.w, v.rendered)
		})
		v.rendered = 0
	}
	v.hidden = true
	v.mu.Unlock()

	if fn != nil {
		fn()
	}

	v.mu.Lock()
	v.hidden = false
	if v.running {
		v.renderLocked(v.currentFrame())
	}
	v.mu.Unlock()
}

func (v *LiveView) Stop() {
	if v == nil {
		return
	}
	v.mu.Lock()
	if !v.running {
		v.mu.Unlock()
		return
	}
	close(v.stop)
	v.running = false
	v.hidden = false
	n := v.rendered
	v.rendered = 0
	done := v.done
	v.mu.Unlock()
	<-done
	if n > 0 {
		WriteSynced(v.w, func() {
			EraseLines(v.w, n)
		})
	}
}

func (v *LiveView) currentFrame() string {
	if v.frame != "" {
		return v.frame
	}
	return defaultFrames.Frames[0]
}
