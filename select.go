package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"strings"
)

func NewSelect(choices []string) *SelectModel {
	return &SelectModel{
		Choices: choices,
	}
}

type SelectModel struct {
	Choices      []string
	Selected     string
	SelectedItem int
	KeyHandler   KeyHandler
	NewKey       tea.Key
	IsQuit       bool
	Title        string
}

func (m *SelectModel) Init() tea.Cmd {
	m.SelectedItem = 0
	return nil
}

func (m *SelectModel) View() string {
	var view strings.Builder
	view.WriteString(m.Title)
	view.WriteRune('\n')
	for i, choice := range m.Choices {
		if i == m.SelectedItem {
			view.WriteString("[x] ")
		} else {
			view.WriteString("[ ] ")
		}
		view.WriteString(choice)
		view.WriteRune('\n')
	}

	return view.String()
}

type KeyHandler func(*SelectModel, tea.Msg) (tea.Model, tea.Cmd)

func (m *SelectModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyEsc, tea.KeyCtrlC, tea.KeyCtrlQ:
			return m, tea.Quit
		case tea.KeyUp:
			m.SelectedItem--
			if m.SelectedItem < 0 {
				m.SelectedItem = len(m.Choices) - 1
			}
			return m, nil
		case tea.KeyDown:
			m.SelectedItem++
			if m.SelectedItem >= len(m.Choices) {
				m.SelectedItem = 0
			}
			return m, nil
		case tea.KeyEnter:
			if m.SelectedItem >= 0 && m.SelectedItem < len(m.Choices) {
				m.Selected = m.Choices[m.SelectedItem]
			}
			return m, tea.Quit
		case m.NewKey.Type:
			newModel, _ := m.KeyHandler(m, msg)
			if m.IsQuit {
				return newModel, tea.Quit
			}
			return newModel, nil
		}
	}

	return m, nil
}
