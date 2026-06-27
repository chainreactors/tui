package terminal

import "testing"

func TestStreamControlSizeAndResizeCallbacks(t *testing.T) {
	control := NewControl(true, 80, 24)
	if !control.IsTerminal() {
		t.Fatal("control should report terminal")
	}

	var gotCols, gotRows int
	unregister := control.OnResize(func(cols, rows int) {
		gotCols, gotRows = cols, rows
	})

	control.SetSize(120, 40)
	if gotCols != 120 || gotRows != 40 {
		t.Fatalf("resize callback got %dx%d, want 120x40", gotCols, gotRows)
	}

	cols, rows := control.Size()
	if cols != 120 || rows != 40 {
		t.Fatalf("control size = %dx%d, want 120x40", cols, rows)
	}

	unregister()
	control.SetSize(100, 30)
	if gotCols != 120 || gotRows != 40 {
		t.Fatalf("resize callback fired after unregister: %dx%d", gotCols, gotRows)
	}
}
