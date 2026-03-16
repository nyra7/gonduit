package builtin

import (
	"app/command"
	"io"
)

// exit command exits the program
func exit(_ *command.Context) (string, error) {
	return "", io.EOF
}

// MakeExitCommand creates a command handler for the exit command
func MakeExitCommand() command.Handler {
	return command.NewHandler("exit", exit, "Exits the program", nil, nil)
}
