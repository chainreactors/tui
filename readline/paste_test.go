package readline

import "testing"

func TestNormalizePastedText(t *testing.T) {
	got := NormalizePastedText("alpha\r\nbeta\rgamma\n")
	want := "alpha\nbeta\ngamma"
	if got != want {
		t.Fatalf("NormalizePastedText() = %q, want %q", got, want)
	}
}

func TestPasteTransformerIsOptional(t *testing.T) {
	rl := NewShell()
	rl.InsertPastedText("demo_reqresp")
	if got := string(*rl.Line()); got != "demo_reqresp" {
		t.Fatalf("line = %q", got)
	}
}

func TestPasteTransformerCanRewritePaste(t *testing.T) {
	rl := NewShell()
	rl.SetPasteTransformer(func(text string) string {
		return "[" + text + "]"
	})
	rl.InsertPastedText("alpha\r\nbeta")
	if got := string(*rl.Line()); got != "[alpha\nbeta]" {
		t.Fatalf("line = %q", got)
	}
}
