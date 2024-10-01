package tui

import (
	"fmt"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/muesli/termenv"
	"regexp"
	"strings"
)

// TreeNode represents a Tree node (file or folder)
type TreeNode struct {
	Name     string
	Children []*TreeNode
	Info     []string // Additional info, like file size, permissions, etc.
}

// KeyActionFunc is a function type for handling custom key actions
type KeyActionFunc func(model *TreeModel) (tea.Model, tea.Cmd)

// DisplayFunc is a function type for custom display logic
type DisplayFunc func(node *TreeNode) string

// TreeModel represents the Bubble Tea model
type TreeModel struct {
	Cursor           int                      // Current Selected node index
	Selected         []string                 // Path to the Selected node
	Tree             *TreeNode                // Current Tree node
	Root             *TreeNode                // Store the Root node for navigation
	headDisplayFn    func() string            // User-defined function for custom display
	contentDisplayFn DisplayFunc              // User-defined function for custom display
	keyBindings      map[string]KeyActionFunc // Key bindings and their actions

}

// Init is the Bubble Tea init function (empty in this case)
func (m TreeModel) Init() tea.Cmd {
	return nil
}

// Update handles messages and updates the model state
func (m TreeModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		key := msg.String()
		// Check if the user has defined an action for this key
		if action, exists := m.keyBindings[key]; exists {
			return action(&m)
		}
		switch key {
		case "up":
			if m.Cursor > 0 {
				m.Cursor--
			}
		case "down":
			if m.Cursor < len(m.Tree.Children)-1 {
				m.Cursor++
			}
		case "ctrl+c":
			return m, tea.Quit
		}
	}
	return m, nil
}

// View renders the current view of the file Tree
func (m TreeModel) View() string {
	var b strings.Builder

	// Render current path
	b.WriteString(m.headDisplayFn())

	// Render the Tree structure
	for i, child := range m.Tree.Children {
		cursor := " " // No Cursor
		displayStr := m.contentDisplayFn(child)
		if m.Cursor == i {
			cursor = ">"
			plainDisplayStr := stripAnsiCodes(displayStr)
			displayStr = termenv.String(plainDisplayStr).Foreground(Pink).String() // Current Selected node
		}

		// Use the custom display function provided by the user

		b.WriteString(fmt.Sprintf("%s %s\n", cursor, displayStr))
	}

	return b.String()
}

func (m TreeModel) SetHeaderView(headerDisplay func() string) TreeModel {
	m.headDisplayFn = headerDisplay
	return m
}
func (m TreeModel) SetChildren(children []*TreeNode) TreeModel {
	m.Tree.Children = children
	return m
}

func (m TreeModel) SetKeyBinding(key string, action KeyActionFunc) TreeModel {
	if m.keyBindings == nil {
		m.keyBindings = make(map[string]KeyActionFunc)
	}
	m.keyBindings[key] = action
	return m
}

func NewTreeModel(root TreeNode, displayFunc DisplayFunc) TreeModel {
	return TreeModel{
		Tree:             &root,
		Root:             &root,
		contentDisplayFn: displayFunc,
	}
}

var ansi = regexp.MustCompile(`\x1b\[[0-9;]*m`)

// Helper function to strip ANSI codes
func stripAnsiCodes(str string) string {
	return ansi.ReplaceAllString(str, "")
}
