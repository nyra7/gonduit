package main

import (
	"app/core"
	"fmt"
	"shared/util"
)

func main() {

	// Run the TUI
	_, err := core.NewApp().Run()

	// Clear the terminal to remove any TUI remains
	util.ClearTerminal()

	// Print the error if any
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	}

}
