package tui

import (
	"fmt"
	"github.com/chainreactors/tui/utils"
	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"os"
)

type Mode int

const (
	ModeFunc Mode = iota
	ModeConsole
)

type Model struct {
	FuncModel   tea.Model
	Help        HelpModel
	console     tea.Model
	currentMode Mode
	isConsole   bool
}

func NewModel(funcModel tea.Model, handler func(value string)) Model {
	consoleModel := NewConsoleModel()
	consoleModel = consoleModel.OnEnter(handler)
	return Model{
		FuncModel: funcModel,
		Help:      NewHelpModel(),
		console:   consoleModel,
	}
}

func (t Model) Run() error {
	p := tea.NewProgram(t)
	_, err := p.Run()
	if err != nil {
		return err
	}
	return nil
}

func (t Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch t.currentMode {
		case ModeFunc:
			if msg.String() == ":" {
				t.currentMode = ModeConsole
				return t, nil
			}
		case ModeConsole:
			if msg.String() == "esc" {
				t.currentMode = ModeFunc
				return t, nil
			}
		}
		switch {
		case key.Matches(msg, utils.DefaultKeys.Help):
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
	switch t.currentMode {
	case ModeFunc:
		return t.FuncModel.View() + t.Help.View()
	case ModeConsole:
		return t.FuncModel.View() + t.Help.View() + t.console.View()
	default:
		return t.FuncModel.View() + t.Help.View()
	}
}

func (t Model) Init() tea.Cmd {
	return nil
}

func main() {
	err := os.Setenv("RUNEWIDTH_EASTASIAN", "0")
	if err != nil {
		fmt.Println("Error setting RUNEWIDTH_EASTASIAN variable:", err)
		return
	}
	err = os.Setenv("LC_CTYPE", "en_US.UTF-8")
	if err != nil {
		fmt.Println("Error setting LC_CTYPE variable:", err)
		return
	}
	newTable := NewTable([]table.Column{
		{Title: "Name", Width: 20},
		{Title: "IsDir", Width: 5},
		{Title: "Size", Width: 7},
		{Title: "ModTime", Width: 10},
		{Title: "Link", Width: 15},
	}, false)
	rows := []table.Row{
		{
			"h3zh1",
			"true",
			"17263",
			"2024.1.18",
			"",
		},
		{
			"h4zh1",
			"true",
			"17263",
			"2024.1.18",
			"",
		},
	}
	newTable.Rows = rows
	newTable.SetRows()
	tableModel := NewModel(newTable, newTable.ConsoleHandler)
	tableModel.Run()
}
