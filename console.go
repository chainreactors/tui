package tui

import (
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

type ConsoleModel struct {
	Model        textinput.Model
	EnterHandler func(value string)
}

func NewConsoleModel() ConsoleModel {
	ti := textinput.New()
	ti.Placeholder = "Enter command"
	ti.Focus()
	ti.CharLimit = 156
	ti.Width = 20
	return ConsoleModel{
		Model: ti,
	}
}

func (m ConsoleModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m ConsoleModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyEsc, tea.KeyCtrlC, tea.KeyCtrlQ:
			return m, tea.Quit
		case tea.KeyEnter:
			m.EnterHandler(m.Model.Value())
			m.Model.Reset()
			m.Model.Update(tea.Quit())
			return m, nil
		}
	}
	m.Model, cmd = m.Model.Update(msg)
	return m, cmd
}

func (m ConsoleModel) View() string {
	return m.Model.View()
}

func (m ConsoleModel) OnEnter(handler func(value string)) ConsoleModel {
	m.EnterHandler = handler
	return m
}
