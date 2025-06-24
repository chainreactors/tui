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
	err := newInput.Run()
	if err != nil {
		return
	}
}
