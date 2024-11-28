package tui

import (
	"fmt"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
	"os"
	"reflect"
	"strings"
)

var (
	output    = termenv.NewOutput(os.Stdout)
	profile   = termenv.ColorProfile()
	Normal    = lipgloss.NewStyle().String()
	Bold      = lipgloss.NewStyle().Bold(true).String()
	Underline = lipgloss.NewStyle().Underline(true).String()
	Blue      = profile.Color("#3398DA")
	Yellow    = profile.Color("#F1C40F")
	Purple    = profile.Color("#8D44AD")
	Green     = profile.Color("#2FCB71")
	Red       = profile.Color("#E74C3C")
	Gray      = profile.Color("#BDC3C7")
	DarkGray  = profile.Color("#808080")
	Cyan      = profile.Color("#1ABC9C")
	Orange    = profile.Color("#E67E22")
	Black     = profile.Color("#000000")
	Pink      = profile.Color("#EE82EE")
	SlateBlue = profile.Color("#6A5ACD")
	White     = profile.Color("#FFFFFF")
) // You can use ANSI color codes directly

var (
	Reset      = output.Reset
	Clear      = output.ClearLine
	UpN        = output.CursorPrevLine
	Down       = output.CursorNextLine
	ClearLines = output.ClearLines
	ClearAll   = output.ClearScreen
)

//var ClientPrompt = AdaptTermColor()

// adaptTermColor - Adapt term color
// TODO: Adapt term color by term(fork grumble ColorTableFg)
func AdaptTermColor(prompt string) string {
	var color string
	if termenv.HasDarkBackground() {
		color = fmt.Sprintf("\033[37m%s> \033[0m", prompt)
	} else {
		color = fmt.Sprintf("\033[30m%s> \033[0m", prompt)
	}
	return color
}

func AdaptSessionColor(prePrompt, sId string) string {
	var sessionPrompt string
	runes := []rune(sId)
	if termenv.HasDarkBackground() {
		sessionPrompt = fmt.Sprintf("\033[37m%s [%s]> \033[0m", prePrompt, string(runes))
	} else {
		sessionPrompt = fmt.Sprintf("\033[30m%s [%s]> \033[0m", prePrompt, string(runes))
	}
	return sessionPrompt
}

func NewSessionColor(prePrompt, sId string) string {
	var sessionPrompt string
	runes := []rune(sId)
	if termenv.HasDarkBackground() {
		sessionPrompt = fmt.Sprintf("%s [%s]> ", DefaultGroupStyle.Render(prePrompt), DefaultNameStyle.Render(string(runes)))
	} else {
		sessionPrompt = fmt.Sprintf("%s [%s]> ", DefaultGroupStyle.Render(prePrompt), DefaultNameStyle.Render(string(runes)))
	}
	return sessionPrompt
}

func RenderStruct(cfg interface{}, keyWidth int, indentLevel int, blacklist ...string) string {
	var builder strings.Builder
	v := reflect.ValueOf(cfg)
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	t := v.Type()

	blacklistMap := make(map[string]struct{})
	for _, name := range blacklist {
		blacklistMap[name] = struct{}{}
	}

	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)
		fieldType := t.Field(i)

		if !field.CanInterface() {
			continue
		}

		key := fieldType.Name
		if _, ok := blacklistMap[key]; ok {
			continue
		}

		coloredKey := termenv.String(fmt.Sprintf("%-*s", keyWidth, key)).Foreground(Blue).String()
		valueStr := fmt.Sprintf("%v", field.Interface())
		coloredValue := termenv.String(valueStr).Foreground(Green).String()

		switch field.Kind() {
		case reflect.Struct:
			builder.WriteString(fmt.Sprintf("%s%s:\n", strings.Repeat("\t", indentLevel), coloredKey))
			builder.WriteString(RenderStruct(field.Addr().Interface(), keyWidth, indentLevel+1))
		case reflect.Ptr:
			if field.IsNil() {
				field.Set(reflect.New(field.Type().Elem()))
			}
			builder.WriteString(fmt.Sprintf("%s%s:\n", strings.Repeat("\t", indentLevel), coloredKey))
			builder.WriteString(RenderStruct(field.Interface(), keyWidth, indentLevel+1))
		case reflect.Slice:
			builder.WriteString(fmt.Sprintf("%s%s:\n", strings.Repeat("\t", indentLevel), coloredKey))
			for j := 0; j < field.Len(); j++ {
				element := field.Index(j)
				elementStr := fmt.Sprintf("%v", element.Interface())
				coloredElement := termenv.String(elementStr).Foreground(Green).String()
				builder.WriteString(fmt.Sprintf("%s- %s\n", strings.Repeat("\t", indentLevel+1), coloredElement))
			}
		case reflect.Map:
			builder.WriteString(fmt.Sprintf("%s%s:\n", strings.Repeat("\t", indentLevel), coloredKey))
			for _, mapKey := range field.MapKeys() {
				mapValue := field.MapIndex(mapKey)
				coloredMapKey := termenv.String(fmt.Sprintf("%v", mapKey.Interface())).Foreground(Blue).String()
				coloredMapValue := termenv.String(fmt.Sprintf("%v", mapValue.Interface())).Foreground(Green).String()
				builder.WriteString(fmt.Sprintf("%s%s: %s\n", strings.Repeat("\t", indentLevel+1), coloredMapKey, coloredMapValue))
			}
		default:
			builder.WriteString(fmt.Sprintf("%s%s: %s\n", strings.Repeat("\t", indentLevel), coloredKey, coloredValue))
		}
	}

	return builder.String()
}
