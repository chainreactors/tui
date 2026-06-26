package console

import (
	"fmt"
	"strings"

	"github.com/chainreactors/tui/readline"
)

// PasteReferenceConfig controls whether pasted text is replaced with a short
// reference in the editable prompt while remaining resolvable to the original
// pasted content. When ShouldReference is nil, only multiline paste is
// referenced.
type PasteReferenceConfig struct {
	Enabled         bool
	Format          func(index, lines int) string
	ShouldReference func(text string) bool
}

// EnablePasteReferences enables paste references and configures readline to
// transform bracketed paste payloads before they are inserted.
func (c *Console) EnablePasteReferences(config PasteReferenceConfig) {
	if c == nil || c.shell == nil {
		return
	}
	config.Enabled = true
	c.pasteMu.Lock()
	c.pasteConfig = config
	if c.pasteRefs == nil {
		c.pasteRefs = make(map[string]string)
	}
	c.pasteMu.Unlock()
	c.shell.SetPasteTransformer(c.referencePastedText)
	_ = c.shell.Config.Set("enable-bracketed-paste", true)
}

// DisablePasteReferences disables prompt paste references.
func (c *Console) DisablePasteReferences() {
	if c == nil || c.shell == nil {
		return
	}
	c.pasteMu.Lock()
	c.pasteConfig = PasteReferenceConfig{}
	c.pasteMu.Unlock()
	c.shell.SetPasteTransformer(nil)
}

// Readline reads one line and coalesces raw multiline paste into a paste
// reference when paste references are enabled.
func (c *Console) Readline() (string, error) {
	if c == nil || c.shell == nil {
		return "", nil
	}
	line, err := c.shell.Readline()
	if err != nil {
		return line, err
	}
	return c.CoalescePastedInput(line), nil
}

// CoalescePastedInput handles terminals that do not emit bracketed paste
// markers. In that mode readline returns the first line and keeps the rest of
// the paste in its key buffer.
func (c *Console) CoalescePastedInput(firstLine string) string {
	if c == nil || c.shell == nil || !c.pasteReferencesEnabled() {
		return firstLine
	}
	if !c.shell.Keys.HasPendingInput() {
		return firstLine
	}
	remaining := readline.NormalizePastedText(string(c.shell.Keys.Read()))
	if remaining == "" {
		return firstLine
	}
	text := readline.NormalizePastedText(strings.TrimRight(firstLine, "\r\n") + "\n" + remaining)
	return c.referencePastedText(text)
}

// ResolvePasteReferences expands paste references in input back to the original
// pasted content.
func (c *Console) ResolvePasteReferences(input string) string {
	if c == nil || input == "" {
		return input
	}
	c.pasteMu.Lock()
	replacements := make(map[string]string, len(c.pasteRefs))
	for placeholder, text := range c.pasteRefs {
		replacements[placeholder] = text
	}
	c.pasteMu.Unlock()

	expanded := input
	for placeholder, text := range replacements {
		expanded = strings.ReplaceAll(expanded, placeholder, text)
	}
	return expanded
}

func (c *Console) pasteReferencesEnabled() bool {
	if c == nil {
		return false
	}
	c.pasteMu.Lock()
	defer c.pasteMu.Unlock()
	return c.pasteConfig.Enabled
}

func (c *Console) referencePastedText(text string) string {
	text = readline.NormalizePastedText(text)
	if text == "" {
		return ""
	}

	c.pasteMu.Lock()
	defer c.pasteMu.Unlock()
	config := c.pasteConfig
	if !config.Enabled || !shouldReferencePastedText(text, config) {
		return text
	}

	c.pasteCounter++
	placeholder := pastedTextPlaceholder(c.pasteCounter, pastedTextLineCount(text), config)
	if c.pasteRefs == nil {
		c.pasteRefs = make(map[string]string)
	}
	c.pasteRefs[placeholder] = text
	return placeholder
}

func shouldReferencePastedText(text string, config PasteReferenceConfig) bool {
	if config.ShouldReference != nil {
		return config.ShouldReference(text)
	}
	return strings.Contains(text, "\n")
}

func pastedTextLineCount(text string) int {
	text = strings.TrimRight(text, "\n")
	if text == "" {
		return 0
	}
	return strings.Count(text, "\n") + 1
}

func pastedTextPlaceholder(id, lines int, config PasteReferenceConfig) string {
	if config.Format != nil {
		return config.Format(id, lines)
	}
	if lines <= 1 {
		return fmt.Sprintf("[Pasted text #%d]", id)
	}
	return fmt.Sprintf("[Pasted text #%d +%d lines]", id, lines)
}
