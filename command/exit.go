package command

import (
	"io"
)

func exit(_ *Context) error {
	return io.EOF
}

func MakeExitCommand() Handler {
	return NewHandler("exit", exit, "Exit gonduit", []Argument{}, []Argument{})
}
