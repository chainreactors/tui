package mux

import (
	"github.com/charmbracelet/lipgloss"
)

var helpContent = func() string {
	h := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("6"))
	d := lipgloss.NewStyle().Foreground(lipgloss.Color("7"))
	dim := lipgloss.NewStyle().Foreground(lipgloss.Color("8"))

	return h.Render("Navigation") + "\n" +
		d.Render("  n / p      ") + dim.Render("next / prev tab") + "\n" +
		d.Render("  o          ") + dim.Render("cycle focus in split") + "\n" +
		d.Render("  w          ") + dim.Render("pane navigator") + "\n" +
		"\n" +
		h.Render("Pane Management") + "\n" +
		d.Render("  c          ") + dim.Render("new console") + "\n" +
		d.Render("  s          ") + dim.Render("session picker → new pane") + "\n" +
		d.Render("  x          ") + dim.Render("close focused pane") + "\n" +
		d.Render(`  "          `) + dim.Render("split vertical") + "\n" +
		d.Render("  %          ") + dim.Render("split horizontal") + "\n" +
		"\n" +
		h.Render("Other") + "\n" +
		d.Render("  [          ") + dim.Render("scrollback mode") + "\n" +
		d.Render("  m          ") + dim.Render("toggle mouse capture") + "\n" +
		d.Render("  ?          ") + dim.Render("this help") + "\n" +
		d.Render("  q          ") + dim.Render("quit multiplexer") + "\n" +
		"\n" +
		dim.Render("  Press any key to close")
}()
