package tui

import (
	"fmt"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"os"
)

type ListModel struct {
}

type Item struct {
	Ititle, Desc string
}

func (i Item) Title() string       { return i.Ititle }
func (i Item) Description() string { return i.Desc }
func (i Item) FilterValue() string { return i.Ititle }

type listModel struct {
	list list.Model
}

func (m listModel) Init() tea.Cmd {
	return nil
}

func (m listModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyCtrlC, tea.KeyEsc, tea.KeyCtrlQ:
			return m, tea.Quit
		}
	case tea.WindowSizeMsg:
		h, v := DocStyle.GetFrameSize()
		m.list.SetSize(msg.Width-h, msg.Height-v)
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m listModel) View() string {
	m.list.SetShowHelp(false)
	return DocStyle.Render(m.list.View())
}

func Newlist(items []list.Item) *listModel {
	return &listModel{
		list: list.New(items, list.NewDefaultDelegate(), 0, 0),
	}
}

func (m listModel) SetSpacing(i int) *listModel {
	d := list.NewDefaultDelegate()
	d.SetSpacing(i)
	m.list.SetDelegate(d)
	return &m
}

func (m listModel) SetHeight(i int) *listModel {
	m.list.SetHeight(i)
	return &m
}

func (m listModel) Run() error {
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
