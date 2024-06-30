package tui

import (
	"github.com/chainreactors/tui/utils"
	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
)

type HelpModel struct {
	Keys     utils.DefaultKeyMap
	Model    help.Model
	lastKey  string
	quitting bool
}

func NewHelpModel(isShortHelp bool) HelpModel {
	if isShortHelp {
		return HelpModel{
			Keys:  utils.DefaultShortKeys,
			Model: help.New(),
		}
	} else {
		return HelpModel{
			Keys:  utils.DefaultKeys,
			Model: help.New(),
		}

	}
}

func (m HelpModel) Init() tea.Cmd {
	return nil
}

func (m HelpModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		// If we set a width on the Help menu it can gracefully truncate
		// its view as needed.
		m.Model.Width = msg.Width

	case tea.KeyMsg:
		switch {
		case key.Matches(msg, m.Keys.Up):
			m.lastKey = "↑"
		case key.Matches(msg, m.Keys.Down):
			m.lastKey = "↓"
		case key.Matches(msg, m.Keys.Left):
			m.lastKey = "←"
		case key.Matches(msg, m.Keys.Right):
			m.lastKey = "→"
		case key.Matches(msg, m.Keys.Help):
			m.Model.ShowAll = !m.Model.ShowAll
		case key.Matches(msg, m.Keys.Quit):
			m.quitting = true
			return m, tea.Quit
		}
	}

	return m, nil
}

func (m HelpModel) View() string {
	helpView := m.Model.View(m.Keys)
	return "\n" + helpView + "\n"
}

func (m HelpModel) SetKeys(keys utils.DefaultKeyMap) {
	m.Keys = keys
}
