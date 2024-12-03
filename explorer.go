package tui

import (
	"errors"
	"fmt"
	tea "github.com/charmbracelet/bubbletea"
	"regexp"
	"strings"
)

const (
	ChildrenTree = 0 + iota
	InfoTree
)

var (
	Children = "|-"
	Last     = "`-"
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
	Cursor            int                           // Current Selected node index
	Selected          []string                      // Path to the Selected node
	Tree              *TreeNode                     // Current Tree node
	Root              *TreeNode                     // Store the Root node for navigation
	headDisplayFn     func(model *TreeModel) string // User-defined function for custom display
	childrenDisplayFn DisplayFunc                   // User-defined function for custom display
	infoDisplayFn     DisplayFunc                   // User-defined function for custom display
	keyBindings       map[string]KeyActionFunc      // Key bindings and their actions
	Type              int                           // Type of the Tree (ChildrenTree or InfoTree)
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
	b.WriteString(m.headDisplayFn(&m))

	switch m.Type {
	case ChildrenTree:
		for i, child := range m.Tree.Children {
			cursor := " " // No Cursor
			displayStr := m.childrenDisplayFn(child)
			if m.Cursor == i {
				cursor = ">"
				plainDisplayStr := stripAnsiCodes(displayStr)
				displayStr = PinkFg.Render(plainDisplayStr) // Current Selected node
			}

			b.WriteString(fmt.Sprintf("%s %s\n", cursor, displayStr))
		}
	case InfoTree:
		var displayStr string
		for i, child := range m.Tree.Children {
			if contains(m.Selected, child.Name) {
				displayStr = m.infoDisplayFn(child)
			} else {
				displayStr = child.Name
			}
			if m.Cursor == i {
				plainDisplayStr := stripAnsiCodes(displayStr)
				displayStr = PinkFg.Render(plainDisplayStr) // Current Selected node
			}
			b.WriteString(fmt.Sprintf("%s\n", displayStr))
		}
	}
	// Render the Tree structure

	return b.String()
}

func (m TreeModel) SetHeaderView(headerDisplay func(model *TreeModel) string) TreeModel {
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

func (m TreeModel) Run() {
	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Println("Error running program:", err)
	}

}
func NewTreeModel(root TreeNode, displayFunc DisplayFunc, treeType int) (TreeModel, error) {
	switch treeType {
	case ChildrenTree:
		return TreeModel{
			Tree:              &root,
			Root:              &root,
			childrenDisplayFn: displayFunc,
			Type:              treeType,
			Selected:          []string{},
		}, nil
	case InfoTree:
		return TreeModel{
			Tree:          &root,
			Root:          &root,
			infoDisplayFn: displayFunc,
			Type:          treeType,
			Selected:      []string{},
		}, nil
	default:
		return TreeModel{}, errors.New("invalid tree type, use ChildrenTree or InfoTree")
	}
}

var ansi = regexp.MustCompile(`\x1b\[[0-9;]*m`)

// Helper function to strip ANSI codes
func stripAnsiCodes(str string) string {
	return ansi.ReplaceAllString(str, "")
}

// contains checks if a string is in a slice
func contains(slice []string, item string) bool {
	for _, v := range slice {
		if v == item {
			return true
		}
	}
	return false
}
