package console

import "testing"

func TestPasteReferenceDefaultOnlyReferencesMultiline(t *testing.T) {
	c := New("test")
	c.EnablePasteReferences(PasteReferenceConfig{Enabled: true})

	if got := c.referencePastedText("demo_reqresp"); got != "demo_reqresp" {
		t.Fatalf("single-line paste = %q, want original text", got)
	}

	placeholder := c.referencePastedText("alpha\r\nbeta\ngamma")
	if placeholder != "[Pasted text #1 +3 lines]" {
		t.Fatalf("placeholder = %q", placeholder)
	}

	if got := c.ResolvePasteReferences("check " + placeholder); got != "check alpha\nbeta\ngamma" {
		t.Fatalf("resolved paste = %q", got)
	}
}

func TestPasteReferenceDisabledKeepsOriginalText(t *testing.T) {
	c := New("test")
	if got := c.referencePastedText("alpha\nbeta"); got != "alpha\nbeta" {
		t.Fatalf("disabled paste = %q, want original text", got)
	}
}

func TestPasteReferenceCustomTrigger(t *testing.T) {
	c := New("test")
	c.EnablePasteReferences(PasteReferenceConfig{
		Enabled: true,
		ShouldReference: func(text string) bool {
			return len(text) > 4
		},
	})

	if got := c.referencePastedText("demo"); got != "demo" {
		t.Fatalf("short paste = %q, want original text", got)
	}
	if got := c.referencePastedText("demo1"); got != "[Pasted text #1]" {
		t.Fatalf("custom referenced paste = %q", got)
	}
}
