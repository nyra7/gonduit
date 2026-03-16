package builtin

import (
	"app"
	"app/command"
)

// clearUI command clears the screen
func clearUI(app app.App, _ *command.Context) (string, error) {

	// Clear the screen and reprint the welcome message
	app.Logger().Clear()
	app.Logger().Write(app.WelcomeMessage())
	app.Logger().Write("")

	return "", nil
}

// MakeClearCommand creates a new command handler for the clear command
func MakeClearCommand(app app.App) command.Handler {
	return command.NewHandler("clear", wrappedExecutor(app, clearUI), "Clears the screen", nil, nil)
}
