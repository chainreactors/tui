package tui

import (
	"fmt"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

func NewInput(title string) *InputModel {
	ti := textinput.New()
	return &InputModel{
		TextInput: ti,
		Title:     title,
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
}

func (m InputModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m InputModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
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
		"%s\n\n%s\n\n",
		m.Title,
		m.TextInput.View(),
	) + "\n"
}

func (m InputModel) SetHandler(handler func()) *InputModel {
	m.handler = handler
	return &m
}
