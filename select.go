package tui

import (
	"github.com/chainreactors/tui/utils"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"strings"
)

func NewSelect(choices []string) *SelectModel {
	return &SelectModel{
		Choices: choices,
		Help:    NewHelpModel(),
	}
}

type SelectModel struct {
	Choices      []string
	selected     []string
	SelectedItem int
	KeyHandler   KeyHandler
	NewKey       tea.Key
	IsQuit       bool
	Title        string
	Help         HelpModel
}

func (m *SelectModel) Init() tea.Cmd {
	m.SelectedItem = 0
	return nil
}

func (m *SelectModel) View() string {
	var view strings.Builder
	view.WriteString(m.Title)
	view.WriteRune('\n')
	if len(m.selected) > 0 {
		view.WriteString("[x] ")
		view.WriteString(m.selected[0])
		view.WriteRune('\n')
	} else {
		for i, choice := range m.Choices {
			if i == m.SelectedItem {
				view.WriteString("[x] ")
			} else {
				view.WriteString("[ ] ")
			}
			view.WriteString(choice)
			view.WriteRune('\n')
		}
	}
	view.WriteString(m.Help.Model.View(m.Help.Keys))
	view.WriteRune('\n')
	return view.String()
}

type KeyHandler func(*SelectModel, tea.Msg) (tea.Model, tea.Cmd)

func (m *SelectModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, utils.DefaultKeys.Help):
			m.Help.Model.ShowAll = !m.Help.Model.ShowAll
			return m, nil
		}
		switch msg.Type {
		case tea.KeyEsc:
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
				m.selected = []string{m.Choices[m.SelectedItem]}
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
