package core

import "testing"

func TestPrependedKeysPrecedeLaterRawInput(t *testing.T) {
	keys := new(Keys)
	keys.Feed(true, []rune("\x02\x02\x02")...)
	// Cursor queries may receive the next keyboard frame between dispatches.
	keys.buf = append(keys.buf, []byte("KEPT_")...)
	var got []byte
	for {
		key, empty := PopKey(keys)
		if empty {
			break
		}
		got = append(got, key)
	}
	if string(got) != "\x02\x02\x02KEPT_" {
		t.Fatalf("input dispatch order = %q", got)
	}
}
