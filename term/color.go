package term

const (
	ANSIReset   = "\033[0m"
	ANSIBold    = "\033[1m"
	ANSIDim     = "\033[2m"
	ANSIRed     = "\033[31m"
	ANSIGreen   = "\033[32m"
	ANSIYellow  = "\033[33m"
	ANSIBlue    = "\033[34m"
	ANSIMagenta = "\033[35m"
	ANSICyan    = "\033[36m"
)

type Color struct {
	Enabled bool
}

func NewColor(enabled bool) Color {
	return Color{Enabled: enabled}
}

func (c Color) Code(code string) string {
	if !c.Enabled {
		return ""
	}
	return code
}

func (c Color) Wrap(s, code string) string {
	if !c.Enabled {
		return s
	}
	return code + s + ANSIReset
}

func (c Color) Bold(s string) string {
	if !c.Enabled {
		return s
	}
	return ANSIBold + s + ANSIReset
}

func (c Color) Dim(s string) string {
	if !c.Enabled {
		return s
	}
	return "\033[90m" + s + ANSIReset
}
