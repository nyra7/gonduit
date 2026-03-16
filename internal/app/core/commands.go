package core

import (
	"app/command"
	"app/command/builtin"
	"context"

	tea "charm.land/bubbletea/v2"
)

func (app *gonduitApp) registerCommands() {
	app.cmdMgr.RegisterCommand(builtin.MakeExitCommand())
	app.cmdMgr.RegisterCommand(builtin.MakeClearCommand(app))
	app.cmdMgr.RegisterCommand(builtin.MakeIdentityCommand(app))
	app.cmdMgr.RegisterCommand(builtin.MakeConnectCommand(app))
	app.cmdMgr.RegisterCommand(builtin.MakeShellCommand(app))
	app.cmdMgr.RegisterCommand(builtin.MakeUploadCommand(app))
	app.cmdMgr.RegisterCommand(builtin.MakeDownloadCommand(app))
	app.cmdMgr.RegisterCommand(builtin.MakeListenCommand(app))
	app.cmdMgr.RegisterCommand(builtin.MakeSessionCommand(app))
	app.cmdMgr.Freeze()
}

// executeCommandAsync runs the command in a goroutine and sends the result back
func executeCommandAsync(cmdMgr *command.Manager, input string, ctx context.Context) tea.Cmd {
	return func() tea.Msg {
		output, err := cmdMgr.ExecWithContext(input, ctx)
		return CommandResultMsg{Output: output, Err: err}
	}
}
