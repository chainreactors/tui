package tui

import (
	"fmt"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"strings"
)

func NewConfirm(title string) *ConfirmModel {
	ti := textinput.New()
	ti.Placeholder = "y/n"
	return &ConfirmModel{
		textInput: ti,
		Title:     title,
	}
}

type ConfirmModel struct {
	textInput textinput.Model
	Title     string
	quitting  bool
	Confirmed bool
}

func (m ConfirmModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m ConfirmModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyEsc, tea.KeyCtrlC, tea.KeyCtrlQ:
			m.quitting = true
			return m, tea.Quit
		case tea.KeyEnter:
			input := strings.ToLower(strings.TrimSpace(m.textInput.Value()))
			if input == "yes" || input == "y" {
				m.Confirmed = true
				m.quitting = true
				return m, tea.Quit
			} else if input == "no" || input == "n" {
				m.Confirmed = false
				m.quitting = true
				return m, tea.Quit
			}
		}
	}

	var cmd tea.Cmd
	m.textInput, cmd = m.textInput.Update(msg)
	return m, cmd
}

func (m ConfirmModel) View() string {
	if m.quitting {
		if m.Confirmed {
			return "You chose: Yes\n"
		}
		return "You chose: No\n"
	}
	return fmt.Sprintf(
		"%s(yes/no)\n\n%s\n", m.Title, m.textInput.View())
}
