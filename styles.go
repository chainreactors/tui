package tui

import (
	"github.com/chainreactors/logs"
	"github.com/charmbracelet/bubbles/table"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
)

const (
	Debug     logs.Level = 10
	Warn      logs.Level = 20
	Info      logs.Level = 30
	Error     logs.Level = 40
	Important logs.Level = 50
)

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
	DefaultLogStyle = map[logs.Level]string{

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
