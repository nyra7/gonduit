package builtin

import (
	"app"
	"app/command"
	"app/core/grpc/session"
	"app/fmtx"
	"fmt"
	"path/filepath"
	"shared/util"
)

func MakeDownloadCommand(app app.App) command.Handler {

	args := []command.Argument{
		command.NewArgument("remote", "Remote file path to download", command.ArgTypeString, true),
		command.NewArgument("local", "Local destination path", command.ArgTypeDirectory, false, "."),
	}

	flags := []command.Argument{
		command.NewArgument("force", "Force overwrite if file exists", command.ArgTypeBool, false),
	}

	return command.NewHandler("download", wrappedExecutor(app, download), "Download a file from the remote server", args, flags)

}

func download(a app.App, ctx *command.Context) (string, error) {

	var sess *session.Session
	var connErr error

	// Ensure we are connected to a server
	if sess, connErr = a.SessionManager().ActiveSession(); connErr != nil {
		return "", connErr
	}

	remotePath := ctx.Argument(0)
	localPath := ctx.Argument(1)
	force := ctx.BoolFlag("force")

	result, err := sess.DownloadSync(ctx.ControlContext(), localPath, remotePath, force)

	if err != nil {
		return "", err
	}

	b, unit := util.HumanReadableBytes(result.Size)

	var amount string
	if unit == "bytes" {
		amount = fmt.Sprintf("%d %s", result.Size, unit)
	} else {
		amount = fmt.Sprintf("%.2f %s", b, unit)
	}

	return fmtx.Successf(
		"downloaded %s to %s (%s)",
		filepath.Base(result.RemotePath),
		result.LocalPath,
		amount,
	), nil

}
