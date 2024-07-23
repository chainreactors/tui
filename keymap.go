package tui

import (
	"github.com/charmbracelet/bubbles/key"
)

// DefaultKeyMap defines a set of keybindings. To work for Help it must satisfy
// key.Map. It could also very easily be a map[string]key.Binding.
type DefaultKeyMap struct {
	Up      key.Binding
	Down    key.Binding
	Left    key.Binding
	Right   key.Binding
	Help    key.Binding
	Quit    key.Binding
	Console key.Binding
}

// ShortHelp returns keybindings to be shown in the mini Help view. It's part
// of the key.Map interface.
func (k DefaultKeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Help, k.Quit}
}

// FullHelp returns keybindings for the expanded Help view. It's part of the
// key.Map interface.
func (k DefaultKeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Up, k.Down, k.Left, k.Right}, // first column
		{k.Help, k.Quit},                // second column
	}
}

var DefaultKeys = DefaultKeyMap{
	Up: key.NewBinding(
		key.WithKeys("up", "k"),
		key.WithHelp("↑/k", "move up"),
	),
	Down: key.NewBinding(
		key.WithKeys("down", "j"),
		key.WithHelp("↓/j", "move down"),
	),
	Left: key.NewBinding(
		key.WithKeys("left", "h"),
		key.WithHelp("←/h", "move left"),
	),
	Right: key.NewBinding(
		key.WithKeys("right", "l"),
		key.WithHelp("→/l", "move right"),
	),
	Help: key.NewBinding(
		key.WithKeys("?"),
		key.WithHelp("?", "toggle Help"),
	),
	Quit: key.NewBinding(
		key.WithKeys("ctrl+q", "esc", "ctrl+c"),
		key.WithHelp("ctrl+q", "quit"),
	),
	Console: key.NewBinding(
		key.WithKeys(":"),
		key.WithHelp(":", "input command"),
	),
}

var DefaultShortKeys = DefaultKeyMap{
	Help: key.NewBinding(
		key.WithKeys("?"),
		key.WithHelp("?", "toggle Help"),
	),
	Quit: key.NewBinding(
		key.WithKeys("ctrl+q", "esc", "ctrl+c"),
		key.WithHelp("ctrl+q", "quit"),
	),
	Console: key.NewBinding(
		key.WithKeys(":"),
		key.WithHelp(":", "input command"),
	),
}
