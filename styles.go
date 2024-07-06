package tui

import (
	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
)

const (
	Debug     Level = 10
	Warn      Level = 20
	Info      Level = 30
	Error     Level = 40
	Important Level = 50
)

type Level int

// base styles
var (
	HeaderStyle = lipgloss.NewStyle().BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(lipgloss.Color("240")).
			BorderBottom(true).
			Bold(false)
	FootStyle = lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("240"))
	SelectStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("229")).
			Background(lipgloss.Color("57")).
			Bold(false)
	HelpStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#626262")).Render
)

// Default Styles
var (
	DefaultLogStyle = map[Level]string{

		Debug:     termenv.String(Rocket+"[+]").Bold().Background(Blue).String() + " %s ",
		Warn:      termenv.String(Zap+"[warn]").Bold().Background(Yellow).String() + " %s ",
		Important: termenv.String(Fire+"[*]").Bold().Background(Purple).String() + " %s ",
		Info:      termenv.String(HotSpring+"[i]").Bold().Background(Green).String() + " %s ",
		Error:     termenv.String(Monster+"[-]").Bold().Background(Red).String() + " %s ",
	}

	DefaultTableStyle = table.Styles{
		Selected: table.DefaultStyles().Selected.Foreground(lipgloss.Color("229")).
			Background(lipgloss.Color("57")).
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
	DocStyle = lipgloss.NewStyle().Margin(1, 2)
)
