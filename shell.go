package tui

import (
	"fmt"
	"github.com/atotto/clipboard"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"regexp"
	"strings"
	"sync"
	"time"
)

// 常量定义
const (
	// 双击时间阈值（毫秒）
	doubleClickThreshold = 300

	// 最大输出行数
	maxOutputLines = 1000

	// 最大历史记录数
	maxHistorySize = 100

	// 默认输入宽度
	defaultInputWidth = 50

	// 默认视口尺寸
	defaultViewportWidth  = 80
	defaultViewportHeight = 20
)

// ShellHandlers 定义shell的事件处理函数
type ShellHandlers struct {
	OnCommand       func(command string) error // 处理命令输入
	OnConnect       func() error               // 处理连接
	OnDisconnect    func() error               // 处理断开连接
	OnResize        func(cols, rows int) error // 处理窗口大小改变
	OnTabSend       func(current string) error // 直接将 Tab 发送到远端（优先级高于本地补全）
	OnCtrlC         func() error               // 处理Ctrl+C
	OnCtrlL         func() error               // 处理Ctrl+L (清屏)
	OnArrowUpSend   func(current string) error // 发送上箭头到远端
	OnArrowDownSend func(current string) error // 发送下箭头到远端
}

// ShellModel represents an interactive shell interface
type ShellModel struct {
	// Core components
	input    textinput.Model
	viewport viewport.Model

	// Session management
	sessionID string
	active    bool
	connected bool

	// Output management
	output      []string
	outputMutex sync.RWMutex

	// Shell state
	prompt string
	width  int
	height int

	// Event handlers - 由外部设置
	handlers *ShellHandlers

	// Command history
	history    []string
	historyIdx int

	// 去重：抑制远端对同一命令的首次回显
	echoToSuppress   string
	suppressNextEcho bool

	// Tab 补全跟踪：按下 Tab 后，下一条响应若携带文本则用于更新输入行
	completionPending bool

	// 历史命令跟踪：按下上下箭头后，下一条响应用于更新输入行
	historyPending bool

	// 为 Tab 同步远端缓冲：记录最近一次已注入到远端的本地输入
	injectedBuffer string

	// Auto-follow output (stay at bottom). When user scrolls up, disable
	follow bool

	// 文本选择相关字段
	selecting     bool  // 是否正在选择文本
	selectStart   int   // 选择开始位置
	selectEnd     int   // 选择结束位置
	lastClickPos  int   // 上次点击位置
	clickCount    int   // 点击次数（用于双击、三击等）
	lastClickTime int64 // 上次点击时间

	// Styles
	promptStyle    lipgloss.Style
	outputStyle    lipgloss.Style
	sessionStyle   lipgloss.Style
	errorStyle     lipgloss.Style
	selectionStyle lipgloss.Style // 选中文本的样式
}

// NewShell creates a new interactive shell model
func NewShell(sessionID string, handlers *ShellHandlers) *ShellModel {
	// Initialize input component
	input := textinput.New()
	// 仅显示自定义提示符，不使用默认的 '>' 提示和占位符
	input.Prompt = ""
	input.Placeholder = ""
	input.Focus()
	input.CharLimit = 0 // No limit
	input.Width = defaultInputWidth

	// Initialize viewport for output
	vp := viewport.New(defaultViewportWidth, defaultViewportHeight)
	// 普通终端样式：无边框、无装饰
	vp.Style = lipgloss.NewStyle()

	shell := &ShellModel{
		input:      input,
		viewport:   vp,
		sessionID:  sessionID,
		prompt:     "$ ",
		output:     make([]string, 0),
		handlers:   handlers,
		history:    make([]string, 0),
		historyIdx: 0,
		follow:     true, // 默认跟随输出到底部

		// 文本选择字段初始化
		selecting:     false,
		selectStart:   0,
		selectEnd:     0,
		lastClickPos:  0,
		clickCount:    0,
		lastClickTime: 0,

		// Default styles
		promptStyle:    lipgloss.NewStyle().Foreground(lipgloss.Color("205")).Bold(true),
		outputStyle:    lipgloss.NewStyle().Foreground(lipgloss.Color("246")),
		sessionStyle:   lipgloss.NewStyle().Foreground(lipgloss.Color("36")).Bold(true),
		errorStyle:     lipgloss.NewStyle().Foreground(lipgloss.Color("196")),
		selectionStyle: lipgloss.NewStyle().Background(lipgloss.Color("57")).Foreground(lipgloss.Color("255")),
	}

	return shell
}

// SetHandlers 设置事件处理器
func (s *ShellModel) SetHandlers(handlers *ShellHandlers) {
	s.handlers = handlers
}

// SetPrompt 设置提示符
func (s *ShellModel) SetPrompt(prompt string) {
	s.prompt = prompt
	// 根据当前窗口宽度，立即调整输入区域宽度
	if s.width > 0 {
		s.input.Width = s.width - lipgloss.Width(s.prompt)
	}
}

// GetInputValue 读取当前输入行内容
func (s *ShellModel) GetInputValue() string {
	return s.input.Value()
}

// 标记：已将 current 注入到远端缓冲（用于后续 Enter 仅发送换行）
func (s *ShellModel) MarkInjectedBuffer(current string) {
	s.injectedBuffer = current
}

// 判断：当前输入是否与已注入缓冲一致
func (s *ShellModel) RemoteBufferMatches(current string) bool {
	return s.injectedBuffer != "" && (current == s.injectedBuffer)
}

// 清除注入标记
func (s *ShellModel) ClearInjectedBuffer() { s.injectedBuffer = "" }

// ShellMsg represents messages for the shell component
type ShellMsg struct {
	Type string
	Data interface{}
}

// Message types
const (
	ShellMsgOutput       = "output"
	ShellMsgError        = "error"
	ShellMsgConnected    = "connected"
	ShellMsgDisconnected = "disconnected"
	ShellMsgPromptChange = "prompt_change"
)

// Init initializes the shell component
func (s *ShellModel) Init() tea.Cmd {
	// 尝试连接
	if s.handlers != nil && s.handlers.OnConnect != nil {
		go func() {
			if err := s.handlers.OnConnect(); err != nil {
				s.AddError(fmt.Sprintf("Connection failed: %v", err))
			} else {
				s.SetConnected(true)
			}
		}()
	}

	return textinput.Blink
}

// Update handles messages and updates the shell state
func (s *ShellModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.MouseMsg:
		return s.handleMouseEvent(msg)

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			if s.HasSelection() {
				s.ClearSelection()
			} else {
				s.input.SetValue("")
			}
			return s, nil
		case "ctrl+l":
			s.clearOutput()
			return s, nil
		case "esc":
			if s.HasSelection() {
				s.ClearSelection()
			}
			return s, nil
		case "ctrl+d":
			if s.handlers != nil && s.handlers.OnDisconnect != nil {
				_ = s.handlers.OnDisconnect()
			}
			return s, tea.Quit
		case "tab":
			if s.handlers != nil && s.handlers.OnTabSend != nil {
				s.completionPending = true
				if err := s.handlers.OnTabSend(s.input.Value()); err != nil {
					s.AddError(fmt.Sprintf("Tab send failed: %v", err))
					s.completionPending = false
				}
			}
			return s, nil
		case "up":
			if s.handlers != nil && s.handlers.OnArrowUpSend != nil {
				s.historyPending = true
				if err := s.handlers.OnArrowUpSend(s.input.Value()); err != nil {
					s.AddError(fmt.Sprintf("Arrow up send failed: %v", err))
					s.historyPending = false
				}
			}
			return s, nil
		case "down":
			if s.handlers != nil && s.handlers.OnArrowDownSend != nil {
				s.historyPending = true
				if err := s.handlers.OnArrowDownSend(s.input.Value()); err != nil {
					s.AddError(fmt.Sprintf("Arrow down send failed: %v", err))
					s.historyPending = false
				}
			}
			return s, nil
		case "enter":
			command := s.input.Value()
			if command != "" {
				s.addToHistory(command)
				s.echoCommandLine(command)
				if s.handlers != nil && s.handlers.OnCommand != nil {
					if err := s.handlers.OnCommand(command); err != nil {
						s.AddError(fmt.Sprintf("Command failed: %v", err))
					}
				}
				s.ClearInjectedBuffer()
				s.input.SetValue("")
			}
			return s, nil
		}
		// 更新输入组件
		var cmd tea.Cmd
		s.input, cmd = s.input.Update(msg)
		return s, cmd

	case tea.WindowSizeMsg:
		s.width = msg.Width
		s.height = msg.Height
		s.viewport.Width = msg.Width
		s.viewport.Height = msg.Height - 2
		s.input.Width = msg.Width - lipgloss.Width(s.prompt)
		s.viewport.MouseWheelEnabled = true
		return s, nil

	case ShellMsg:
		return s.handleShellMsg(msg)
	}

	// Update viewport
	var cmd tea.Cmd
	s.viewport, cmd = s.viewport.Update(msg)
	if cmd != nil {
		cmds = append(cmds, cmd)
	}

	return s, tea.Batch(cmds...)
}

// handleMouseEvent 处理鼠标事件
func (s *ShellModel) handleMouseEvent(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	// 处理滚轮事件
	if msg.Button == tea.MouseButtonWheelUp || msg.Button == tea.MouseButtonWheelDown {
		s.follow = false
		prevYOffset := s.viewport.YOffset
		var cmd tea.Cmd
		s.viewport, cmd = s.viewport.Update(msg)

		if s.viewport.YOffset != prevYOffset && !s.viewport.AtBottom() {
			s.follow = false
		}
		if s.viewport.AtBottom() {
			s.follow = true
		}

		if cmd != nil {
			return s, cmd
		}
		return s, nil
	}

	// 处理鼠标左键事件
	if msg.Button == tea.MouseButtonLeft {
		clickPos := s.calculateClickPosition(msg.X, msg.Y)

		switch msg.Action {
		case tea.MouseActionPress:
			currentTime := time.Now().UnixMilli()
			if currentTime-s.lastClickTime < doubleClickThreshold {
				s.clickCount++
			} else {
				s.clickCount = 1
			}
			s.lastClickTime = currentTime
			s.selecting = true
			s.selectStart = clickPos
			s.selectEnd = clickPos
			s.lastClickPos = clickPos

		case tea.MouseActionRelease:
			if s.selecting {
				s.selectEnd = clickPos
				if s.selectStart > s.selectEnd {
					s.selectStart, s.selectEnd = s.selectEnd, s.selectStart
				}
				if s.selectStart != s.selectEnd {
					selectedText := s.getSelectedText()
					if selectedText != "" {
						if err := clipboard.WriteAll(selectedText); err != nil {
							s.AddError(fmt.Sprintf("Failed to copy to clipboard: %v", err))
						}
						s.ClearSelection()
					}
				}
			}

		case tea.MouseActionMotion:
			if s.selecting {
				s.selectEnd = clickPos
			}
		}
	}

	return s, nil
}

// View renders the shell interface
func (s *ShellModel) View() string {
	if s.width == 0 {
		return "Initializing shell..."
	}

	// 更新 viewport 内容
	s.updateViewportContent()

	// 直接返回 viewport 视图，让它包含所有内容（包括输入行）

	return s.viewport.View()
}

// AddOutput adds output to the shell (thread-safe)
func (s *ShellModel) AddOutput(text string) {
	s.outputMutex.Lock()
	defer s.outputMutex.Unlock()
	s.addOutput(text)
}

// stripANSI 移除 ANSI 控制序列（颜色、光标等），避免错位
var ansiRegexp = regexp.MustCompile(`\x1b\[[0-9;?]*[a-zA-Z@-~]`)

var cursorPattern = regexp.MustCompile(`(\x1b|\\u\{1b\})\[\d+;\d+H`)

var promptPatterns = []struct {
	re   *regexp.Regexp
	desc string
}{
	{regexp.MustCompile(`([A-Z]:\\[^>]*>)\s*$`), "Windows drive prompt"},
	{regexp.MustCompile(`(PS\s+[A-Z]:\\[^>]*>\s*)$`), "PowerShell prompt"},
	{regexp.MustCompile(`(.+@.+:\S*[\$#>])\s*$`), "user@host:path prompt"},
	{regexp.MustCompile(`(.+:\S*[\$#>])\s*$`), "path prompt"},
	{regexp.MustCompile(`([\$#>])\s*$`), "simple prompt"},
	{regexp.MustCompile(`(.+[\$#>])\s*$`), "general prompt"},
	{regexp.MustCompile(`(.+λ)\s*$`), "lambda prompt"},
}

func stripANSI(s string) string {
	return ansiRegexp.ReplaceAllString(s, "")
}

// AddError adds error output to the shell (thread-safe)
func (s *ShellModel) AddError(text string) {
	s.outputMutex.Lock()
	defer s.outputMutex.Unlock()
	s.addOutput(s.errorStyle.Render("ERROR: " + text))
}

// SetConnected updates connection status
func (s *ShellModel) SetConnected(connected bool) {
	s.connected = connected
}

// GetSessionID returns the session ID
func (s *ShellModel) GetSessionID() string {
	return s.sessionID
}

// CompletionPending 返回是否处于待补全状态
func (s *ShellModel) CompletionPending() bool {
	return s.completionPending
}

// HistoryPending 返回是否处于待历史命令状态
func (s *ShellModel) HistoryPending() bool {
	return s.historyPending
}

// ApplyCompletionText 提取补全结果并更新输入行
func (s *ShellModel) ApplyCompletionText(text string) {
	//cleanText := stripANSI(text)
	// 查找最后一个光标位置移动后的文本
	// 光标移动格式: \u{1b}[行;列H 或 \x1b[行;列H
	matches := cursorPattern.FindAllStringIndex(text, -1)

	if len(matches) > 0 {
		// 取最后一个光标移动后的文本
		lastMatch := matches[len(matches)-1]
		afterCursor := text[lastMatch[1]:]

		// 去掉颜色控制字符
		completion := strings.TrimPrefix(afterCursor, "?25h")
		completion = strings.TrimPrefix(completion, "m")
		completion = stripANSI(completion)
		if completion != "" {
			s.input.SetValue(completion)
		}
	} else {
		// 如果没有找到光标移动，使用清理后的文本
		s.input.SetValue(text)
	}

	s.completionPending = false
}

// ApplyHistoryText 处理历史命令响应并更新输入行
func (s *ShellModel) ApplyHistoryText(text string) {
	// 去除 ANSI 转义序列，得到纯文本
	cleanText := stripANSI(text)

	// 历史命令响应通常直接包含完整的命令文本
	// 设置为输入行内容
	s.input.SetValue(cleanText)

	// 清除待历史命令标记
	s.historyPending = false
}

// ClearCompletionPending 手动清除待补全标记（备用）
func (s *ShellModel) ClearCompletionPending() {
	s.completionPending = false
}

// ClearHistoryPending 手动清除待历史命令标记（备用）
func (s *ShellModel) ClearHistoryPending() {
	s.historyPending = false
}

// Internal methods

// extractPromptFromLine 从单行文本中提取prompt
func (s *ShellModel) extractPromptFromLine(line string) string {
	if line == "" {
		return ""
	}

	// 常见的prompt模式，按优先级排序
	for _, pattern := range promptPatterns {
		matches := pattern.re.FindStringSubmatch(line)
		if len(matches) > 1 {
			// 返回捕获的组（提取的prompt部分）
			prompt := strings.TrimSpace(matches[1])
			if prompt != "" {
				return prompt
			}
		}
	}

	return ""
}

func (s *ShellModel) addOutput(text string) {
	// 去除 ANSI 转义序列，避免显示错位
	text = stripANSI(text)
	// Split multiline text
	lines := strings.Split(text, "\n")

	// 先检查最后一行是否包含prompt，如果有就提取并移除
	var extractedPrompt string
	if len(lines) > 0 {
		lastLine := strings.TrimRight(lines[len(lines)-1], "\r")
		if prompt := s.extractPromptFromLine(lastLine); prompt != "" {
			extractedPrompt = prompt
			// 移除最后一行（包含prompt的行）
			lines = lines[:len(lines)-1]
		}
	}

	// 添加剩余的输出行
	for _, line := range lines {
		trimmed := strings.TrimRight(line, "\r")
		if s.suppressNextEcho && trimmed == s.echoToSuppress {
			// 抑制远端首次对同一命令的回显
			s.suppressNextEcho = false
			continue
		}
		s.output = append(s.output, line)
	}

	// 如果提取到了prompt，设置它
	if extractedPrompt != "" {
		s.SetPrompt(extractedPrompt + " ") // 确保prompt后有空格
	}

	// Keep last 1000 lines to prevent memory issues
	if len(s.output) > maxOutputLines {
		s.output = s.output[len(s.output)-maxOutputLines:]
	}

	// 添加输出后，自动滚动到顶部（让新输入框可见）
	s.follow = false
	s.viewport.GotoTop()
}

func (s *ShellModel) clearOutput() {
	s.outputMutex.Lock()
	defer s.outputMutex.Unlock()
	s.output = make([]string, 0)
}

func (s *ShellModel) addToHistory(command string) {
	// 避免重复的连续命令
	if len(s.history) == 0 || s.history[len(s.history)-1] != command {
		s.history = append(s.history, command)

		// 限制历史记录数量
		if len(s.history) > maxHistorySize {
			s.history = s.history[1:]
		}
	}

	// 重置历史索引
	s.historyIdx = len(s.history)
}

// echoCommandLine 在输出区域立即回显一行：提示符 + 命令，并设置去重标记
func (s *ShellModel) echoCommandLine(command string) {
	line := lipgloss.JoinHorizontal(lipgloss.Left, s.promptStyle.Render(s.prompt), command)
	// 在 UI 线程内调用，无需加锁
	s.addOutput(line)
	// 记录去重目标：远端通常会仅回显命令本身
	s.echoToSuppress = command
	s.suppressNextEcho = true
}

func (s *ShellModel) updateViewportContent() {
	s.outputMutex.RLock()
	defer s.outputMutex.RUnlock()

	// 构建完整内容：输出历史 + 在最后一行添加当前输入
	var lines []string

	// 1. 添加所有输出内容
	lines = append(lines, s.output...)

	// 2. 总是在新行添加当前输入（使用提取的prompt）
	prefix := s.promptStyle.Render(s.prompt)
	inputLine := lipgloss.JoinHorizontal(lipgloss.Left, prefix, s.input.View())
	lines = append(lines, inputLine)

	// 3. 设置 viewport 内容（使用带高亮的内容）
	content := strings.Join(lines, "\n")

	// 如果有文本选择，应用高亮
	if s.HasSelection() {
		content = s.renderContentWithSelection()
	}

	// 记录当前位置，避免刷新内容时丢失滚动位置
	prev := s.viewport.YOffset
	s.viewport.SetContent(content)

	if s.follow {
		// 跟随输出到底（包括输入行）
		s.viewport.GotoBottom()
	} else {
		// 保持用户当前查看的位置
		s.viewport.SetYOffset(prev)
	}
}

func (s *ShellModel) handleShellMsg(msg ShellMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case ShellMsgOutput:
		if text, ok := msg.Data.(string); ok {
			s.AddOutput(text)
		}
	case ShellMsgError:
		if text, ok := msg.Data.(string); ok {
			s.AddError(text)
		}
	case ShellMsgConnected:
		s.SetConnected(true)
		s.AddOutput(s.sessionStyle.Render("Connected to shell session: " + s.sessionID))
	case ShellMsgDisconnected:
		s.SetConnected(false)
		s.AddOutput(s.errorStyle.Render("Disconnected from shell session"))
	case ShellMsgPromptChange:
		if prompt, ok := msg.Data.(string); ok {
			s.SetPrompt(prompt)
			// 更新 input 宽度以贴合新的前缀
			s.input.Width = s.width - lipgloss.Width(s.prompt)
		}
	}
	return s, nil
}

// 文本选择相关的辅助方法

// calculateClickPosition 计算鼠标点击位置对应的文本位置
func (s *ShellModel) calculateClickPosition(x, y int) int {
	// 获取当前内容
	content := s.getViewportContent()
	if content == "" {
		return 0
	}

	// 将内容按行分割
	lines := strings.Split(content, "\n")

	// 计算点击位置对应的行号（考虑滚动偏移）
	lineNum := y + s.viewport.YOffset

	// 确保行号在有效范围内
	if lineNum < 0 {
		lineNum = 0
	}
	if lineNum >= len(lines) {
		lineNum = len(lines) - 1
	}

	// 计算在该行中的列位置
	colPos := x
	if colPos < 0 {
		colPos = 0
	}

	// 计算总的文本位置
	textPos := 0
	for i := 0; i < lineNum; i++ {
		textPos += len(lines[i]) + 1 // +1 for newline
	}

	// 确保列位置不超过当前行的长度
	if colPos > len(lines[lineNum]) {
		colPos = len(lines[lineNum])
	}

	textPos += colPos

	// 确保位置在内容范围内
	if textPos > len(content) {
		textPos = len(content)
	}

	return textPos
}

// selectWordAt 在指定位置选择单词
func (s *ShellModel) selectWordAt(pos int) {
	// 获取当前内容
	content := s.getViewportContent()
	if pos >= len(content) {
		return
	}

	// 向前查找单词边界
	start := pos
	for start > 0 && isWordChar(content[start-1]) {
		start--
	}

	// 向后查找单词边界
	end := pos
	for end < len(content) && isWordChar(content[end]) {
		end++
	}

	s.selectStart = start
	s.selectEnd = end
}

// selectLineAt 在指定位置选择整行
func (s *ShellModel) selectLineAt(pos int) {
	content := s.getViewportContent()
	if pos >= len(content) {
		return
	}

	// 向前查找行首
	start := pos
	for start > 0 && content[start-1] != '\n' {
		start--
	}

	// 向后查找行尾
	end := pos
	for end < len(content) && content[end] != '\n' {
		end++
	}

	s.selectStart = start
	s.selectEnd = end
}

// isWordChar 判断字符是否为单词字符
func isWordChar(ch byte) bool {
	return (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') ||
		(ch >= '0' && ch <= '9') || ch == '_' || ch == '-'
}

// renderContentWithSelection 渲染带高亮的文本内容
func (s *ShellModel) renderContentWithSelection() string {
	content := s.getViewportContent()
	if content == "" {
		return ""
	}

	// 如果没有选择，直接返回原内容
	if !s.HasSelection() {
		return content
	}

	// 确保选择范围有效
	if s.selectStart >= len(content) || s.selectEnd > len(content) || s.selectStart >= s.selectEnd {
		return content
	}

	// 分割内容：选择前 + 选中部分 + 选择后
	before := content[:s.selectStart]
	selected := content[s.selectStart:s.selectEnd]
	after := content[s.selectEnd:]

	// 应用高亮样式到选中部分
	highlighted := s.selectionStyle.Render(selected)

	// 组合所有部分
	return before + highlighted + after
}

// getViewportContent 获取 viewport 的完整内容
func (s *ShellModel) getViewportContent() string {
	s.outputMutex.RLock()
	defer s.outputMutex.RUnlock()

	// 构建完整内容：输出历史 + 当前输入
	var lines []string
	lines = append(lines, s.output...)

	// 添加当前输入行
	prefix := s.promptStyle.Render(s.prompt)
	inputLine := lipgloss.JoinHorizontal(lipgloss.Left, prefix, s.input.View())
	lines = append(lines, inputLine)

	return strings.Join(lines, "\n")
}

// getSelectedText 获取选中的文本
func (s *ShellModel) getSelectedText() string {
	content := s.getViewportContent()
	if s.selectStart >= len(content) || s.selectEnd > len(content) {
		return ""
	}

	if s.selectStart >= s.selectEnd {
		return ""
	}

	return content[s.selectStart:s.selectEnd]
}

// ClearSelection 清除当前选择
func (s *ShellModel) ClearSelection() {
	s.selecting = false
	s.selectStart = 0
	s.selectEnd = 0
}

// HasSelection 检查是否有选中的文本
func (s *ShellModel) HasSelection() bool {
	return s.selecting && s.selectStart != s.selectEnd
}

// GetSelectionRange 获取选择范围
func (s *ShellModel) GetSelectionRange() (start, end int) {
	return s.selectStart, s.selectEnd
}

// GetSelectedText 获取选中的文本（公共方法）
func (s *ShellModel) GetSelectedText() string {
	return s.getSelectedText()
}

// IsSelecting 检查是否正在选择文本
func (s *ShellModel) IsSelecting() bool {
	return s.selecting
}

func (s *ShellModel) Run() error {
	program := tea.NewProgram(s, tea.WithAltScreen(), tea.WithMouseCellMotion())

	_, err := program.Run()
	return err
}
