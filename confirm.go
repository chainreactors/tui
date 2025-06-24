package tui

import (
	"bytes"
	"fmt"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"os"
	"strings"
)

func NewConfirm(title string) *ConfirmModel {
	ti := textinput.New()
	ti.Placeholder = "y/n"
	ti.Focus()
	return &ConfirmModel{
		textInput: ti,
		Title:     title,
	}
}

type ConfirmModel struct {
	textInput textinput.Model
	Title     string
	quitting  bool
	confirmed bool
	handle    func()
	*bytes.Buffer
}

func (m *ConfirmModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m *ConfirmModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyEsc, tea.KeyCtrlC, tea.KeyCtrlQ:
			m.quitting = true
			return m, tea.Quit
		case tea.KeyEnter:
			input := strings.ToLower(strings.TrimSpace(m.textInput.Value()))
			if input == "yes" || input == "y" {
				if m.handle != nil {
					m.handle()
				}
				m.confirmed = true
				m.quitting = true
				return m, tea.Quit
			} else if input == "no" || input == "n" {
				m.confirmed = false
				m.quitting = true
				return m, tea.Quit
			}
		}
	}

	var cmd tea.Cmd
	m.textInput, cmd = m.textInput.Update(msg)
	return m, cmd
}

func (m *ConfirmModel) View() string {
	if m.quitting {
		if m.confirmed {
			return "You chose: Yes\n"
		}
		return "You chose: No\n"
	}
	return fmt.Sprintf(
		"%s(yes/no)\n\n%s\n", m.Title, m.textInput.View())
}

func (m *ConfirmModel) Run() error {
	p := tea.NewProgram(m)
	_, err := p.Run()
	if err != nil {
		return err
	}
	fmt.Printf(HelpStyle("<Press enter to exit>\n"))
	os.Stdin.Write([]byte("\n"))
	ClearLines(1)
	return nil
}

func (m *ConfirmModel) SetHandle(handle func()) {
	m.handle = handle
}

func (m *ConfirmModel) GetConfirmed() bool {
	return m.confirmed
}
