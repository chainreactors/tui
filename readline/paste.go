package readline

import "strings"

const (
	BracketedPasteStart       = "\x1b[200~"
	BracketedPasteStartSuffix = "[200~"
	BracketedPasteEnd         = "\x1b[201~"
)

// SetPasteTransformer sets a function used to rewrite pasted text before it is
// inserted into the input buffer.
func (rl *Shell) SetPasteTransformer(fn func(text string) string) {
	if rl == nil {
		return
	}
	rl.PasteTransformer = fn
}

// InsertPastedText normalizes, transforms, and inserts pasted text into the
// current input buffer.
func (rl *Shell) InsertPastedText(text string) {
	if rl == nil {
		return
	}
	text = NormalizePastedText(text)
	if text == "" {
		return
	}
	if rl.PasteTransformer != nil {
		text = rl.PasteTransformer(text)
	}
	if text == "" {
		return
	}
	rl.History.Save()
	rl.cursor.InsertAt([]rune(text)...)
	rl.Display.Refresh()
}

// HandleBracketedPastePending handles bytes that remain after an Escape binding
// consumed the ESC byte from a bracketed paste start sequence.
func (rl *Shell) HandleBracketedPastePending(pending string) bool {
	if rl == nil {
		return false
	}

	switch {
	case strings.HasPrefix(pending, BracketedPasteStartSuffix):
		rl.insertBracketedPasteAfterStart(strings.TrimPrefix(pending, BracketedPasteStartSuffix))
		return true
	case strings.HasPrefix(pending, BracketedPasteStart):
		rl.insertBracketedPasteAfterStart(strings.TrimPrefix(pending, BracketedPasteStart))
		return true
	default:
		return false
	}
}

func (rl *Shell) insertBracketedPasteAfterStart(alreadyRead string) {
	text := alreadyRead
	rest := ""
	if idx := strings.Index(text, BracketedPasteEnd); idx >= 0 {
		rest = text[idx+len(BracketedPasteEnd):]
		text = text[:idx]
	} else {
		text += string(rl.Keys.ReadUntilSequence([]byte(BracketedPasteEnd)))
	}
	rl.InsertPastedText(text)
	if rest != "" {
		rl.Keys.Feed(true, []rune(rest)...)
	}
}

// NormalizePastedText normalizes CRLF and CR line endings to LF and trims
// trailing newlines added by terminal paste wrappers.
func NormalizePastedText(text string) string {
	text = strings.ReplaceAll(text, "\r\n", "\n")
	text = strings.ReplaceAll(text, "\r", "\n")
	return strings.TrimRight(text, "\n")
}
