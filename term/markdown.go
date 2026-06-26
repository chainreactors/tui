package term

import (
	"os"
	"strings"
	"sync"

	"github.com/charmbracelet/glamour"
	"github.com/muesli/termenv"
	"golang.org/x/term"
)

var (
	mdRenderer    *glamour.TermRenderer
	mdRendererErr error
	mdRendererW   int
	mdRendererMu  sync.Mutex
)

func RenderMarkdown(content string, enabled bool) string {
	content = strings.TrimSpace(content)
	if content == "" {
		return ""
	}
	if !enabled {
		return content
	}
	r, err := getMarkdownRenderer()
	if err != nil {
		return content
	}
	rendered, err := r.Render(content)
	if err != nil {
		return content
	}
	rendered = strings.TrimSpace(TrimRenderedMarkdownLineEnds(rendered))
	if rendered == "" {
		return content
	}
	return rendered
}

func getMarkdownRenderer() (*glamour.TermRenderer, error) {
	w := TerminalWidth()
	mdRendererMu.Lock()
	defer mdRendererMu.Unlock()
	if mdRenderer != nil && w == mdRendererW {
		return mdRenderer, mdRendererErr
	}
	opts := []glamour.TermRendererOption{
		glamour.WithAutoStyle(),
		glamour.WithColorProfile(termenv.ColorProfile()),
		glamour.WithEmoji(),
	}
	if w > 0 {
		opts = append(opts, glamour.WithWordWrap(w))
	}
	mdRenderer, mdRendererErr = glamour.NewTermRenderer(opts...)
	mdRendererW = w
	return mdRenderer, mdRendererErr
}

func TerminalWidth() int {
	if w, _, err := term.GetSize(int(os.Stdout.Fd())); err == nil && w > 0 {
		return w
	}
	return 0
}

func TrimRenderedMarkdownLineEnds(s string) string {
	if s == "" {
		return ""
	}
	var b strings.Builder
	b.Grow(len(s))
	start := 0
	for start < len(s) {
		rel := strings.IndexByte(s[start:], '\n')
		if rel < 0 {
			b.WriteString(TrimANSIVisibleRight(s[start:]))
			break
		}
		end := start + rel
		b.WriteString(TrimANSIVisibleRight(s[start:end]))
		b.WriteByte('\n')
		start = end + 1
	}
	return b.String()
}
