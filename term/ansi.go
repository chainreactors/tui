package term

import (
	"fmt"
	"io"
	"strings"
	"unicode"
	"unicode/utf8"
)

const (
	SyncBegin = "\x1b[?2026h"
	SyncEnd   = "\x1b[?2026l"
	EraseLine = "\x1b[2K"
	Carriage  = "\r"
	CursorUp  = "\x1b[1A"
)

func WriteSynced(w io.Writer, fn func()) {
	if w == nil {
		return
	}
	fmt.Fprint(w, SyncBegin)
	defer fmt.Fprint(w, SyncEnd)
	fn()
}

func EraseLines(w io.Writer, n int) {
	if n <= 0 {
		return
	}
	fmt.Fprint(w, Carriage+EraseLine)
	for i := 1; i < n; i++ {
		fmt.Fprint(w, CursorUp+EraseLine)
	}
}

func AnsiEscapeEnd(s string, start int) (int, bool) {
	if start >= len(s) || s[start] != '\x1b' {
		return 0, false
	}
	if start+1 >= len(s) {
		return start + 1, true
	}
	switch s[start+1] {
	case '[':
		for i := start + 2; i < len(s); i++ {
			if s[i] >= 0x40 && s[i] <= 0x7e {
				return i + 1, true
			}
		}
		return len(s), true
	case ']':
		for i := start + 2; i < len(s); i++ {
			switch {
			case s[i] == '\a':
				return i + 1, true
			case s[i] == '\x1b' && i+1 < len(s) && s[i+1] == '\\':
				return i + 2, true
			}
		}
		return len(s), true
	default:
		return start + 2, true
	}
}

func AnsiClosesStyle(seq string) bool {
	if strings.HasPrefix(seq, "\x1b]8;;") {
		return true
	}
	if len(seq) < 3 || seq[0] != '\x1b' || seq[1] != '[' || seq[len(seq)-1] != 'm' {
		return false
	}
	params := seq[2 : len(seq)-1]
	if params == "" {
		return true
	}
	for _, param := range strings.FieldsFunc(params, func(r rune) bool { return r == ';' || r == ':' }) {
		switch param {
		case "0", "22", "23", "24", "25", "27", "28", "29", "39", "49", "59":
			return true
		}
	}
	return false
}

func TrimANSIVisibleRight(line string) string {
	cut := 0
	extendCutWithANSI := false
	for i := 0; i < len(line); {
		if end, ok := AnsiEscapeEnd(line, i); ok {
			if extendCutWithANSI && AnsiClosesStyle(line[i:end]) {
				cut = end
			}
			i = end
			continue
		}
		r, size := utf8.DecodeRuneInString(line[i:])
		if r == utf8.RuneError && size == 1 {
			cut = i + size
			extendCutWithANSI = true
			i += size
			continue
		}
		end := i + size
		if unicode.IsSpace(r) {
			extendCutWithANSI = false
		} else {
			cut = end
			extendCutWithANSI = true
		}
		i = end
	}
	return line[:cut]
}
