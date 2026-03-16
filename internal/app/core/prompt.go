package core

import (
	"app/component"
	"context"

	tea "charm.land/bubbletea/v2"
)

// PromptInput asks the user for input during a running command
func (app *gonduitApp) PromptInput(ctx context.Context, promptText string, secure bool) (string, error) {
	return app.prompter.PromptInput(ctx, promptText, secure)
}

func (app *gonduitApp) handlePromptRequest(msg component.PromptRequestMsg) (tea.Model, tea.Cmd) {

	if msg.Req.Response == nil {
		return app, nil
	}

	app.resetCompletion()

	cmd := app.prompter.HandleRequest(msg.Req)
	return app, tea.Batch(cmd, tick())

}

func (app *gonduitApp) handlePromptEnter() (tea.Model, tea.Cmd) {
	if !app.prompter.IsActive() {
		return app, nil
	}

	app.prompter.Submit(app.prompter.Value())
	app.prompter.Restore()
	return app, nil
}

func (app *gonduitApp) handlePromptCancel() (tea.Model, tea.Cmd) {
	app.prompter.Cancel()
	app.prompter.Restore()
	if app.cmdMgr.IsRunning() && app.cancelCommand != nil {
		app.cancelCommand()
	}
	return app, nil
}

func (app *gonduitApp) NavigateHistoryUp() {
	app.prompter.SetValue(app.history.NavigateUp(app.prompter.Value()))
	app.prompter.CursorEnd()
}

func (app *gonduitApp) NavigateHistoryDown() {
	app.prompter.SetValue(app.history.NavigateDown(app.prompter.Value()))
	app.prompter.CursorEnd()
}

// ResetHistory resets the history navigation state
func (app *gonduitApp) ResetHistory() {
	app.history.Reset()
	app.scrollView.Reset()
}

func (app *gonduitApp) AddToHistory(value string) {
	app.history.Add(value)
}
