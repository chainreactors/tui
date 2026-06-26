package mux

import "time"

// Option configures the Mux model.
type Option func(m *Mux)

// WithPrefixKey sets the prefix key byte (default: 0x02 = Ctrl+B).
func WithPrefixKey(key byte) Option {
	return func(m *Mux) { m.prefixKey = key }
}

// WithRefreshInterval sets how often the UI re-renders even when idle.
// A lower value gives smoother updates but uses more CPU. Default: 50ms.
func WithRefreshInterval(d time.Duration) Option {
	return func(m *Mux) { m.refreshInterval = d }
}

// WithKeyMap replaces the default prefix-key action map.
func WithKeyMap(km map[byte]MuxAction) Option {
	return func(m *Mux) { m.keyMap = km }
}

// WithSidebarWidth sets the console manager sidebar width in columns.
// Set to 0 to disable the sidebar. Default: 20.
func WithSidebarWidth(w int) Option {
	return func(m *Mux) { m.sidebarWidth = w }
}

// PaneFactory is a function the Mux calls to create a new TermPane.
// The caller provides this so the Mux doesn't need to know the specific
// executable or arguments. id and dimensions are provided by the Mux.
type PaneFactory func(id int, width, height int) (*TermPane, error)

// SessionPaneFactory creates a pane pre-bound to a specific session.
// sessionID is passed as --use <sid> to the child process.
type SessionPaneFactory func(id int, sessionID string, width, height int) (*TermPane, error)

// WithPaneFactory sets the function used to create new panes.
func WithPaneFactory(f PaneFactory) Option {
	return func(m *Mux) { m.paneFactory = f }
}

// WithSessionPaneFactory sets the function used to create session-bound panes.
func WithSessionPaneFactory(f SessionPaneFactory) Option {
	return func(m *Mux) { m.sessionPaneFactory = f }
}
