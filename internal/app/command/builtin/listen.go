package builtin

import (
	"app"
	"app/command"
	"app/fmtx"
	"errors"
)

func listen(app app.App, ctx *command.Context) (string, error) {

	// Fetch arguments and flag values
	addr := ctx.Argument(0)
	port := ctx.IntFlag("port")
	insecure := ctx.BoolFlag("insecure")
	accept := ctx.Flag("accept")

	if port <= 0 || port > 65535 {
		return "", errors.New("invalid port number")
	}

	if ctx.IsFlagSet("stop") {
		return fmtx.Successf("stopped listener"), app.SessionManager().StopListening()
	}

	if accept == "" {
		if insecure {
			app.Logger().Warnf(
				"listening in insecure mode will automatically accept all incoming reverse connections!" +
					" consider using '--accept' to restrict allowed IPs",
			)
		} else if app.SessionManager().IsSelfSignedIdentity() || !app.SessionManager().HasIdentity() {
			app.Logger().Warnf(
				"listening with a self-signed certificate will automatically accept all incoming reverse connections!" +
					" consider using '--accept' to restrict allowed IPs",
			)
		}
	}

	return fmtx.Successf("listening on %s:%d", addr, port), app.SessionManager().Listen(addr, port, accept, insecure)

}

func MakeListenCommand(app app.App) command.Handler {

	args := []command.Argument{
		command.NewArgument("addr", "The address to bind the listener to", command.ArgTypeString, false, "0.0.0.0"),
	}

	flags := []command.Argument{
		command.NewArgument("port", "The port to bind the listener to", command.ArgTypeInt, false, 1337),
		command.NewArgument("accept", "Allowed IP addresses (comma-separated IPs and/or CIDRs)", command.ArgTypeString, false),
		command.NewArgument("insecure", "Accepts all incoming server connections without validation", command.ArgTypeBool, false),
		command.NewArgument("stop", "Stops the current listener", command.ArgTypeBool, false),
	}

	return command.NewHandler("listen", wrappedExecutor(app, listen), "Manages the listener for reverse connections", args, flags)

}
