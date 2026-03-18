package mux

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Direction specifies a split orientation.
type Direction int

const (
	Horizontal Direction = iota // side by side (left | right)
	Vertical                    // stacked (top / bottom)
)

// LayoutNode is a binary tree node representing either a single pane (leaf) or
// a split container (branch) holding two children.
type LayoutNode struct {
	// Leaf fields — set when this node holds a pane.
	Pane *TermPane

	// Branch fields — set when this node is a split container.
	Direction Direction
	Children  [2]*LayoutNode
	Ratio     float64 // 0.0–1.0, size allocated to Children[0]
}

// NewLeaf creates a leaf node containing a single pane.
func NewLeaf(pane *TermPane) *LayoutNode {
	return &LayoutNode{Pane: pane, Ratio: 0.5}
}

// IsLeaf returns true if this node contains a pane rather than children.
func (n *LayoutNode) IsLeaf() bool {
	return n.Pane != nil
}

// Split replaces the pane identified by id with a split container holding the
// original pane and a new pane. Returns the new LayoutNode created for newPane,
// or nil if id was not found.
func (n *LayoutNode) Split(id int, dir Direction, newPane *TermPane) *LayoutNode {
	target := n.findNode(id)
	if target == nil || !target.IsLeaf() {
		return nil
	}

	// Move the existing pane into child[0], put the new pane in child[1].
	existing := target.Pane
	target.Pane = nil
	target.Direction = dir
	target.Ratio = 0.5
	target.Children[0] = NewLeaf(existing)
	target.Children[1] = NewLeaf(newPane)

	return target.Children[1]
}

// Remove removes the pane identified by id from the tree. The sibling of the
// removed node is promoted to replace the parent split. Returns true if
// removal succeeded.
func (n *LayoutNode) Remove(id int) bool {
	return n.removeChild(id)
}

// removeChild recursively searches for a child pane with the given id and
// removes it, promoting the sibling.
func (n *LayoutNode) removeChild(id int) bool {
	if n.IsLeaf() {
		return false
	}

	for i, child := range n.Children {
		if child == nil {
			continue
		}
		if child.IsLeaf() && child.Pane != nil && child.Pane.ID() == id {
			// Promote the sibling to replace this split node.
			sibling := n.Children[1-i]
			*n = *sibling
			return true
		}
		if child.removeChild(id) {
			return true
		}
	}
	return false
}

// FindPane returns the TermPane with the given id, or nil.
func (n *LayoutNode) FindPane(id int) *TermPane {
	if n.IsLeaf() {
		if n.Pane != nil && n.Pane.ID() == id {
			return n.Pane
		}
		return nil
	}
	for _, child := range n.Children {
		if child == nil {
			continue
		}
		if p := child.FindPane(id); p != nil {
			return p
		}
	}
	return nil
}

// Panes returns all panes in depth-first order.
func (n *LayoutNode) Panes() []*TermPane {
	if n.IsLeaf() {
		if n.Pane != nil {
			return []*TermPane{n.Pane}
		}
		return nil
	}
	var out []*TermPane
	for _, child := range n.Children {
		if child != nil {
			out = append(out, child.Panes()...)
		}
	}
	return out
}

// Resize propagates dimensions to all panes in the tree.
func (n *LayoutNode) Resize(width, height int) {
	if n.IsLeaf() {
		if n.Pane != nil {
			n.Pane.Resize(width, height)
		}
		return
	}

	switch n.Direction {
	case Horizontal:
		leftW := int(float64(width) * n.Ratio)
		rightW := width - leftW - 1 // -1 for separator
		if leftW < 1 {
			leftW = 1
		}
		if rightW < 1 {
			rightW = 1
		}
		if n.Children[0] != nil {
			n.Children[0].Resize(leftW, height)
		}
		if n.Children[1] != nil {
			n.Children[1].Resize(rightW, height)
		}

	case Vertical:
		topH := int(float64(height) * n.Ratio)
		botH := height - topH - 1 // -1 for separator
		if topH < 1 {
			topH = 1
		}
		if botH < 1 {
			botH = 1
		}
		if n.Children[0] != nil {
			n.Children[0].Resize(width, topH)
		}
		if n.Children[1] != nil {
			n.Children[1].Resize(width, botH)
		}
	}
}

// Render produces a string representation of the layout tree. Leaf nodes
// render their pane's terminal content. Branch nodes join children with a
// separator.
func (n *LayoutNode) Render(width, height int, focusedID int) string {
	if n.IsLeaf() {
		if n.Pane == nil {
			return ""
		}
		content := n.Pane.Render()
		style := lipgloss.NewStyle().
			Width(width).
			Height(height).
			MaxWidth(width).
			MaxHeight(height)
		return style.Render(content)
	}

	switch n.Direction {
	case Horizontal:
		leftW := int(float64(width) * n.Ratio)
		rightW := width - leftW - 1
		if leftW < 1 {
			leftW = 1
		}
		if rightW < 1 {
			rightW = 1
		}

		left := ""
		right := ""
		if n.Children[0] != nil {
			left = n.Children[0].Render(leftW, height, focusedID)
		}
		if n.Children[1] != nil {
			right = n.Children[1].Render(rightW, height, focusedID)
		}

		sep := renderVerticalSep(height)
		return lipgloss.JoinHorizontal(lipgloss.Top, left, sep, right)

	case Vertical:
		topH := int(float64(height) * n.Ratio)
		botH := height - topH - 1
		if topH < 1 {
			topH = 1
		}
		if botH < 1 {
			botH = 1
		}

		top := ""
		bot := ""
		if n.Children[0] != nil {
			top = n.Children[0].Render(width, topH, focusedID)
		}
		if n.Children[1] != nil {
			bot = n.Children[1].Render(width, botH, focusedID)
		}

		sep := renderHorizontalSep(width)
		return lipgloss.JoinVertical(lipgloss.Left, top, sep, bot)
	}

	return ""
}

// PaneOrigin returns the (x, y) top-left offset of the pane with the given id
// within a layout area of the given dimensions. The coordinates are 0-indexed
// and use the same split arithmetic as Render/Resize. Returns found=false when
// the id is not present in this subtree.
func (n *LayoutNode) PaneOrigin(id, width, height int) (x, y int, found bool) {
	if n.IsLeaf() {
		if n.Pane != nil && n.Pane.ID() == id {
			return 0, 0, true
		}
		return 0, 0, false
	}

	switch n.Direction {
	case Horizontal:
		leftW := int(float64(width) * n.Ratio)
		rightW := width - leftW - 1 // -1 for separator
		if leftW < 1 {
			leftW = 1
		}
		if rightW < 1 {
			rightW = 1
		}
		if n.Children[0] != nil {
			if ox, oy, ok := n.Children[0].PaneOrigin(id, leftW, height); ok {
				return ox, oy, true
			}
		}
		if n.Children[1] != nil {
			if ox, oy, ok := n.Children[1].PaneOrigin(id, rightW, height); ok {
				return leftW + 1 + ox, oy, true // +1 for separator column
			}
		}

	case Vertical:
		topH := int(float64(height) * n.Ratio)
		botH := height - topH - 1 // -1 for separator
		if topH < 1 {
			topH = 1
		}
		if botH < 1 {
			botH = 1
		}
		if n.Children[0] != nil {
			if ox, oy, ok := n.Children[0].PaneOrigin(id, width, topH); ok {
				return ox, oy, true
			}
		}
		if n.Children[1] != nil {
			if ox, oy, ok := n.Children[1].PaneOrigin(id, width, botH); ok {
				return ox, topH + 1 + oy, true // +1 for separator row
			}
		}
	}

	return 0, 0, false
}

// findNode finds the leaf node containing the pane with the given id.
func (n *LayoutNode) findNode(id int) *LayoutNode {	if n.IsLeaf() {
		if n.Pane != nil && n.Pane.ID() == id {
			return n
		}
		return nil
	}
	for _, child := range n.Children {
		if child == nil {
			continue
		}
		if found := child.findNode(id); found != nil {
			return found
		}
	}
	return nil
}

// renderVerticalSep returns a vertical separator bar of the given height.
func renderVerticalSep(height int) string {
	return renderVerticalSepColor(height, "8")
}

func renderVerticalSepColor(height int, color string) string {
	sep := lipgloss.NewStyle().
		Foreground(lipgloss.Color(color)).
		Render("│")
	lines := make([]string, height)
	for i := range lines {
		lines[i] = sep
	}
	return strings.Join(lines, "\n")
}

// renderHorizontalSep returns a horizontal separator line of the given width.
func renderHorizontalSep(width int) string {
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color("8")).
		Render(strings.Repeat("─", width))
}
