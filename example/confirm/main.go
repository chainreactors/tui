package main

import "github.com/chainreactors/tui"

func main() {
	newConfirm := tui.NewConfirm("test confirm")
	newModel := tui.NewModel(newConfirm, nil, false, true)
	err := newModel.Run()
	if err != nil {
		return
	}
}
