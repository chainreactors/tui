//go:build unix
// +build unix

package display

import (
	"github.com/chainreactors/tui/readline/internal/core"
	"github.com/chainreactors/tui/readline/internal/term"
)

// WatchResize redisplays the interface on terminal resize events.
func WatchResize(eng *Engine) chan<- bool {
	resizeChannel := core.GetTerminalResize(eng.keys)
	done := make(chan bool, 1)
	output := term.Output()
	control := term.CurrentControl()
	unregister := term.OnResize(func(_, _ int) {
		if eng.keys != nil && !eng.keys.IsReading() && !eng.keys.IsWaiting() {
			restore := term.Activate(output, control)
			eng.completer.GenerateCached()
			eng.Refresh()
			restore()
		}
	})

	go func() {
		defer unregister()
		for {
			select {
			case <-resizeChannel:
				if eng.keys != nil && !eng.keys.IsReading() && !eng.keys.IsWaiting() {
					restore := term.Activate(output, control)
					eng.completer.GenerateCached()
					eng.Refresh()
					restore()
				}
			case <-done:
				return
			}
		}
	}()

	return done
}
