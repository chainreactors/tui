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

	Blue      = lipgloss.Color("#3398DA")
	Yellow    = lipgloss.Color("#F1C40F")
	Purple    = lipgloss.Color("#8D44AD")
	Green     = lipgloss.Color("#2FCB71")
	Red       = lipgloss.Color("#E74C3C")
	Gray      = lipgloss.Color("#BDC3C7")
	DarkGray  = lipgloss.Color("#808080")
	Cyan      = lipgloss.Color("#1ABC9C")
	Orange    = lipgloss.Color("#E67E22")
	Black     = lipgloss.Color("#000000")
	Pink      = lipgloss.Color("#EE82EE")
	SlateBlue = lipgloss.Color("#6A5ACD")
	White     = lipgloss.Color("#FFFFFF")

	BlueFg      = lipgloss.NewStyle().Foreground(Blue)
	YellowFg    = lipgloss.NewStyle().Foreground(Yellow)
	PurpleFg    = lipgloss.NewStyle().Foreground(Purple)
	GreenFg     = lipgloss.NewStyle().Foreground(Green)
	RedFg       = lipgloss.NewStyle().Foreground(Red)
	CyanFg      = lipgloss.NewStyle().Foreground(Cyan)
	OrangeFg    = lipgloss.NewStyle().Foreground(Orange)
	WhiteFg     = lipgloss.NewStyle().Foreground(White)
	BlackFg     = lipgloss.NewStyle().Foreground(Black)
	GrayFg      = lipgloss.NewStyle().Foreground(Gray)
	SlateBlueFg = lipgloss.NewStyle().Foreground(SlateBlue)
	DarkGrayFg  = lipgloss.NewStyle().Foreground(DarkGray)
	PinkFg      = lipgloss.NewStyle().Foreground(Pink)

	BlueBg      = lipgloss.NewStyle().Background(Blue)
	YellowBg    = lipgloss.NewStyle().Background(Yellow)
	PurpleBg    = lipgloss.NewStyle().Background(Purple)
	GreenBg     = lipgloss.NewStyle().Background(Green)
	RedBg       = lipgloss.NewStyle().Background(Red)
	CyanBg      = lipgloss.NewStyle().Background(Cyan)
	OrangeBg    = lipgloss.NewStyle().Background(Orange)
	WhiteBg     = lipgloss.NewStyle().Background(White)
	BlackBg     = lipgloss.NewStyle().Background(Black)
	GrayBg      = lipgloss.NewStyle().Background(Gray)
	SlateBlueBg = lipgloss.NewStyle().Background(SlateBlue)
	DarkGrayBg  = lipgloss.NewStyle().Background(DarkGray)
	PinkBg      = lipgloss.NewStyle().Background(Pink)
)
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

func RendStructDefault(stru interface{}, blacklist ...string) string {
	return RenderStruct(stru, 5, 1, blacklist...)
}

func RenderStruct(stru interface{}, keyWidth int, indentLevel int, blacklist ...string) string {
	var builder strings.Builder
	v := reflect.ValueOf(stru)
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

		coloredKey := BlueFg.Render(fmt.Sprintf("%-*s", keyWidth, key+":"))
		valueStr := fmt.Sprintf("%v", field.Interface())
		coloredValue := GreenFg.Render(valueStr)

		switch field.Kind() {
		case reflect.Struct:
			builder.WriteString(fmt.Sprintf("%s%s\n", strings.Repeat("  ", indentLevel), coloredKey))
			builder.WriteString(RenderStruct(field.Addr().Interface(), keyWidth, indentLevel+1))
		case reflect.Ptr:
			if field.IsNil() {
				field.Set(reflect.New(field.Type().Elem()))
			}
			builder.WriteString(fmt.Sprintf("%s%s\n", strings.Repeat("  ", indentLevel), coloredKey))
			builder.WriteString(RenderStruct(field.Interface(), keyWidth, indentLevel+1))
		case reflect.Slice:
			builder.WriteString(fmt.Sprintf("%s%s\n", strings.Repeat("  ", indentLevel), coloredKey))
			for j := 0; j < field.Len(); j++ {
				element := field.Index(j)
				if element.Kind() == reflect.Struct || element.Kind() == reflect.Ptr {
					builder.WriteString(fmt.Sprintf("%s- \n", strings.Repeat("  ", indentLevel+1)))
					builder.WriteString(RenderStruct(element.Interface(), keyWidth, indentLevel+2, blacklist...))
				} else {
					builder.WriteString(fmt.Sprintf("%s- %s\n", strings.Repeat("  ", indentLevel+1), GreenFg.Render(fmt.Sprintf("%v", element.Interface()))))
				}
			}
		case reflect.Map:
			builder.WriteString(fmt.Sprintf("%s%s\n", strings.Repeat("  ", indentLevel), coloredKey))
			for _, mapKey := range field.MapKeys() {
				mapValue := field.MapIndex(mapKey)
				coloredMapKey := BlueFg.Render(fmt.Sprintf("%v", mapKey.Interface()))
				if mapValue.Kind() == reflect.Struct || mapValue.Kind() == reflect.Ptr {
					builder.WriteString(fmt.Sprintf("%s%s \n", strings.Repeat("  ", indentLevel+1), coloredMapKey))
					builder.WriteString(RenderStruct(mapValue.Interface(), keyWidth, indentLevel+2, blacklist...))
				} else {
					builder.WriteString(fmt.Sprintf("%s%s %s\n", strings.Repeat("  ", indentLevel+1), coloredMapKey, GreenFg.Render(fmt.Sprintf("%v", mapValue.Interface()))))
				}
			}
		case reflect.Interface:
			if !field.IsNil() {
				builder.WriteString(fmt.Sprintf("%s%s\n", strings.Repeat("  ", indentLevel), coloredKey))
				builder.WriteString(RenderStruct(field.Interface(), keyWidth, indentLevel+1, blacklist...))
			} else {
				builder.WriteString(fmt.Sprintf("%s%s <nil>\n", strings.Repeat("  ", indentLevel), coloredKey))
			}
		default:
			builder.WriteString(fmt.Sprintf("%s%s %s\n", strings.Repeat("  ", indentLevel), coloredKey, coloredValue))
		}
	}

	return builder.String()
}
