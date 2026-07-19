package console

import (
	"context"
	"reflect"
	"sync"
	"testing"

	"github.com/spf13/cobra"
)

func TestParseEscapeMode(t *testing.T) {
	tests := []struct {
		name  string
		mode  EscapeMode
		input string
		want  []string
	}{
		{
			name:  "shell consumes backslashes",
			mode:  EscapeShell,
			input: `run C:\Windows\Temp`,
			want:  []string{"run", `C:WindowsTemp`},
		},
		{
			name:  "literal preserves backslashes",
			mode:  EscapeLiteral,
			input: `run C:\Windows\Temp`,
			want:  []string{"run", `C:\Windows\Temp`},
		},
		{
			name:  "literal preserves trailing backslash",
			mode:  EscapeLiteral,
			input: `run C:\Windows\Temp\`,
			want:  []string{"run", `C:\Windows\Temp\`},
		},
		{
			name:  "literal still groups quotes",
			mode:  EscapeLiteral,
			input: `run "a b" C:\Temp`,
			want:  []string{"run", "a b", `C:\Temp`},
		},
		{
			name:  "literal still removes comments",
			mode:  EscapeLiteral,
			input: `run C:\Temp # note`,
			want:  []string{"run", `C:\Temp`},
		},
		{
			name:  "literal joins multiline arguments",
			mode:  EscapeLiteral,
			input: "run --host 127.0.0.1\n\t--port 1080",
			want:  []string{"run", "--host", "127.0.0.1", "--port", "1080"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			console := &Console{mutex: &sync.RWMutex{}}
			console.SetEscapeMode(tt.mode)

			got, err := console.parse(tt.input)
			if err != nil {
				t.Fatalf("parse(%q) error = %v", tt.input, err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("parse(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestAcceptMultilineEscapeMode(t *testing.T) {
	tests := []struct {
		name string
		mode EscapeMode
		line string
		want bool
	}{
		{"shell trailing backslash continues", EscapeShell, `run C:\Temp\`, false},
		{"literal path trailing backslash completes", EscapeLiteral, `run C:\Temp\`, true},
		{"literal standalone backslash completes", EscapeLiteral, `run arg \`, true},
		{"literal unterminated quote continues", EscapeLiteral, `run "unfinished`, false},
		{"literal complete multiline input accepts", EscapeLiteral, "run alpha\n\tbeta", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			console := &Console{mutex: &sync.RWMutex{}}
			console.SetEscapeMode(tt.mode)

			if got := console.acceptMultiline([]rune(tt.line)); got != tt.want {
				t.Fatalf("acceptMultiline(%q) = %v, want %v", tt.line, got, tt.want)
			}
		})
	}
}

func TestSplitArgsEscapeMode(t *testing.T) {
	tests := []struct {
		name  string
		mode  EscapeMode
		input string
		want  []string
	}{
		{"shell retains completion prefix", EscapeShell, `run C:\Windows`, []string{"run", `C:\Windows`}},
		{"literal preserves backslashes", EscapeLiteral, `run C:\Windows`, []string{"run", `C:\Windows`}},
		{"literal preserves trailing backslash", EscapeLiteral, `run C:\Windows\`, []string{"run", `C:\Windows\`}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := []rune(tt.input)
			got, _, _ := splitArgs(input, len(input), tt.mode)
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("splitArgs(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestRunCommandLineEscapeMode(t *testing.T) {
	tests := []struct {
		name string
		mode EscapeMode
		line string
		want []string
	}{
		{"shell consumes backslashes", EscapeShell, `run C:\Windows\Temp`, []string{`C:WindowsTemp`}},
		{"literal preserves backslashes", EscapeLiteral, `run C:\Windows\Temp`, []string{`C:\Windows\Temp`}},
		{"literal preserves trailing backslash", EscapeLiteral, `run C:\Windows\Temp\`, []string{`C:\Windows\Temp\`}},
		{"literal still groups quotes", EscapeLiteral, `run "a b" C:\x`, []string{"a b", `C:\x`}},
		{"literal joins multiline arguments", EscapeLiteral, "run alpha\n\tbeta", []string{"alpha", "beta"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			console := New("test")
			console.SetEscapeMode(tt.mode)
			menu := console.ActiveMenu()

			var got []string
			root := &cobra.Command{Use: "root"}
			root.AddCommand(&cobra.Command{
				Use: "run",
				Run: func(_ *cobra.Command, args []string) {
					got = args
				},
			})
			menu.Command = root

			if err := menu.RunCommandLine(context.Background(), tt.line); err != nil {
				t.Fatalf("RunCommandLine(%q) error = %v", tt.line, err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("RunCommandLine(%q) args = %q, want %q", tt.line, got, tt.want)
			}
		})
	}
}
