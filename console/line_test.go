package console

import (
	"reflect"
	"strings"
	"testing"
)

func TestScanLineContinuationsHorizontalWhitespaceMatrix(t *testing.T) {
	t.Parallel()

	before := []string{" ", "\t", " \t", "\t ", "  \t "}
	after := []string{"", " ", "\t", " \t", "\t "}

	for _, leading := range before {
		for _, trailing := range after {
			input := "cmd arg" + leading + "\\" + trailing
			name := "before=" + strings.ReplaceAll(leading, "\t", "<tab>") +
				"/after=" + strings.ReplaceAll(trailing, "\t", "<tab>")

			t.Run(name, func(t *testing.T) {
				normalized, pending := scanLineContinuations(input)
				if !pending {
					t.Fatalf("scanLineContinuations(%q) pending = false, want true", input)
				}
				if normalized != input {
					t.Fatalf("scanLineContinuations(%q) normalized = %q, want unchanged", input, normalized)
				}

				completed, stillPending := scanLineContinuations(input + "\nnext")
				wantCompleted := "cmd arg" + leading + "next"
				if completed != wantCompleted {
					t.Fatalf("completed continuation normalized = %q, want %q", completed, wantCompleted)
				}
				if stillPending {
					t.Fatal("completed continuation remained pending")
				}
			})
		}
	}
}

func TestScanLineContinuationsClassification(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{name: "space before marker", input: "cmd arg \\", want: true},
		{name: "trailing horizontal whitespace", input: "cmd arg \\ \t", want: true},
		{name: "closed single quote before marker", input: "cmd 'arg' \\", want: true},
		{name: "closed double quote before marker", input: "cmd \"arg\" \\", want: true},
		{name: "escaped preceding space follows raw suffix grammar", input: "cmd foo\\ \\", want: true},
		{name: "no leading horizontal whitespace", input: "cmd\\", want: false},
		{name: "standalone backslash", input: "\\", want: false},
		{name: "two backslashes", input: "cmd \\\\", want: false},
		{name: "two backslashes and trailing whitespace", input: "cmd \\\\ \t", want: false},
		{name: "three backslashes", input: "cmd \\\\\\", want: false},
		{name: "four backslashes", input: "cmd \\\\\\\\", want: false},
		{name: "content after backslash", input: "cmd \\ tail", want: false},
		{name: "inside single quote", input: "cmd 'literal \\ \t", want: false},
		{name: "inside double quote", input: "cmd \"literal \\ \t", want: false},
		{name: "escaped quote stays inside double quote", input: "cmd \"a\\\"b \\ \t", want: false},
		{name: "comment", input: "cmd # comment \\ \t", want: false},
		{name: "windows path", input: `cmd C:\Temp\`, want: false},
		{name: "windows path trailing whitespace", input: "cmd C:\\Temp\\ \t", want: false},
		{name: "UNC path", input: `cmd \\server\share\`, want: false},
		{name: "registry path", input: `cmd HKLM\Software\Test\`, want: false},
		{name: "named pipe", input: `cmd \\.\pipe\agent\`, want: false},
		{name: "non breaking space", input: "cmd\u00a0\\", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, got := scanLineContinuations(tt.input)
			if got != tt.want {
				t.Fatalf("scanLineContinuations(%q) pending = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestScanLineContinuationsNormalizesCompletedMarkers(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		input       string
		want        string
		wantPending bool
	}{
		{name: "LF", input: "cmd a \\\nnext", want: "cmd a next"},
		{name: "trailing spaces", input: "cmd a \\   \nnext", want: "cmd a next"},
		{name: "mixed tabs and spaces", input: "cmd a \t\\ \t\n\t  next", want: "cmd a \t\t  next"},
		{name: "CRLF", input: "cmd a \\\r\nnext", want: "cmd a next"},
		{name: "lone CR", input: "cmd a \\\rnext", want: "cmd a next"},
		{
			name:  "multiple mixed line endings",
			input: "artifact publish RIGID_ \\ \r\n\t--website web\t\\\t\n  --path /x",
			want:  "artifact publish RIGID_ \t--website web\t  --path /x",
		},
		{name: "pending EOF marker", input: "cmd a \\ \t", want: "cmd a \\ \t", wantPending: true},
		{name: "ordinary newline", input: "cmd a\nnext", want: "cmd a\nnext"},
		{name: "single quoted marker", input: "cmd 'literal \\ \t\nnext'", want: "cmd 'literal \\ \t\nnext'"},
		{name: "double quoted marker", input: "cmd \"literal \\ \t\nnext\"", want: "cmd \"literal \\ \t\nnext\""},
		{name: "comment marker", input: "cmd # comment \\ \t\nnext", want: "cmd # comment \\ \t\nnext"},
		{
			name:  "word state crosses completed continuation",
			input: "cmd foo\\ \\\n#literal \\\nnext",
			want:  "cmd foo\\ #literal next",
		},
		{name: "two backslashes", input: "cmd \\\\ \t\nnext", want: "cmd \\\\ \t\nnext"},
		{name: "three backslashes", input: "cmd \\\\\\ \t\nnext", want: "cmd \\\\\\ \t\nnext"},
		{name: "windows paths", input: "cmd C:\\Temp\\\ncmd \\\\.\\pipe\\agent\\", want: "cmd C:\\Temp\\\ncmd \\\\.\\pipe\\agent\\"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, pending := scanLineContinuations(tt.input)
			if got != tt.want {
				t.Fatalf("scanLineContinuations(%q) normalized = %q, want %q", tt.input, got, tt.want)
			}
			if pending != tt.wantPending {
				t.Fatalf("scanLineContinuations(%q) pending = %v, want %v", tt.input, pending, tt.wantPending)
			}
		})
	}
}

func TestConsoleAcceptMultilineUsesExplicitMarkers(t *testing.T) {
	t.Parallel()

	console := &Console{}
	tests := []struct {
		name       string
		input      string
		wantAccept bool
	}{
		{name: "explicit marker", input: "cmd arg \\ \t", wantAccept: false},
		{name: "tab marker", input: "cmd arg\t\\\t", wantAccept: false},
		{name: "windows path", input: `cmd C:\Temp\`, wantAccept: true},
		{name: "UNC path", input: `cmd \\server\share\`, wantAccept: true},
		{name: "registry path", input: `cmd HKLM\Software\Test\`, wantAccept: true},
		{name: "named pipe", input: `cmd \\.\pipe\agent\`, wantAccept: true},
		{name: "two backslashes", input: `cmd \\`, wantAccept: true},
		{name: "three backslashes", input: `cmd \\\`, wantAccept: true},
		{name: "unterminated single quote", input: "cmd 'unterminated", wantAccept: false},
		{name: "unterminated double quote", input: `cmd "unterminated`, wantAccept: false},
		{name: "completed continuation", input: "cmd a \\\nnext", wantAccept: true},
		{name: "second pending continuation", input: "cmd a \\\nnext \\  ", wantAccept: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := console.acceptMultiline([]rune(tt.input)); got != tt.wantAccept {
				t.Fatalf("acceptMultiline(%q) = %v, want %v", tt.input, got, tt.wantAccept)
			}
		})
	}
}

func TestParseRemovesExplicitLineContinuations(t *testing.T) {
	t.Parallel()

	console := &Console{}
	input := "artifact publish RIGID_ \\\n\t--website web \\\n\t--path /x"
	want := []string{
		"artifact", "publish", "RIGID_",
		"--website", "web",
		"--path", "/x",
	}

	got, err := console.parse(input)
	if err != nil {
		t.Fatalf("parse() error = %v", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("parse() = %#v, want %#v", got, want)
	}
}

func TestParseRemovesContinuationsWithTrailingWhitespaceAndCRLF(t *testing.T) {
	t.Parallel()

	console := &Console{}
	want := []string{
		"artifact", "publish", "RIGID_",
		"--website", "web",
		"--path", "/x",
	}
	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "trailing horizontal whitespace",
			input: "artifact publish RIGID_ \\ \t\n\t--website web\t\\\t \n\t--path /x",
		},
		{
			name:  "mixed CRLF and LF",
			input: "artifact publish RIGID_ \\ \r\n\t--website web\t\\\t\n\t--path /x",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := console.parse(tt.input)
			if err != nil {
				t.Fatalf("parse() error = %v", err)
			}
			if !reflect.DeepEqual(got, want) {
				t.Fatalf("parse() = %#v, want %#v", got, want)
			}
		})
	}
}

func TestParsePreservesWindowsBackslashes(t *testing.T) {
	t.Parallel()

	console := &Console{}
	tests := []struct {
		name  string
		input string
		want  []string
	}{
		{name: "windows path", input: `cmd C:\Windows\System32`, want: []string{"cmd", `C:\Windows\System32`}},
		{name: "trailing windows backslash", input: `cmd C:\Temp\`, want: []string{"cmd", `C:\Temp\`}},
		{name: "quoted windows path", input: `cmd "C:\Program Files\App\a.exe"`, want: []string{"cmd", `C:\Program Files\App\a.exe`}},
		{name: "registry path", input: `cmd HKLM\Software\Vendor\Key`, want: []string{"cmd", `HKLM\Software\Vendor\Key`}},
		{name: "UNC path", input: `cmd \\server\share\file`, want: []string{"cmd", `\\server\share\file`}},
		{name: "named pipe", input: `cmd \\.\pipe\malice`, want: []string{"cmd", `\\.\pipe\malice`}},
		{name: "windows path before newline", input: "cmd C:\\Temp\\\nnext", want: []string{"cmd", `C:\Temp\`, "next"}},
		{name: "windows path before CRLF", input: "cmd C:\\Temp\\\r\nnext", want: []string{"cmd", `C:\Temp\`, "next"}},
		{name: "UNC path before newline", input: "cmd \\\\server\\share\\\nnext", want: []string{"cmd", `\\server\share\`, "next"}},
		{name: "UNC path before CRLF", input: "cmd \\\\server\\share\\\r\nnext", want: []string{"cmd", `\\server\share\`, "next"}},
		{name: "named pipe before newline", input: "cmd \\\\.\\pipe\\malice\\\nnext", want: []string{"cmd", `\\.\pipe\malice\`, "next"}},
		{name: "named pipe before CRLF", input: "cmd \\\\.\\pipe\\malice\\\r\nnext", want: []string{"cmd", `\\.\pipe\malice\`, "next"}},
		{name: "quoted backslash before newline", input: "cmd \"literal \\\nnext\"", want: []string{"cmd", "literal \\\nnext"}},
		{name: "path with trailing whitespace before newline", input: `cmd C:\Temp\   ` + "\nnext", want: []string{"cmd", `C:\Temp\`, "next"}},
		{name: "two backslashes before newline", input: `cmd \\` + "\nnext", want: []string{"cmd", `\\`, "next"}},
		{name: "three backslashes before newline", input: `cmd \\\` + "\nnext", want: []string{"cmd", `\\\`, "next"}},
		{name: "unicode windows path", input: `cmd C:\临时\目录`, want: []string{"cmd", `C:\临时\目录`}},
		{name: "placeholder collision", input: "cmd \ue000 " + `C:\Temp\` + "\nnext", want: []string{"cmd", "\ue000", `C:\Temp\`, "next"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := console.parse(tt.input)
			if err != nil {
				t.Fatalf("parse() error = %v", err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("parse() = %#v, want %#v", got, tt.want)
			}
		})
	}
}

func TestParseContinuesAfterHashInEscapedWord(t *testing.T) {
	t.Parallel()

	console := &Console{}
	got, err := console.parse("cmd foo\\ \\\n#literal \\\nnext")
	if err != nil {
		t.Fatalf("parse() error = %v", err)
	}

	want := []string{"cmd", `foo\`, "#literal", "next"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("parse() = %#v, want %#v", got, want)
	}
}

func TestParseDoesNotContinueComments(t *testing.T) {
	t.Parallel()

	console := &Console{}
	input := `cmd # comment \` + "\nnext"
	got, err := console.parse(input)
	if err != nil {
		t.Fatalf("parse() error = %v", err)
	}

	want := []string{"cmd", "next"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("parse() = %#v, want %#v", got, want)
	}
}
