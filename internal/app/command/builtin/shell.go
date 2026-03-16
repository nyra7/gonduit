package builtin

import (
	"app"
	"app/command"
	"app/core/grpc/session"
	"app/fmtx"
	"app/style"
	"fmt"
	"os"
	"shared/proto"
	"sort"
	"strings"
)

func MakeShellCommand(app app.App) command.Handler {

	args := []command.Argument{
		command.NewArgument("path", "The path of the shell to execute", command.ArgTypeString, false),
		command.NewArgument("args", "The arguments to pass to the shell", command.ArgTypeVariadic, false),
	}

	flags := []command.Argument{
		command.NewArgument("list", "List all active shells sessions", command.ArgTypeBool, false),
		command.NewArgument("valid", "List all valid shell executables on the remote server", command.ArgTypeBool, false),
		command.NewArgument("kill", "Kill the specified shell session", command.ArgTypeInt, false),
		command.NewArgument("interact", "Attach and interact with the specified shell", command.ArgTypeInt, false),
		command.NewArgument("dump", "Dump the contents of the specified shell session", command.ArgTypeInt, false),
		command.NewArgument("output", "The path to write the output of the shell session", command.ArgTypeString, false),
	}

	return command.NewHandler("shell", wrappedExecutor(app, shell), "Manage shells on the remote server", args, flags)

}

func shell(app app.App, ctx *command.Context) (string, error) {

	var session *session.Session
	var connErr error

	// Ensure we are connected to a server
	if session, connErr = app.SessionManager().ActiveSession(); connErr != nil {
		return "", connErr
	}

	// Get the arguments and flags from the context
	path := ctx.Argument(0)
	args := ctx.Variadic(1)
	list := ctx.BoolFlag("list")
	valid := ctx.BoolFlag("valid")
	shellId := ctx.IntFlag("interact")
	kill := ctx.IntFlag("kill")
	dump := ctx.IntFlag("dump")
	output := ctx.Flag("output")

	// If the valid flag is set, request the shell list from the server
	if valid {

		// Send a request to the server
		shells, err := session.ListShells(ctx.ControlContext())

		// Return the error if we failed to get the list
		if err != nil {
			return "", fmt.Errorf("failed to list valid shells: %w", err)
		}

		// Return the formatted shell list
		return formatShellList(shells), nil

	}

	// If the list flag is set, request the shell sessions from the server
	if list {

		// Send a request to the server
		sessions, err := session.ListSessions(ctx.ControlContext())

		// Return the error if we failed to get the list
		if err != nil {
			return "", fmt.Errorf("failed to list active shell sessions: %w", err)
		}

		return formatShellSessions(sessions.Sessions), nil

	}

	if ctx.IsFlagSet("dump") {

		if output == "" {
			return "", fmt.Errorf("--output must be specified for dumping history")
		}

		data, err := session.DumpShellHistory(ctx.ControlContext(), uint64(dump))

		if err != nil {
			return "", fmt.Errorf("failed to dump shell history: %w", err)
		}

		date := data.Session.CreatedAt.AsTime().Format("2006-01-02 15:04:05")

		var content strings.Builder
		content.WriteString(fmt.Sprintf("Shell %d running %s at %s\n", dump, data.Session.Path, date))
		content.WriteString(strings.Repeat("-", 80) + "\n")
		content.Write(data.Data)

		if err = os.WriteFile(output, []byte(content.String()), 0600); err != nil {
			return "", fmt.Errorf("failed to write dump to file: %w", err)
		}

		return fmtx.Successf("shell history dumped to %s", output), nil

	}

	// If the kill flag is set, kill the specified shell session
	if ctx.IsFlagSet("kill") {

		err := session.KillShell(ctx.ControlContext(), uint64(kill))

		if err != nil {
			return "", fmt.Errorf("failed to kill shell: %w", err)
		}

		return fmtx.Successf("killed shell %d", kill), nil

	}

	// If the interact flag is set, attach to the specified shell
	if ctx.IsFlagSet("interact") {
		return "", session.AttachShell(ctx.ControlContext(), uint64(shellId))
	}

	// Otherwise, create a new shell session
	id, err := session.CreateShell(ctx.ControlContext(), path, args)

	if err != nil {
		return "", err
	}

	return fmtx.Successf("shell started (id %d)", id), nil

}

func formatShellList(shells []string) string {
	if len(shells) == 0 {
		return style.InfoLabel.Render("Shells: ") + style.InfoStyle.Render("none reported by server")
	}
	header := style.InfoLabel.Render("Shells: ") + style.InfoStyle.Render(fmt.Sprintf("%d available", len(shells)))
	bulletStyle := style.Muted
	pathStyle := style.Text

	lines := make([]string, 0, len(shells)+1)
	lines = append(lines, header)
	for _, s := range shells {
		lines = append(lines, fmt.Sprintf("%s %s", bulletStyle.Render("-"), pathStyle.Render(s)))
	}
	return strings.Join(lines, "\n")
}

func formatShellSessions(shells []*proto.ShellSession) string {
	if len(shells) == 0 {
		return style.InfoLabel.Render("Shells: ") + style.InfoStyle.Render("none active")
	}

	bulletStyle := style.Muted
	pathStyle := style.Text

	sort.Slice(shells, func(i, j int) bool {
		return shells[i].Id < shells[j].Id
	})

	lines := make([]string, 0, len(shells)+1)
	lines = append(lines, style.InfoLabel.Render("Shells: ")+style.InfoStyle.Render(fmt.Sprintf("%d active", len(shells))))

	for _, sh := range shells {
		timeFmt := sh.CreatedAt.AsTime().Format("2006-01-02 15:04:05")
		if sh.IsAttached {
			lines = append(lines, fmt.Sprintf("%s Shell %d: %s at %s %s", bulletStyle.Render("-"), sh.Id, pathStyle.Render(sh.Path), timeFmt, style.SuccessStyle.Render("(attached)")))
		} else {
			lines = append(lines, fmt.Sprintf("%s Shell %d: %s at %s", bulletStyle.Render("-"), sh.Id, pathStyle.Render(sh.Path), timeFmt))
		}
	}
	return strings.Join(lines, "\n")
}
