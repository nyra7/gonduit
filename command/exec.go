package command

import (
	"fmt"
	"gonduit/util"
	"os/exec"
)

func execute(ctx *Context) error {

	args, _ := ctx.NextVariadic()

	if len(args) == 0 {
		return fmt.Errorf("no command specified")
	}

	command := args[0]
	args = args[1:]

	cmd := exec.Command(command, args...)

	out, err := cmd.CombinedOutput()

	util.WriteConn(ctx.Conn, string(out))

	return err

}

func MakeExecHandler() Handler {

	args := []Argument{
		NewArgument("command", "The commands and arguments to pass to the command line", ArgTypeVariadic, true),
	}

	return NewHandler("exec", execute, "Execute a command", args, []Argument{})

}
