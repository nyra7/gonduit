package core

import (
	"app/component"
	"app/fmtx"
	"app/style"
	"context"
	"io"
	"shared/util"
	"strings"
	"time"

	"charm.land/bubbles/v2/progress"
	"charm.land/bubbles/v2/spinner"
	tea "charm.land/bubbletea/v2"
)

// Update handles Bubble Tea messages
func (app *gonduitApp) Update(msg tea.Msg) (tea.Model, tea.Cmd) {

	// Dispatch different messages
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		return app.handleWindowSize(msg)

	case tea.MouseWheelMsg:
		return app.handleMouseWheel(msg)

	case tea.KeyPressMsg:
		return app.handleKey(msg)

	case progress.FrameMsg:
		progressModel, cmd := app.progress.Update(msg)
		app.progress = progressModel
		return app, cmd

	case spinner.TickMsg:
		var cmd tea.Cmd
		app.spinner, cmd = app.spinner.Update(msg)
		return app, cmd

	case component.PromptRequestMsg:
		return app.handlePromptRequest(msg)

	case TickMsg:
		return app.handleTick(msg)

	case CommandResultMsg:
		return app.handleCommandResult(msg)

	case CommandCancelMsg:
		return app.handleCommandCancel()

	case ExitMsg:
		app.Close()
		return app, tea.Batch(tea.Quit)
	}

	// Update the text input
	return app, app.prompter.Update(msg)

}

func (app *gonduitApp) handleWindowSize(msg tea.WindowSizeMsg) (tea.Model, tea.Cmd) {

	app.cmdMgr.SetDimensions(msg.Width, msg.Height)
	app.scrollView.SetDimensions(msg.Width, msg.Height)
	app.sessionMgr.WindowResized(util.TerminalSize{Rows: int32(msg.Height), Columns: int32(msg.Width)})

	inputWidth := msg.Width - 2
	if inputWidth < 0 {
		inputWidth = 0
	}
	app.prompter.SetWidth(inputWidth)

	return app, tea.ClearScreen

}

func (app *gonduitApp) handleTick(msg TickMsg) (tea.Model, tea.Cmd) {

	if app.cmdMgr.IsRunning() || app.prompter.IsActive() {
		now := time.Time(msg)
		dt := now.Sub(app.lastFrame).Seconds()
		app.lastFrame = now
		app.loader.Update(dt, app.scrollView.Width())
		return app, tick()
	}

	return app, nil

}

func (app *gonduitApp) handleCommandResult(msg CommandResultMsg) (tea.Model, tea.Cmd) {

	app.transferActive = false
	if app.cancelCommand != nil {
		app.cancelCommand()
		app.cancelCommand = nil
	}
	app.prompter.Restore()

	if msg.Err != nil {
		return app.handleCommandError(msg.Err)
	}

	return app.handleCommandOutput(msg.Output)
}

func (app *gonduitApp) handleFileTransferProgress(filename string, current, total uint64) {
	app.transferActive = true
	app.transferBytes = current
	app.transferTotal = total
	app.transferName = filename

	// Update progress bar
	percent := float64(current) / float64(total)
	app.program.Send(app.progress.SetPercent(percent))
}

func (app *gonduitApp) handleCommandError(err error) (tea.Model, tea.Cmd) {

	// Exit command
	if err == io.EOF {
		return app, func() tea.Msg { return ExitMsg{} }
	}

	app.Logger().Write(fmtx.Error(util.ParseGrpcError(err).Error()))
	return app, nil

}

func (app *gonduitApp) handleCommandOutput(output string) (tea.Model, tea.Cmd) {
	if output != "" {
		for _, line := range strings.Split(output, "\n") {
			app.Logger().Write(line)
		}
	}
	// Reset scroll when new output arrives
	app.scrollView.Reset()
	return app, nil
}

func (app *gonduitApp) handleCommandCancel() (tea.Model, tea.Cmd) {
	if app.cmdMgr.IsRunning() && app.cancelCommand != nil {
		app.cancelCommand()
		app.prompter.Restore()
		app.progress.SetPercent(0)
	}
	return app, nil
}

func (app *gonduitApp) handleKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {

	if app.prompter.IsActive() {

		switch msg.String() {
		case "enter":
			return app.handlePromptEnter()
		case "ctrl+c":
			return app.handlePromptCancel()
		}

		return app, app.prompter.Update(msg)

	}

	// If command is running, only allow Ctrl+C to cancel
	if app.cmdMgr.IsRunning() {
		if msg.String() == "ctrl+c" {
			return app, func() tea.Msg { return CommandCancelMsg{} }
		}
		// Ignore other keys while command is running
		return app, nil
	}

	// Intercept left/right to navigate completions when active
	if app.completer.IsActive() && app.completer.NumItems() > 1 {
		switch msg.String() {
		case "left":
			app.prompter.SetValue(app.completer.Complete(app.prompter.Value(), component.CycleBackward))
			app.prompter.CursorEnd()
			return app, nil
		case "right":
			app.prompter.SetValue(app.completer.Complete(app.prompter.Value(), component.CycleForward))
			app.prompter.CursorEnd()
			return app, nil
		}
	}

	switch msg.String() {
	case "enter":
		return app.handleEnter()
	case "up":
		app.resetCompletion()
		app.NavigateHistoryUp()
		return app, nil
	case "down":
		app.resetCompletion()
		app.NavigateHistoryDown()
		return app, nil

	case "tab":
		app.handleTabCompletion()
		return app, nil

	case "ctrl+c":
		if app.prompter.Value() == "" {
			app.Logger().Warn("Interrupt. Type 'exit' to quit")
		}
		app.prompter.SetValue("")
		app.resetCompletion()
		return app, nil

	default:
		// Any other key resets completion mode
		app.resetCompletion()

		if msg.Code == tea.KeyBackspace || msg.Code == tea.KeyDelete {
			app.scrollView.Reset()
		}
	}

	return app, app.prompter.Update(msg)

}

func (app *gonduitApp) handleMouseWheel(msg tea.MouseWheelMsg) (tea.Model, tea.Cmd) {

	switch msg.Button {
	case tea.MouseWheelUp:
		app.scrollUp()
	case tea.MouseWheelDown:
		app.scrollDown()
	default:
	}

	return app, nil
}

func (app *gonduitApp) handleEnter() (tea.Model, tea.Cmd) {

	input := app.prompter.Value()

	if input == "" {
		return app, nil
	}

	// Check if we're in completion mode - if so, accept current selection
	if app.completer.IsActive() {

		newInput, _ := app.completer.AcceptAndContinue(input)
		app.prompter.SetValue(newInput)
		app.prompter.CursorEnd()

		return app, nil

	}

	// Reset completion state
	app.resetCompletion()

	// Add to history
	app.AddToHistory(input)
	app.ResetHistory()

	// Add styled input to scrollback
	promptStyled := style.Primary.Render(">")
	inputStyled := style.Text.Render(" " + input)
	app.Logger().Write(promptStyled + inputStyled)

	app.prompter.Clear()

	// Create cancellable context for command execution
	ctx, cancel := context.WithCancel(context.Background())
	app.cancelCommand = cancel

	app.lastFrame = time.Now()

	return app, tea.Batch(
		executeCommandAsync(app.cmdMgr, input, ctx),
		tick(),
	)
}

// calculateAvailableLines computes the number of lines available for the scrollback output (height - ui elements)
func (app *gonduitApp) calculateAvailableLines() int {

	// Number of lines in the header and footer (individually)
	const headerFooterLines = 2

	// Start with the full height of the terminal
	available := app.scrollView.Height()

	// Count 4 lines for the header (title + sep) and footer (sep + prompt)
	available -= 2 * headerFooterLines

	// Count 1 line for the completion hints (if visible)
	if app.completer.NumItems() > 1 {
		available -= 1
	}

	// Add a blank line if a command is running or a prompt is active (looks better)
	if app.cmdMgr.IsRunning() || app.prompter.IsActive() {
		available -= 1
	}

	return max(0, available)

}

func (app *gonduitApp) resetCompletion() {
	app.completer.Reset()
}

func (app *gonduitApp) scrollUp() {
	app.scrollView.ScrollUp(app.calculateAvailableLines())
}

func (app *gonduitApp) scrollDown() {
	app.scrollView.ScrollDown()
}
