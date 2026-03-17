package mux

// MuxAction represents a multiplexer command triggered after the prefix key.
type MuxAction int

const (
	ActionNone          MuxAction = iota
	ActionNextTab                 // switch to next tab
	ActionPrevTab                 // switch to previous tab
	ActionNewPane                 // create a new pane/tab
	ActionClosePane               // close the focused pane
	ActionSplitH                  // split focused pane horizontally
	ActionSplitV                  // split focused pane vertically
	ActionFocusNext               // move focus to next pane in layout
	ActionFocusPrev               // move focus to previous pane in layout
	ActionSessionPicker           // open session picker overlay
	ActionPaneList                // open pane navigator overlay
	ActionScrollback              // enter scrollback mode
	ActionQuit                    // quit the multiplexer
	ActionHelp                    // show help
)

// DefaultKeyMap maps bytes (received after the prefix key) to mux actions.
// Modeled after tmux defaults.
var DefaultKeyMap = map[byte]MuxAction{
	'n': ActionNextTab,
	'p': ActionPrevTab,
	'c': ActionNewPane,
	'x': ActionClosePane,
	'"': ActionSplitV,
	'%': ActionSplitH,
	'o': ActionFocusNext,
	's': ActionSessionPicker,
	'w': ActionPaneList,
	'[': ActionScrollback,
	'q': ActionQuit,
	'?': ActionHelp,
}
