package main

import "github.com/chainreactors/tui"

func main() {
	newConfirm := tui.NewConfirm("test confirm")
	err := newConfirm.Run()
	if err != nil {
		return
	}
}
