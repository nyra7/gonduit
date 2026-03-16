package builtin

import (
	"app"
	"app/command"
	"errors"
)

// connect is the root command used to connect to a gonduit server
func connect(app app.App, ctx *command.Context) (string, error) {

	// Fetch arguments and flag values
	host := ctx.Argument(0)
	port := ctx.IntFlag("port")

	if port <= 0 || port > 65535 {
		return "", errors.New("invalid port number")
	}

	return "", app.SessionManager().Connect(ctx.ControlContext(), host, port)

}

func MakeConnectCommand(app app.App) command.Handler {

	args := []command.Argument{
		command.NewArgument("host", "The host to connect to", command.ArgTypeString, true),
	}

	flags := []command.Argument{
		command.NewArgument("port", "The port to connect to", command.ArgTypeInt, false, 1337),
	}

	return command.NewHandler("connect", wrappedExecutor(app, connect), "Connects to a gonduit server", args, flags)

}
