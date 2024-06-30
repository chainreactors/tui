package main

import (
	"fmt"
	"github.com/chainreactors/tui"
)

func main() {
	newInput := tui.NewInput("test input")
	newInput = newInput.SetHandler(func() {
		fmt.Println(newInput.TextInput.Value())
	})
	newModel := tui.NewModel(newInput, nil, false, true)
	err := newModel.Run()
	if err != nil {
		return
	}
}
