package mux

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// PickerItem represents one selectable item in the picker overlay.
type PickerItem struct {
	ID    string
	Label string
	Desc  string
	Icon  string // e.g. "●" or "○"
	Color string // lipgloss color for the icon
}

// PickerState holds the state for an interactive list picker overlay.
type PickerState struct {
	Title    string
	Items    []PickerItem
	Cursor   int
	Filter   string
	Hint     string // e.g. "Enter: select  Esc: cancel"
	OnSelect func(item PickerItem) // called when user presses Enter
}

// NewPicker creates a picker with the given title and items.
func NewPicker(title string, items []PickerItem, onSelect func(PickerItem)) *PickerState {
	return &PickerState{
		Title:    title,
		Items:    items,
		Hint:     "Enter: select  Esc: cancel",
		OnSelect: onSelect,
	}
}

// filteredItems returns items matching the current filter.
func (p *PickerState) filteredItems() []PickerItem {
	if p.Filter == "" {
		return p.Items
	}
	var out []PickerItem
	lower := strings.ToLower(p.Filter)
	for _, item := range p.Items {
		if strings.Contains(strings.ToLower(item.Label), lower) ||
			strings.Contains(strings.ToLower(item.Desc), lower) ||
			strings.Contains(strings.ToLower(item.ID), lower) {
			out = append(out, item)
		}
	}
	return out
}

// HandleKey processes a keypress in the picker. Returns (consumed, shouldClose).
func (p *PickerState) HandleKey(keyStr string) (consumed bool, close bool) {
	filtered := p.filteredItems()

	switch keyStr {
	case "up", "k":
		if p.Cursor > 0 {
			p.Cursor--
		}
		return true, false
	case "down", "j":
		if p.Cursor < len(filtered)-1 {
			p.Cursor++
		}
		return true, false
	case "enter":
		if p.Cursor < len(filtered) && p.OnSelect != nil {
			p.OnSelect(filtered[p.Cursor])
		}
		return true, true
	case "esc", "ctrl+c":
		return true, true
	case "backspace":
		if len(p.Filter) > 0 {
			p.Filter = p.Filter[:len(p.Filter)-1]
			p.Cursor = 0
		}
		return true, false
	default:
		// Single printable character → add to filter.
		if len(keyStr) == 1 && keyStr[0] >= 32 && keyStr[0] < 127 {
			p.Filter += keyStr
			p.Cursor = 0
			return true, false
		}
	}
	return false, false
}

// Render produces the picker content (without the overlay frame — that's added by renderOverlay).
func (p *PickerState) Render(maxWidth int) string {
	dim := lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	highlight := lipgloss.NewStyle().
		Foreground(lipgloss.Color("0")).
		Background(lipgloss.Color("6"))

	var lines []string

	// Filter input
	filterLine := fmt.Sprintf(" Filter: %s_", p.Filter)
	lines = append(lines, dim.Render(filterLine))
	lines = append(lines, "")

	// Items
	filtered := p.filteredItems()
	for i, item := range filtered {
		icon := lipgloss.NewStyle().Foreground(lipgloss.Color(item.Color)).Render(item.Icon)
		entry := fmt.Sprintf(" %s %-12s %s", icon, item.Label, dim.Render(item.Desc))

		if i == p.Cursor {
			entry = highlight.Render(fmt.Sprintf(" ► %-12s %s", item.Label, item.Desc))
		}

		// Truncate to width
		if lipgloss.Width(entry) > maxWidth {
			entry = entry[:maxWidth]
		}
		lines = append(lines, entry)
	}

	if len(filtered) == 0 {
		lines = append(lines, dim.Render(" (no matches)"))
	}

	lines = append(lines, "")
	lines = append(lines, dim.Render(" "+p.Hint))

	return strings.Join(lines, "\n")
}
