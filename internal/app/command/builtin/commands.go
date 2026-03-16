package builtin

import (
	"app"
	"app/command"
)

// AppExecutor is an extended command.Executor that includes an app instance
type AppExecutor func(app app.App, ctx *command.Context) (string, error)

// wrappedExecutor wraps an app instance in a command executor
func wrappedExecutor(app app.App, handler AppExecutor) command.Executor {
	return func(ctx *command.Context) (string, error) { return handler(app, ctx) }
}
