package tui

import (
	"bytes"
	"fmt"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"os"
	"strings"
)

type Mode int

const (
	ModeFunc Mode = iota
	ModeConsole
)

type Model struct {
	FuncModel tea.Model
	Help      HelpModel
	*bytes.Buffer
	console     tea.Model
	currentMode Mode
	isConsole   bool
}

func NewModel(funcModel tea.Model, handler func(value string), isConsole bool, isShortHelp bool) Model {
	consoleModel := NewConsoleModel()
	consoleModel = consoleModel.OnEnter(handler)
	return Model{
		FuncModel: funcModel,
		Help:      NewHelpModel(isShortHelp),
		console:   consoleModel,
		isConsole: isConsole,
		Buffer:    bytes.NewBuffer([]byte{}),
	}
}

func (t Model) Run() error {
	p := tea.NewProgram(t)
	_, err := p.Run()
	if err != nil {
		return err
	}
	fmt.Printf(HelpStyle("<Press enter to exit>\n"))
	os.Stdin.Write([]byte("\n"))
	ClearLines(1)
	return nil
}

func (t Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch t.currentMode {
		case ModeFunc:
			if msg.String() == ":" {
				if t.isConsole {
					t.currentMode = ModeConsole
					return t, nil
				}
			} else if msg.String() == "enter" || msg.String() == "ctrl+c" || msg.String() == "ctrl+q" ||
				msg.String() == "esc" {
				t.Help.Quitting = true
			}
		case ModeConsole:
			if msg.String() == "esc" {
				t.currentMode = ModeFunc
				return t, nil
			}
		}
		switch {
		case key.Matches(msg, DefaultKeys.Help):
			t.Help.Model.ShowAll = !t.Help.Model.ShowAll
			return t, nil
		}
	}
	switch t.currentMode {
	case ModeFunc:
		newFuncModel, cmd := t.FuncModel.Update(msg)
		t.FuncModel = newFuncModel
		return t, cmd
	case ModeConsole:
		newConsoleModel, _ := t.console.Update(msg)
		t.console = newConsoleModel
		switch msg := msg.(type) {
		case tea.KeyMsg:
			if msg.Type == tea.KeyEnter {
				t.currentMode = ModeFunc
			}
			t.console.Update(tea.Quit())
		}
		return t, nil
	}
	return t, nil
}

func (t Model) View() string {
	var s strings.Builder
	s.WriteString(t.FuncModel.View())
	s.WriteString(t.Help.View())
	switch t.currentMode {
	case ModeFunc:
	case ModeConsole:
		s.WriteString(t.console.View())
	default:
	}

	s.WriteString(t.Buffer.String())

	return s.String()
}

func (t Model) Init() tea.Cmd {
	return nil
}
