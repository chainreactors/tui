package mux

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var (
	overlayBorder = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("6")).
			Padding(1, 2)

	overlayTitle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("6"))

	overlayDim = lipgloss.NewStyle().
			Foreground(lipgloss.Color("8"))
)

// renderOverlay renders a centered floating panel over the given background.
func renderOverlay(bg string, title string, content string, bgWidth, bgHeight int) string {
	panel := overlayBorder.Render(overlayTitle.Render(title) + "\n" + content)

	panelW := lipgloss.Width(panel)
	panelH := lipgloss.Height(panel)

	// Center the panel.
	padLeft := (bgWidth - panelW) / 2
	padTop := (bgHeight - panelH) / 2
	if padLeft < 0 {
		padLeft = 0
	}
	if padTop < 0 {
		padTop = 0
	}

	// Split the background into lines and overlay the panel.
	bgLines := strings.Split(bg, "\n")
	for len(bgLines) < bgHeight {
		bgLines = append(bgLines, strings.Repeat(" ", bgWidth))
	}

	panelLines := strings.Split(panel, "\n")
	for i, pLine := range panelLines {
		row := padTop + i
		if row >= len(bgLines) {
			break
		}
		// Replace the section of the background line with the panel line.
		bgLine := bgLines[row]
		bgRunes := []rune(bgLine)

		prefix := string(bgRunes[:min(padLeft, len(bgRunes))])
		suffixStart := padLeft + lipgloss.Width(pLine)
		suffix := ""
		if suffixStart < len(bgRunes) {
			suffix = string(bgRunes[suffixStart:])
		}

		bgLines[row] = prefix + pLine + suffix
	}

	return strings.Join(bgLines[:bgHeight], "\n")
}

