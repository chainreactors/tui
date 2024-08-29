package tui

import (
	"fmt"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
	"os"
)

var (
	output    = termenv.NewOutput(os.Stdout)
	profile   = termenv.ColorProfile()
	Normal    = lipgloss.NewStyle().String()
	Bold      = lipgloss.NewStyle().Bold(true).String()
	Underline = lipgloss.NewStyle().Underline(true).String()
	Blue      = profile.Color("#3398DA")
	Yellow    = profile.Color("#F1C40F")
	Purple    = profile.Color("#8D44AD")
	Green     = profile.Color("#2FCB71")
	Red       = profile.Color("#E74C3C")
	Gray      = profile.Color("#BDC3C7")
	Cyan      = profile.Color("#1ABC9C")
	Orange    = profile.Color("#E67E22")
	Black     = profile.Color("#000000")
) // You can use ANSI color codes directly

var (
	Reset      = output.Reset
	Clear      = output.ClearLine
	UpN        = output.CursorPrevLine
	Down       = output.CursorNextLine
	ClearLines = output.ClearLines
)

//var ClientPrompt = AdaptTermColor()

// adaptTermColor - Adapt term color
// TODO: Adapt term color by term(fork grumble ColorTableFg)
func AdaptTermColor(prompt string) string {
	var color string
	if termenv.HasDarkBackground() {
		color = fmt.Sprintf("\033[37m%s> \033[0m", prompt)
	} else {
		color = fmt.Sprintf("\033[30m%s> \033[0m", prompt)
	}
	return color
}

func AdaptSessionColor(prePrompt, sId string) string {
	var sessionPrompt string
	runes := []rune(sId)
	if termenv.HasDarkBackground() {
		sessionPrompt = fmt.Sprintf("\033[37m%s [%s]> \033[0m", prePrompt, string(runes))
	} else {
		sessionPrompt = fmt.Sprintf("\033[30m%s [%s]> \033[0m", prePrompt, string(runes))
	}
	return sessionPrompt
}

func NewSessionColor(prePrompt, sId string) string {
	var sessionPrompt string
	runes := []rune(sId)
	if termenv.HasDarkBackground() {
		sessionPrompt = fmt.Sprintf("%s [%s]> ", DefaultGroupStyle.Render(prePrompt), DefaultNameStyle.Render(string(runes)))
	} else {
		sessionPrompt = fmt.Sprintf("%s [%s]> ", DefaultGroupStyle.Render(prePrompt), DefaultNameStyle.Render(string(runes)))
	}
	return sessionPrompt
}
