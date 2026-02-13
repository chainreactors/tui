package tui

import (
	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/lipgloss"
	etable "github.com/evertras/bubble-table/table"
)

// base styles
var (
	FootStyle = lipgloss.NewStyle().BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(lipgloss.Color("240")).
			BorderBottom(true).
			Bold(false)
	HeaderStyle = lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("240"))
	SelectStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("255")).
			Background(lipgloss.Color("24")).
			Bold(false)
	HelpStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#626262")).Render
)

// Default Styles
var (
	DefaultTableStyle = table.Styles{
		Selected: table.DefaultStyles().Selected.Foreground(lipgloss.Color("255")).
			Background(lipgloss.Color("24")).
			Bold(false),
		Header: table.DefaultStyles().Header.BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(lipgloss.Color("240")).
			BorderBottom(true).
			Bold(false),
		Cell: lipgloss.NewStyle().Padding(0, 1),
	}
	DefaultTableHighlineStyle = table.DefaultStyles().Selected.Foreground(lipgloss.Color("229")).
					Background(lipgloss.Color("107")).
					Bold(false)
	DocStyle          = lipgloss.NewStyle().Margin(1, 2)
	DefaultGroupStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#8BE9FD"))
	DefaultNameStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF79C6"))
	DefaultBorder     = etable.Border{
		Top:    lipgloss.RoundedBorder().Top,
		Left:   lipgloss.RoundedBorder().Left,
		Right:  lipgloss.RoundedBorder().Right,
		Bottom: lipgloss.RoundedBorder().Bottom,

		TopRight:    lipgloss.RoundedBorder().TopRight,
		TopLeft:     lipgloss.RoundedBorder().TopLeft,
		BottomRight: lipgloss.RoundedBorder().BottomLeft,
		BottomLeft:  lipgloss.RoundedBorder().BottomRight,

		TopJunction:    lipgloss.RoundedBorder().Top,
		LeftJunction:   lipgloss.RoundedBorder().Left,
		RightJunction:  lipgloss.RoundedBorder().Right,
		BottomJunction: "┴",
		InnerJunction:  "┬",
		InnerDivider:   "",
	}
)
