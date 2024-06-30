package tui

import (
	"fmt"
	"github.com/chainreactors/tui/utils"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

func NewInput(title string) *InputModel {
	ti := textinput.New()
	return &InputModel{
		TextInput: ti,
		Title:     title,
		Help:      NewHelpModel(),
	}
}

type (
	errMsg error
)

type InputModel struct {
	TextInput textinput.Model
	Title     string
	err       error
	handler   func()
	Help      HelpModel
}

func (m InputModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m InputModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, utils.DefaultKeys.Help):
			m.Help.Model.ShowAll = !m.Help.Model.ShowAll
			return m, nil
		}
		switch msg.Type {
		case tea.KeyEsc, tea.KeyCtrlC, tea.KeyCtrlQ:
			return m, tea.Quit
		case tea.KeyEnter:
			m.handler()
			return m, tea.Quit
		}

	// We handle errors just like any other message
	case errMsg:
		m.err = msg
		return m, nil
	}

	m.TextInput, cmd = m.TextInput.Update(msg)
	return m, cmd
}

func (m InputModel) View() string {
	return fmt.Sprintf(
		"%s\n\n%s\n\n%s",
		m.Title,
		m.TextInput.View(),
		m.Help.View(),
	) + "\n"
}

func (m InputModel) SetHandler(handler func()) {
	m.handler = handler
}
