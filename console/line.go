package console

import (
	"bytes"
	"errors"
	"regexp"
	"strings"
	"unicode/utf8"

	"mvdan.cc/sh/v3/syntax"
)

var (
	splitChars        = " \n\t"
	singleChar        = '\''
	doubleChar        = '"'
	escapeChar        = '\\'
	doubleEscapeChars = "$`\"\n\\"
)

var (
	errUnterminatedSingleQuote = errors.New("unterminated single-quoted string")
	errUnterminatedDoubleQuote = errors.New("unterminated double-quoted string")
	errUnterminatedEscape      = errors.New("unterminated backslash-escape")
)

type continuationState uint8

// Quote states intentionally match the regular single and double quotes
// supported by shellSplit; this console does not implement full Bash quoting.
const (
	continuationBare continuationState = iota
	continuationSingleQuoted
	continuationDoubleQuoted
	continuationComment
)

type continuationContext struct {
	state       continuationState
	atWordStart bool
}

// parse is in charge of removing all comments from the input line
// before execution, and if successfully parsed, split into words.
func (c *Console) parse(line string) (args []string, err error) {
	line, _ = scanLineContinuations(line)
	line, backslashPlaceholder := protectLineEndBackslashes(line)

	lineReader := strings.NewReader(line)
	parser := syntax.NewParser(syntax.KeepComments(false))

	// Parse the shell string a syntax, removing all comments.
	stmts, err := parser.Parse(lineReader, "")
	if err != nil {
		return nil, err
	}

	var parsedLine bytes.Buffer

	err = syntax.NewPrinter().Print(&parsedLine, stmts)
	if err != nil {
		return nil, err
	}

	// Split the line into shell words.
	parsed := strings.ReplaceAll(parsedLine.String(), backslashPlaceholder, string(escapeChar))
	return shellSplit(parsed)
	//return shellquote.Split(parsedLine.String())
}

func shellSplit(command string) (args []string, err error) {
	re := regexp.MustCompile(`[^\s"']+|"([^"]*)"|'([^']*)'`)
	matches := re.FindAllStringSubmatch(command, -1)

	var parts []string
	for _, match := range matches {
		if match[1] != "" { // Matched double-quoted part
			parts = append(parts, match[1])
		} else if match[2] != "" { // Matched single-quoted part
			parts = append(parts, match[2])
		} else { // Unquoted part
			parts = append(parts, match[0])
		}
	}
	return parts, nil
}

// acceptMultiline determines if the line just accepted is complete (in which case
// we should execute it), or incomplete (in which case we must read in multiline).
func (c *Console) acceptMultiline(line []rune) (accept bool) {
	if _, pending := scanLineContinuations(string(line)); pending {
		return false
	}

	// Errors are either: unterminated quotes, or unterminated escapes.
	_, _, err := split(string(line), false)
	if err == nil {
		return true
	}

	// Currently, unterminated quotes are obvious to treat: keep reading.
	switch err {
	case errUnterminatedDoubleQuote, errUnterminatedSingleQuote:
		return false
	}

	return true
}

// scanLineContinuations normalizes completed continuation markers and reports
// whether the final physical line ends with a pending marker. A marker consists
// of horizontal whitespace, one unquoted backslash, and optional trailing
// horizontal whitespace.
func scanLineContinuations(input string) (normalized string, pending bool) {
	var output strings.Builder
	output.Grow(len(input))

	context := continuationContext{
		state:       continuationBare,
		atWordStart: true,
	}
	start := 0

	for {
		lineEnd, newlineEnd, ok := nextPhysicalLine(input, start)
		if !ok {
			break
		}

		line := input[start:lineEnd]
		marker, markerContext, continuation := explicitContinuationMarker(line, context)
		if continuation {
			output.WriteString(line[:marker])
			context = markerContext
		} else {
			output.WriteString(input[start:newlineEnd])
			context = scanContinuationState(line, context)
			switch context.state {
			case continuationComment:
				context = continuationContext{state: continuationBare, atWordStart: true}
			case continuationBare:
				context.atWordStart = true
			}
		}

		start = newlineEnd
	}

	tail := input[start:]
	output.WriteString(tail)
	_, _, pending = explicitContinuationMarker(tail, context)

	return output.String(), pending
}

// protectLineEndBackslashes prevents the shell parser from applying POSIX
// backslash-newline semantics to lines that do not use an explicit marker.
// The placeholder is restored immediately after the parser removes comments.
func protectLineEndBackslashes(input string) (protected string, placeholder string) {
	placeholder = unusedBackslashPlaceholder(input)

	var output strings.Builder
	output.Grow(len(input))
	start := 0

	for {
		lineEnd, newlineEnd, ok := nextPhysicalLine(input, start)
		if !ok {
			break
		}

		line := input[start:lineEnd]
		end := len(line)
		for end > 0 && isHorizontalWhitespace(line[end-1]) {
			end--
		}

		runStart := end
		for runStart > 0 && line[runStart-1] == '\\' {
			runStart--
		}

		if runStart == end {
			output.WriteString(input[start:newlineEnd])
		} else {
			output.WriteString(line[:runStart])
			for i := runStart; i < end; i++ {
				output.WriteString(placeholder)
			}
			output.WriteString(line[end:])
			output.WriteString(input[lineEnd:newlineEnd])
		}

		start = newlineEnd
	}

	output.WriteString(input[start:])
	return output.String(), placeholder
}

func unusedBackslashPlaceholder(input string) string {
	placeholder := "\ue000"
	for strings.Contains(input, placeholder) {
		placeholder += "\ue001"
	}
	return placeholder
}

func explicitContinuationMarker(line string, initial continuationContext) (marker int, context continuationContext, ok bool) {
	end := len(line)
	for end > 0 && isHorizontalWhitespace(line[end-1]) {
		end--
	}

	if end == 0 || line[end-1] != '\\' {
		return 0, initial, false
	}

	marker = end - 1
	if marker == 0 || !isHorizontalWhitespace(line[marker-1]) {
		return 0, initial, false
	}

	context = scanContinuationState(line[:marker], initial)
	if context.state != continuationBare {
		return 0, context, false
	}

	return marker, context, true
}

func scanContinuationState(line string, context continuationContext) continuationContext {
	for i := 0; i < len(line); i++ {
		char := line[i]

		switch context.state {
		case continuationComment:
			return context
		case continuationSingleQuoted:
			if char == '\'' {
				context.state = continuationBare
			}
			continue
		case continuationDoubleQuoted:
			if char == '\\' && i+1 < len(line) && strings.ContainsRune(doubleEscapeChars, rune(line[i+1])) {
				i++
				continue
			}
			if char == '"' {
				context.state = continuationBare
			}
			continue
		}

		switch {
		case isHorizontalWhitespace(char), isShellControlCharacter(char):
			context.atWordStart = true
		case char == '\\':
			context.atWordStart = false
			if i+1 < len(line) {
				i++
			}
		case char == '\'':
			context.state = continuationSingleQuoted
			context.atWordStart = false
		case char == '"':
			context.state = continuationDoubleQuoted
			context.atWordStart = false
		case char == '#' && context.atWordStart:
			context.state = continuationComment
		default:
			context.atWordStart = false
		}
	}

	return context
}

func nextPhysicalLine(input string, start int) (lineEnd, newlineEnd int, ok bool) {
	for i := start; i < len(input); i++ {
		switch input[i] {
		case '\n':
			return i, i + 1, true
		case '\r':
			if i+1 < len(input) && input[i+1] == '\n' {
				return i, i + 2, true
			}
			return i, i + 1, true
		}
	}

	return len(input), len(input), false
}

func isHorizontalWhitespace(char byte) bool {
	return char == ' ' || char == '\t'
}

func isShellControlCharacter(char byte) bool {
	return strings.ContainsRune(";|&()<>", rune(char))
}

// split has been copied from go-shellquote and slightly modified so as to also
// return the remainder when the parsing failed because of an unterminated quote.
func split(input string, hl bool) (words []string, remainder string, err error) {
	var buf bytes.Buffer
	words = make([]string, 0)

	for len(input) > 0 {
		// skip any splitChars at the start
		c, l := utf8.DecodeRuneInString(input)
		if strings.ContainsRune(splitChars, c) {
			// Keep these characters in the result when higlighting the line.
			if hl {
				if len(words) == 0 {
					words = append(words, string(c))
				} else {
					words[len(words)-1] += string(c)
				}
			}

			input = input[l:]

			continue
		} else if c == escapeChar {
			// Look ahead for escaped newline so we can skip over it
			next := input[l:]
			if len(next) == 0 {
				if hl {
					remainder = string(escapeChar)
				}

				err = errUnterminatedEscape

				return words, remainder, err
			}

			c2, l2 := utf8.DecodeRuneInString(next)
			if c2 == '\n' {
				if hl {
					if len(words) == 0 {
						words = append(words, string(c)+string(c2))
					} else {
						words[len(words)-1] += string(c) + string(c2)
					}
				}

				input = next[l2:]

				continue
			}
		}

		var word string

		word, input, err = splitWord(input, &buf, hl)
		if err != nil {
			remainder = input
			return words, remainder, err
		}

		words = append(words, word)
	}

	return words, remainder, err
}

// splitWord has been modified to return the remainder of the input (the part that has not been
// added to the buffer) even when an error is returned.
func splitWord(input string, buf *bytes.Buffer, hl bool) (word string, remainder string, err error) {
	buf.Reset()

raw:
	{
		cur := input
		for len(cur) > 0 {
			c, l := utf8.DecodeRuneInString(cur)
			cur = cur[l:]
			if c == singleChar {
				buf.WriteString(input[0 : len(input)-len(cur)-l])
				input = cur
				goto single
			} else if c == doubleChar {
				buf.WriteString(input[0 : len(input)-len(cur)-l])
				input = cur
				goto double
			} else if c == escapeChar {
				buf.WriteString(input[0 : len(input)-len(cur)-l])
				if hl {
					buf.WriteRune(c)
				}
				input = cur
				goto escape
			} else if strings.ContainsRune(splitChars, c) {
				buf.WriteString(input[0 : len(input)-len(cur)-l])
				if hl {
					buf.WriteRune(c)
				}

				return buf.String(), cur, nil
			}
		}
		if len(input) > 0 {
			buf.WriteString(input)
			input = ""
		}
		goto done
	}

escape:
	{
		if len(input) == 0 {
			if hl {
				input = buf.String() + input
			}
			return "", input, errUnterminatedEscape
		}
		c, l := utf8.DecodeRuneInString(input)
		if c == '\n' {
			// a backslash-escaped newline is elided from the output entirely
		} else {
			buf.WriteString(input[:l])
		}
		input = input[l:]
	}

	goto raw

single:
	{
		i := strings.IndexRune(input, singleChar)
		if i == -1 {
			if hl {
				input = buf.String() + seqFgYellow + string(singleChar) + input
			}
			return "", input, errUnterminatedSingleQuote
		}
		// Catch up opening quote
		if hl {
			buf.WriteString(seqFgYellow)
			buf.WriteRune(singleChar)
		}

		buf.WriteString(input[0:i])
		input = input[i+1:]

		if hl {
			buf.WriteRune(singleChar)
			buf.WriteString(seqFgReset)
		}
		goto raw
	}

double:
	{
		cur := input
		for len(cur) > 0 {
			c, l := utf8.DecodeRuneInString(cur)
			cur = cur[l:]
			if c == doubleChar {
				// Catch up opening quote
				if hl {
					buf.WriteString(seqFgYellow)
					buf.WriteRune(c)
				}

				buf.WriteString(input[0 : len(input)-len(cur)-l])

				if hl {
					buf.WriteRune(c)
					buf.WriteString(seqFgReset)
				}
				input = cur
				goto raw
			} else if c == escapeChar && !hl {
				// bash only supports certain escapes in double-quoted strings
				c2, l2 := utf8.DecodeRuneInString(cur)
				cur = cur[l2:]
				if strings.ContainsRune(doubleEscapeChars, c2) {
					buf.WriteString(input[0 : len(input)-len(cur)-l-l2])
					if c2 == '\n' {
						// newline is special, skip the backslash entirely
					} else {
						buf.WriteRune(c2)
					}
					input = cur
				}
			}
		}

		if hl {
			input = buf.String() + seqFgYellow + string(doubleChar) + input
		}

		return "", input, errUnterminatedDoubleQuote
	}

done:
	return buf.String(), input, nil
}

func trimSpacesMatch(remain []string) (trimmed []string) {
	for _, word := range remain {
		trimmed = append(trimmed, strings.TrimSpace(word))
	}

	return
}

func (c *Console) lineEmpty(line string) bool {
	empty := true

	for _, r := range line {
		if !strings.ContainsRune(string(c.EmptyChars), r) {
			empty = false
			break
		}
	}

	return empty
}
