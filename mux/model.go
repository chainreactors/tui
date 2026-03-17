package mux

import (
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// tickMsg drives periodic screen refreshes.
type tickMsg time.Time

// Mux is the top-level Bubble Tea model that manages multiple terminal panes
// arranged in tabs, each tab containing a layout tree of split panes.
type Mux struct {
	tabs      []*LayoutNode
	activeTab int
	focusedID int // pane that currently receives input
	nextID    int // monotonic pane ID allocator

	prefixMode bool
	prefixKey  byte
	keyMap     map[byte]MuxAction

	width, height   int
	sidebarWidth    int
	refreshInterval time.Duration

	paneFactory        PaneFactory
	sessionPaneFactory SessionPaneFactory
	sidebarState       SidebarState

	// Overlay state — at most one overlay is active at a time.
	overlayMode overlayType
	picker      *PickerState

	quitting bool
}

type overlayType int

const (
	overlayNone overlayType = iota
	overlayHelp
	overlaySessionPicker
	overlayPaneList
)

// SessionInfo describes one session for sidebar display.
type SessionInfo struct {
	ID       string
	Name     string
	OS       string // short: "lin"/"win"/"mac"
	LastSeen string // relative: "5s"/"2m"/"1h"
	Alive    bool
	HasPane  bool // true if a pane is already bound to this session
}

// SidebarState holds global status counters displayed in the sidebar.
type SidebarState struct {
	SessionAlive  int
	SessionTotal  int
	ListenerCount int
	PipelineCount int
	Sessions      []SessionInfo
}

// SetSidebarState updates the global status counters shown in the sidebar.
// Safe to call from any goroutine.
func (m *Mux) SetSidebarState(s SidebarState) {
	m.sidebarState = s
}

// New creates a new Mux with the given options. A PaneFactory must be provided
// via WithPaneFactory so the Mux knows how to create subprocess panes.
func New(opts ...Option) *Mux {
	m := &Mux{
		prefixKey:       0x02, // Ctrl+B
		keyMap:          DefaultKeyMap,
		sidebarWidth:    20,
		refreshInterval: 50 * time.Millisecond,
		width:           80,
		height:          24,
	}
	for _, opt := range opts {
		opt(m)
	}
	return m
}

// Init implements tea.Model. It creates the first pane and starts the
// refresh ticker.
func (m *Mux) Init() tea.Cmd {
	cmds := []tea.Cmd{m.tickCmd()}

	if m.paneFactory != nil && len(m.tabs) == 0 {
		w, h := m.paneArea()
		pane, err := m.paneFactory(m.nextID, w, h)
		if err == nil {
			m.nextID++
			pane.Focus()
			m.focusedID = pane.ID()
			m.tabs = append(m.tabs, NewLeaf(pane))
		}
	}

	return tea.Batch(cmds...)
}

// Update implements tea.Model.
func (m *Mux) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.resizeAll()
		return m, nil

	case tickMsg:
		m.reapDeadPanes()
		m.drainMuxCmds()
		if m.quitting {
			return m, tea.Quit
		}
		return m, m.tickCmd()

	case tea.KeyMsg:
		return m.handleKey(msg)

	case tea.MouseMsg:
		if msg.Button == tea.MouseButtonLeft && msg.Action == tea.MouseActionPress {
			return m.handleClick(msg.X, msg.Y)
		}
		// Forward scroll wheel as mouse escape sequences to the focused pane's PTY.
		// Uses SGR (1006) mouse encoding: \x1b[<button;x;y;M/m
		if msg.Button == tea.MouseButtonWheelUp || msg.Button == tea.MouseButtonWheelDown {
			btn := 64 // wheel up
			if msg.Button == tea.MouseButtonWheelDown {
				btn = 65
			}
			seq := fmt.Sprintf("\x1b[<%d;%d;%dM", btn, msg.X+1, msg.Y+1)
			m.forwardBytes([]byte(seq))
		}
	}

	return m, nil
}

// View implements tea.Model.
func (m *Mux) View() string {
	if m.quitting {
		return ""
	}

	contentW, contentH := m.paneArea()
	statusH := 1

	// Render the active tab's layout.
	var content string
	if m.activeTab < len(m.tabs) {
		content = m.tabs[m.activeTab].Render(contentW, contentH-statusH, m.focusedID)
	}

	// Render sidebar.
	var sidebar string
	if m.sidebarWidth > 0 {
		sidebar = renderSidebar(m.tabs, m.activeTab, m.focusedID, m.sidebarState, m.sidebarWidth, contentH-statusH)
		sep := renderVerticalSep(contentH - statusH)
		content = lipgloss.JoinHorizontal(lipgloss.Top, sidebar, sep, content)
	}

	// Render status bar.
	bar := renderStatusBar(m.tabs, m.activeTab, m.focusedID, m.prefixMode, m.width)

	view := lipgloss.JoinVertical(lipgloss.Left, content, bar)

	// Render overlay on top if active.
	switch m.overlayMode {
	case overlayHelp:
		view = renderOverlay(view, "Mux Keybindings", helpContent, m.width, m.height)
	case overlaySessionPicker, overlayPaneList:
		if m.picker != nil {
			pickerContent := m.picker.Render(m.width - 10)
			view = renderOverlay(view, m.picker.Title, pickerContent, m.width, m.height)
		}
	}

	return view
}

// Run starts the Bubble Tea program with alt screen enabled.
func (m *Mux) Run() error {
	p := tea.NewProgram(m, tea.WithAltScreen(), tea.WithMouseCellMotion())
	_, err := p.Run()
	m.closeAll()
	return err
}

// --- Key handling ---

func (m *Mux) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// If an overlay is active, route keys to it.
	if m.overlayMode != overlayNone {
		return m.handleOverlayKey(msg)
	}

	// In prefix mode, the next key is a mux command.
	if m.prefixMode {
		m.prefixMode = false
		action := m.resolveAction(msg)
		return m.execAction(action, msg)
	}

	// Check for prefix key.
	if msg.Type == tea.KeyCtrlB && m.prefixKey == 0x02 {
		m.prefixMode = true
		return m, nil
	}
	// Generic prefix key check for non-default keys.
	if len(msg.Runes) == 0 && byte(msg.Type) == m.prefixKey {
		m.prefixMode = true
		return m, nil
	}

	// Forward input to the focused pane.
	m.forwardKey(msg)
	return m, nil
}

func (m *Mux) handleOverlayKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch m.overlayMode {
	case overlayHelp:
		// Any key closes help.
		m.overlayMode = overlayNone
		return m, nil

	case overlaySessionPicker, overlayPaneList:
		if m.picker != nil {
			_, close := m.picker.HandleKey(msg.String())
			if close {
				m.overlayMode = overlayNone
				m.picker = nil
			}
		}
		return m, nil
	}
	return m, nil
}

func (m *Mux) resolveAction(msg tea.KeyMsg) MuxAction {
	var b byte
	if len(msg.Runes) > 0 {
		b = byte(msg.Runes[0])
	} else {
		b = byte(msg.Type)
	}
	if action, ok := m.keyMap[b]; ok {
		return action
	}
	return ActionNone
}

func (m *Mux) execAction(action MuxAction, msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch action {
	case ActionNextTab:
		m.switchTab((m.activeTab + 1) % max(len(m.tabs), 1))
	case ActionPrevTab:
		m.switchTab((m.activeTab - 1 + len(m.tabs)) % max(len(m.tabs), 1))
	case ActionNewPane:
		return m, m.createPane()
	case ActionClosePane:
		m.closeFocusedPane()
	case ActionSplitH:
		return m, m.splitFocused(Horizontal)
	case ActionSplitV:
		return m, m.splitFocused(Vertical)
	case ActionFocusNext:
		m.cycleFocus(1)
	case ActionSessionPicker:
		m.openSessionPicker()
	case ActionPaneList:
		m.openPaneList()
	case ActionHelp:
		m.overlayMode = overlayHelp
	case ActionQuit:
		m.quitting = true
		return m, tea.Quit
	case ActionNone:
		// Not a mux command — forward both prefix key and this key.
		m.forwardByte(m.prefixKey)
		m.forwardKey(msg)
	}
	return m, nil
}

// --- Pane management ---

func (m *Mux) createPane() tea.Cmd {
	if m.paneFactory == nil {
		return nil
	}

	w, h := m.paneArea()
	pane, err := m.paneFactory(m.nextID, w, h)
	if err != nil {
		return nil
	}
	m.nextID++

	// Blur current, focus new.
	m.blurFocused()
	pane.Focus()
	m.focusedID = pane.ID()

	// Add as a new tab.
	m.tabs = append(m.tabs, NewLeaf(pane))
	m.activeTab = len(m.tabs) - 1

	return nil
}

func (m *Mux) splitFocused(dir Direction) tea.Cmd {
	if m.paneFactory == nil || m.activeTab >= len(m.tabs) {
		return nil
	}

	w, h := m.paneArea()
	pane, err := m.paneFactory(m.nextID, w/2, h)
	if err != nil {
		return nil
	}
	m.nextID++

	tab := m.tabs[m.activeTab]
	tab.Split(m.focusedID, dir, pane)

	// Move focus to the new pane.
	m.blurFocused()
	pane.Focus()
	m.focusedID = pane.ID()

	m.resizeAll()
	return nil
}

func (m *Mux) closeFocusedPane() {
	if m.activeTab >= len(m.tabs) {
		return
	}

	tab := m.tabs[m.activeTab]
	pane := tab.FindPane(m.focusedID)
	if pane != nil {
		pane.Close()
	}

	// If only one pane in the tab and it's a leaf, remove the whole tab.
	if tab.IsLeaf() {
		m.tabs = append(m.tabs[:m.activeTab], m.tabs[m.activeTab+1:]...)
		if len(m.tabs) == 0 {
			m.quitting = true
			return
		}
		if m.activeTab >= len(m.tabs) {
			m.activeTab = len(m.tabs) - 1
		}
	} else {
		tab.Remove(m.focusedID)
	}

	// Focus the first pane in the current tab.
	m.focusFirst()
	m.resizeAll()
}

func (m *Mux) switchTab(idx int) {
	if idx < 0 || idx >= len(m.tabs) {
		return
	}
	m.blurFocused()
	m.activeTab = idx
	m.focusFirst()
}

func (m *Mux) cycleFocus(delta int) {
	if m.activeTab >= len(m.tabs) {
		return
	}
	panes := m.tabs[m.activeTab].Panes()
	if len(panes) == 0 {
		return
	}

	// Find current index.
	cur := 0
	for i, p := range panes {
		if p.ID() == m.focusedID {
			cur = i
			break
		}
	}

	m.blurFocused()
	next := (cur + delta + len(panes)) % len(panes)
	panes[next].Focus()
	m.focusedID = panes[next].ID()
}

func (m *Mux) focusFirst() {
	if m.activeTab >= len(m.tabs) {
		return
	}
	panes := m.tabs[m.activeTab].Panes()
	if len(panes) > 0 {
		panes[0].Focus()
		m.focusedID = panes[0].ID()
	}
}

func (m *Mux) blurFocused() {
	for _, tab := range m.tabs {
		if p := tab.FindPane(m.focusedID); p != nil {
			p.Blur()
			return
		}
	}
}

// --- Mouse handling ---

type hitZone int

const (
	hitNone hitZone = iota
	hitConsole
	hitSession
)

func (m *Mux) handleClick(x, y int) (tea.Model, tea.Cmd) {
	if m.sidebarWidth <= 0 || x >= m.sidebarWidth {
		return m, nil
	}
	zone, index := m.sidebarHitTest(y)
	switch zone {
	case hitConsole:
		m.switchTab(index)
	case hitSession:
		if index < len(m.sidebarState.Sessions) {
			s := m.sidebarState.Sessions[index]
			m.handleMuxCmd(MuxCmd{Action: "MuxOpen", Arg: s.ID})
		}
	}
	return m, nil
}

// sidebarHitTest maps a Y coordinate to a sidebar zone and index.
// Layout: title(0) status(1) sep(2) header(3) consoles... sep sessions...
func (m *Mux) sidebarHitTest(y int) (hitZone, int) {
	consoleStart := 4
	consoleCount := 0
	for _, tab := range m.tabs {
		consoleCount += len(tab.Panes())
	}

	// +1 sep +1 header before sessions
	sessionStart := consoleStart + consoleCount + 2

	if y >= consoleStart && y < consoleStart+consoleCount {
		return hitConsole, y - consoleStart
	}
	if len(m.sidebarState.Sessions) > 0 && y >= sessionStart {
		return hitSession, y - sessionStart
	}
	return hitNone, 0
}

// --- Input forwarding ---

func (m *Mux) forwardKey(msg tea.KeyMsg) {
	raw := KeyToBytes(msg)
	if raw == nil {
		return
	}
	m.forwardBytes(raw)
}

func (m *Mux) forwardByte(b byte) {
	m.forwardBytes([]byte{b})
}

func (m *Mux) forwardBytes(data []byte) {
	if m.activeTab >= len(m.tabs) {
		return
	}
	pane := m.tabs[m.activeTab].FindPane(m.focusedID)
	if pane != nil {
		pane.WriteInput(data)
	}
}

// --- Layout helpers ---

func (m *Mux) paneArea() (width, height int) {
	w := m.width
	if m.sidebarWidth > 0 {
		w -= m.sidebarWidth + 1 // +1 for separator
	}
	if w < 1 {
		w = 1
	}
	return w, m.height
}

func (m *Mux) resizeAll() {
	w, h := m.paneArea()
	statusH := 1
	for _, tab := range m.tabs {
		tab.Resize(w, h-statusH)
	}
}

func (m *Mux) closeAll() {
	for _, tab := range m.tabs {
		for _, p := range tab.Panes() {
			p.Close()
		}
	}
}

func (m *Mux) openSessionPicker() {
	var items []PickerItem
	for _, s := range m.sidebarState.Sessions {
		icon := "●"
		color := "2"
		if !s.Alive {
			icon = "○"
			color = "8"
		}
		items = append(items, PickerItem{
			ID:    s.ID,
			Label: s.Name,
			Desc:  fmt.Sprintf("%s %s", s.OS, s.LastSeen),
			Icon:  icon,
			Color: color,
		})
	}
	m.picker = NewPicker("Select Session", items, func(item PickerItem) {
		if m.sessionPaneFactory != nil {
			w, h := m.paneArea()
			pane, err := m.sessionPaneFactory(m.nextID, item.ID, w, h)
			if err == nil {
				m.nextID++
				m.blurFocused()
				pane.Focus()
				m.focusedID = pane.ID()
				m.tabs = append(m.tabs, NewLeaf(pane))
				m.activeTab = len(m.tabs) - 1
				m.resizeAll()
			}
		}
	})
	m.overlayMode = overlaySessionPicker
}

func (m *Mux) openPaneList() {
	var items []PickerItem
	for i, tab := range m.tabs {
		for _, p := range tab.Panes() {
			icon := "◆"
			color := "6"
			if p.IsDead() {
				icon = "✗"
				color = "8"
			}
			desc := ""
			if i == m.activeTab && p.ID() == m.focusedID {
				desc = "(active)"
			}
			items = append(items, PickerItem{
				ID:    fmt.Sprintf("%d", p.ID()),
				Label: p.Name(),
				Desc:  desc,
				Icon:  icon,
				Color: color,
			})
		}
	}
	m.picker = NewPicker("Panes", items, func(item PickerItem) {
		// Find and focus the selected pane.
		for i, tab := range m.tabs {
			for _, p := range tab.Panes() {
				if fmt.Sprintf("%d", p.ID()) == item.ID {
					m.blurFocused()
					m.activeTab = i
					p.Focus()
					m.focusedID = p.ID()
					return
				}
			}
		}
	})
	m.picker.Hint = "Enter: focus  Esc: cancel"
	m.overlayMode = overlayPaneList
}

// reapDeadPanes removes panes whose subprocess has exited.
// If the active pane dies, focus moves to the next live pane.
// If all panes in a tab die, the tab is removed.
func (m *Mux) reapDeadPanes() {
	changed := false
	for i := len(m.tabs) - 1; i >= 0; i-- {
		tab := m.tabs[i]
		for _, p := range tab.Panes() {
			if !p.IsDead() {
				continue
			}
			p.Close()
			if tab.IsLeaf() {
				m.tabs = append(m.tabs[:i], m.tabs[i+1:]...)
			} else {
				tab.Remove(p.ID())
			}
			changed = true
		}
	}
	if !changed {
		return
	}
	if len(m.tabs) == 0 {
		m.quitting = true
		return
	}
	if m.activeTab >= len(m.tabs) {
		m.activeTab = len(m.tabs) - 1
	}
	m.focusFirst()
}

// drainMuxCmds processes any pending OSC commands from child panes.
func (m *Mux) drainMuxCmds() {
	for _, tab := range m.tabs {
		for _, pane := range tab.Panes() {
			for {
				select {
				case cmd := <-pane.MuxCmds:
					m.handleMuxCmd(cmd)
				default:
					goto next
				}
			}
		next:
		}
	}
}

func (m *Mux) handleMuxCmd(cmd MuxCmd) {
	switch cmd.Action {
	case "MuxRename":
		// Rename the pane that sent this command.
		for _, tab := range m.tabs {
			if p := tab.FindPane(cmd.PaneID); p != nil {
				p.SetName(cmd.Arg)
				return
			}
		}
	case "MuxOpen":
		// Open a new pane bound to a specific session.
		if m.sessionPaneFactory == nil {
			return
		}
		w, h := m.paneArea()
		pane, err := m.sessionPaneFactory(m.nextID, cmd.Arg, w, h)
		if err != nil {
			return
		}
		m.nextID++
		m.blurFocused()
		pane.Focus()
		m.focusedID = pane.ID()
		m.tabs = append(m.tabs, NewLeaf(pane))
		m.activeTab = len(m.tabs) - 1
		m.resizeAll()
	}
}

func (m *Mux) tickCmd() tea.Cmd {
	return tea.Tick(m.refreshInterval, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

var _ tea.Model = (*Mux)(nil)
