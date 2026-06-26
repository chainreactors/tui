package mux

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var (
	tabStyle = lipgloss.NewStyle().
			Padding(0, 1)

	activeTabStyle = lipgloss.NewStyle().
			Padding(0, 1).
			Bold(true).
			Foreground(lipgloss.Color("0")).
			Background(lipgloss.Color("6"))

	deadTabStyle = lipgloss.NewStyle().
			Padding(0, 1).
			Foreground(lipgloss.Color("8")).
			Strikethrough(true)

	prefixIndicator = lipgloss.NewStyle().
			Padding(0, 1).
			Bold(true).
			Foreground(lipgloss.Color("3")).
			Render("PREFIX")

	helpHint = lipgloss.NewStyle().
			Foreground(lipgloss.Color("8"))
)

// renderStatusBar produces the bottom status bar showing tabs and mode info.
func renderStatusBar(tabs []*LayoutNode, activeTab int, focusedID int, prefixMode bool, mouseEnabled bool, width int) string {
	var parts []string

	for i, tab := range tabs {
		panes := tab.Panes()
		name := fmt.Sprintf("%d", i)
		if len(panes) == 1 {
			name = fmt.Sprintf("%d:%s", i, panes[0].Name())
		} else if len(panes) > 1 {
			name = fmt.Sprintf("%d:(%d panes)", i, len(panes))
		}

		switch {
		case i == activeTab:
			parts = append(parts, activeTabStyle.Render(name))
		case allDead(panes):
			parts = append(parts, deadTabStyle.Render(name))
		default:
			parts = append(parts, tabStyle.Render(name))
		}
	}

	left := strings.Join(parts, "")

	hint := "Ctrl+B ? help"
	if !mouseEnabled {
		hint = "mouse:off  Ctrl+B ? help"
	}
	right := helpHint.Render(hint)
	if prefixMode {
		right = prefixIndicator
	}

	gap := width - lipgloss.Width(left) - lipgloss.Width(right)
	if gap < 0 {
		gap = 0
	}

	return left + strings.Repeat(" ", gap) + right
}

func allDead(panes []*TermPane) bool {
	for _, p := range panes {
		if !p.IsDead() {
			return false
		}
	}
	return true
}

// renderSidebar produces a left-side panel with starship-style status icons
// and a console list.
//
// Icon legend:
//
//	◆  sessions (alive/total)
//	◈  listeners
//	⇌  pipelines
func renderSidebar(tabs []*LayoutNode, activeTab int, focusedID int, state SidebarState, width, height int) string {
	dim := lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	cyan := lipgloss.NewStyle().Foreground(lipgloss.Color("6")).Bold(true)
	green := lipgloss.NewStyle().Foreground(lipgloss.Color("2"))
	yellow := lipgloss.NewStyle().Foreground(lipgloss.Color("3"))
	purple := lipgloss.NewStyle().Foreground(lipgloss.Color("5"))

	var lines []string

	// Title + starship-style status in one compact block
	title := cyan.Render(" ◆ IoM")
	lines = append(lines, title)

	// Status line: ◆ 3/5  ◈ 2  ⇌ 1
	status := fmt.Sprintf(" %s %s  %s %s  %s %s",
		green.Render("◆"), green.Render(fmt.Sprintf("%d/%d", state.SessionAlive, state.SessionTotal)),
		yellow.Render("◈"), yellow.Render(fmt.Sprintf("%d", state.ListenerCount)),
		purple.Render("⇌"), purple.Render(fmt.Sprintf("%d", state.PipelineCount)),
	)
	lines = append(lines, status)
	lines = append(lines, dim.Render(strings.Repeat("─", width)))

	// Console section header
	lines = append(lines, lipgloss.NewStyle().Bold(true).Width(width).Render("  Consoles"))

	// Console entries
	for i, tab := range tabs {
		panes := tab.Panes()
		for _, p := range panes {
			prefix := "  "
			if p.ID() == focusedID {
				prefix = "► "
			}

			name := p.Name()
			if p.IsDead() {
				name += " ✗"
			}

			style := lipgloss.NewStyle().Width(width)
			if i == activeTab && p.ID() == focusedID {
				style = style.
					Foreground(lipgloss.Color("0")).
					Background(lipgloss.Color("6"))
			}
			lines = append(lines, style.Render(prefix+name))
		}
	}

	// Session list
	if len(state.Sessions) > 0 {
		lines = append(lines, dim.Render(strings.Repeat("─", width)))
		lines = append(lines, lipgloss.NewStyle().Bold(true).Width(width).Render("  Sessions"))
		for _, s := range state.Sessions {
			indicator := green.Render("●")
			age := green.Render(s.LastSeen)
			if !s.Alive {
				indicator = dim.Render("○")
				age = dim.Render("✗")
			}
			// Truncate name to fit sidebar
			name := s.Name
			maxName := width - 10 // space for indicator + os + age
			if maxName < 4 {
				maxName = 4
			}
			if len(name) > maxName {
				name = name[:maxName-1] + "…"
			}
			entry := fmt.Sprintf(" %s %-*s %s %s", indicator, maxName, name, dim.Render(s.OS), age)
			lines = append(lines, entry)
		}
	}

	// Pad remaining height.
	for len(lines) < height-2 {
		lines = append(lines, strings.Repeat(" ", width))
	}

	// Keybinding hints at bottom.
	lines = append(lines, dim.Render(strings.Repeat("─", width)))
	hints := dim.Width(width).Render("c:new s:sess ?:help")
	lines = append(lines, hints)

	if len(lines) > height {
		lines = lines[:height]
	}
	return strings.Join(lines, "\n")
}
