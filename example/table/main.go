package main

import (
	"fmt"
	"github.com/chainreactors/tui"
	"github.com/charmbracelet/bubbles/table"
	"os"
)

func main() {
	err := os.Setenv("RUNEWIDTH_EASTASIAN", "0")
	if err != nil {
		fmt.Println("Error setting RUNEWIDTH_EASTASIAN variable:", err)
		return
	}
	err = os.Setenv("LC_CTYPE", "en_US.UTF-8")
	if err != nil {
		fmt.Println("Error setting LC_CTYPE variable:", err)
		return
	}
	newTable := tui.NewTable([]table.Column{
		{Title: "Name", Width: 20},
		{Title: "IsDir", Width: 5},
		{Title: "Size", Width: 7},
		{Title: "ModTime", Width: 10},
		{Title: "Link", Width: 15},
	}, false)
	rows := []table.Row{
		{
			"h3zh1",
			"true",
			"17263",
			"2024.1.18",
			"",
		},
		{
			"h4zh1",
			"true",
			"17263",
			"2024.1.18",
			"",
		},
		{
			"h3zh2",
			"true",
			"17263",
			"2024.1.18",
			"",
		},
	}
	newTable.Rows = rows
	newTable.SetRows()
	tableModel := tui.NewModel(newTable, newTable.ConsoleHandler)
	tableModel.Run()
}
