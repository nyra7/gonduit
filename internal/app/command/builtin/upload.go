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

func MakeUploadCommand(app app.App) command.Handler {

	args := []command.Argument{
		command.NewArgument("local", "Local file path to upload", command.ArgTypeFile, true),
		command.NewArgument("remote", "Remote destination path", command.ArgTypeString, false, "."),
	}

	flags := []command.Argument{
		command.NewArgument("force", "Force overwrite if file exists", command.ArgTypeBool, false),
	}

	return command.NewHandler("upload", wrappedExecutor(app, upload), "Upload a file to the remote server", args, flags)

}

func upload(a app.App, ctx *command.Context) (string, error) {

	var sess *session.Session
	var connErr error

	// Ensure we are connected to a server
	if sess, connErr = a.SessionManager().ActiveSession(); connErr != nil {
		return "", connErr
	}

	localPath := ctx.Argument(0)
	remotePath := ctx.Argument(1)
	force := ctx.BoolFlag("force")

	result, err := sess.UploadSync(ctx.ControlContext(), localPath, remotePath, force)

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
		"uploaded %s to %s (%s)",
		filepath.Base(result.LocalPath),
		result.RemotePath,
		amount,
	), nil

}
