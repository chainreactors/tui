package tui

import (
	"github.com/charmbracelet/bubbles/help"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
)

type HelpModel struct {
	Keys     DefaultKeyMap
	Model    help.Model
	lastKey  string
	Quitting bool
}

func NewHelpModel(isShortHelp bool) HelpModel {
	if isShortHelp {
		return HelpModel{
			Keys:  DefaultShortKeys,
			Model: help.New(),
		}
	} else {
		return HelpModel{
			Keys:  DefaultKeys,
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
			m.Quitting = true
			return m, tea.Quit
		}
		switch msg.Type {
		case tea.KeyEnter:
			m.Quitting = true
			return m, tea.ClearScreen
		}
	}

	return m, nil
}

func (m HelpModel) View() string {
	var helpView string
	if m.Quitting {
		helpView = ""
	} else {
		helpView = m.Model.View(m.Keys)
	}
	return "\n" + helpView + "\n"
}

func (m HelpModel) SetKeys(keys DefaultKeyMap) {
	m.Keys = keys
}
