package core

import (
	"app/component"
	"app/style"
	"fmt"
	"path/filepath"
	"shared/pkg"
	"shared/util"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

const completionEllipsis = "…"

// View renders the terminal UI
func (app *gonduitApp) View() tea.View {

	app.view.MouseMode = tea.MouseModeAllMotion

	app.view.KeyboardEnhancements = tea.KeyboardEnhancements{ReportEventTypes: false}
	if app.scrollView.Width() == 0 || app.scrollView.Height() == 0 {
		app.view.SetContent(app.prompter.View())
		return app.view
	}

	var b strings.Builder
	b.Grow(app.scrollView.Width() * app.scrollView.Height() * 2)

	// Header with divider
	app.renderHeader(&b)

	// Middle scrollback
	app.renderScrollback(&b)

	// Completion hint (if multiple matches)
	app.renderCompletionHint(&b)

	// Bottom section (loader, status, or prompt)
	app.renderBottomSection(&b)

	app.view.SetContent(b.String())

	return app.view

}

func (app *gonduitApp) renderHeader(b *strings.Builder) {

	// Header with app name
	header := style.Header.Render(fmt.Sprintf(" GONDUIT (version %s)", pkg.Version))

	var statusStr string

	num := app.sessionMgr.NumSessions()
	sess, err := app.sessionMgr.ActiveSession()

	if err != nil {

		word := "sessions"
		if num == 1 {
			word = "session"
		}

		if num == 0 {
			statusStr = style.Muted.Render("⊘ No Sessions ")
		} else {
			statusStr = style.Primary.Render(fmt.Sprintf("⧉  %d background %s ", num, word))
		}

	} else {

		info := sess.HostInfo()
		statusText := fmt.Sprintf("● Session %d | %s@%s (TLS) ", sess.ID(), info.Username, info.Hostname)
		statusStr = style.SuccessText.Render(statusText)

	}

	// Strip ANSI codes to get true display length
	headerLen := lipgloss.Width(header)
	statusLen := lipgloss.Width(statusStr)

	// Calculate padding to align status to the right edge
	padding := app.scrollView.Width() - headerLen - statusLen
	if padding < 1 {
		padding = 1
	}

	headerLine := header + strings.Repeat(" ", padding) + statusStr
	b.WriteString(headerLine)
	b.WriteRune('\n')

	// Divider line
	if app.cmdMgr.IsRunning() || app.prompter.IsActive() {
		b.WriteString(app.loader.Render(app.scrollView.Width()))
	} else {
		divider := style.Divider.Render(strings.Repeat("─", app.scrollView.Width()))
		b.WriteString(divider)
	}

	b.WriteRune('\n')

}

func (app *gonduitApp) renderScrollback(b *strings.Builder) {

	// Compute the number of available lines
	available := app.calculateAvailableLines()

	// Get the visible lines based on the available space and scroll state
	visible := app.scrollView.VisibleLines(available)

	// Render the lines
	for _, line := range visible {
		b.WriteString(line)
		b.WriteRune('\n')
	}

	// Fill remaining space with empty lines
	for i := len(visible); i < available; i++ {
		b.WriteRune('\n')
	}

}

func (app *gonduitApp) handleTabCompletion() {

	input := app.prompter.Value()

	// If not yet in completion mode, start it
	if !app.completer.IsActive() {

		newInput := app.completer.Complete(input, component.CycleForward)
		app.prompter.SetValue(newInput)
		app.prompter.CursorEnd()

		// Auto-drill into single directory completions
		if app.completer.NumItems() == 1 {
			if names := app.completer.DisplayNames(); len(names) > 0 && strings.HasSuffix(names[0], string(filepath.Separator)) {
				if accepted, isDir := app.completer.AcceptAndContinue(newInput); isDir {
					app.prompter.SetValue(accepted)
					app.prompter.CursorEnd()
				}
			}
		}

	} else {
		app.prompter.SetValue(app.completer.Complete(input, component.CycleForward))
		app.prompter.CursorEnd()
	}

}

func (app *gonduitApp) renderCompletionHint(b *strings.Builder) {
	total := app.completer.NumItems()
	if total <= 1 {
		return
	}

	completions := app.completer.DisplayNames()
	selectedIdx := app.completer.Index()

	// Calculate how many pills fit on screen
	termWidth := app.scrollView.Width()
	visibleItems, pageStart := calcVisiblePage(completions, selectedIdx, termWidth)

	hasMore := pageStart+visibleItems < total
	hasPrev := pageStart > 0

	var pills []string

	if hasPrev {
		pills = append(pills, style.Muted.Padding(0, 1).Render(completionEllipsis))
	}

	for i := pageStart; i < pageStart+visibleItems && i < total; i++ {
		st := style.Muted.Padding(0, 1)
		if i == selectedIdx {
			st = style.Primary.Bold(true).Padding(0, 1)
		}
		pills = append(pills, st.Render(completions[i]))
	}

	if hasMore {
		remaining := total - (pageStart + visibleItems)
		pills = append(pills, style.Muted.Padding(0, 1).Render(fmt.Sprintf("%s +%d", completionEllipsis, remaining)))
	}

	b.WriteString(strings.Join(pills, ""))
	b.WriteRune('\n')

}

// calcVisiblePage returns how many items fit and where the page starts
func calcVisiblePage(items []string, selectedIdx, maxWidth int) (visibleCount, pageStart int) {

	// Measure pill widths (visible chars + 2 padding)
	widths := make([]int, len(items))
	for i, s := range items {
		widths[i] = lipgloss.Width(s) + 2 // matches Padding(0,1) on each side
	}

	ellipsisW := lipgloss.Width(completionEllipsis) + 4

	// Try pages starting from multiples of a window, keeping selected visible
	start := 0

	for start < len(items) {

		budget := maxWidth

		// Reserve space for leading ellipsis if not at start
		if start > 0 {
			budget -= ellipsisW
		}

		count := 0
		for i := start; i < len(items); i++ {
			needed := widths[i]
			// Reserve trailing ellipsis if more items remain after this
			if i < len(items)-1 && budget-needed < ellipsisW+2 {
				break
			}
			budget -= needed
			count++
		}

		// Does selected fall in this window?
		if selectedIdx >= start && selectedIdx < start+count {
			return count, start
		}
		start += count

	}

	return 1, selectedIdx

}

func (app *gonduitApp) renderBottomSection(b *strings.Builder) {

	if app.cmdMgr.IsRunning() || app.prompter.IsActive() {

		// Render the main scroll view
		b.WriteRune('\n')
		b.WriteString(app.loader.Render(app.scrollView.Width()))
		b.WriteRune('\n')

		if app.prompter.IsActive() {

			// Render the text input if prompting
			b.WriteString(app.prompter.View())

		} else {

			if app.transferActive {

				// Render the file transfer status
				app.renderFileTransferStatus(b)

			} else {

				// Render execution info line
				status := style.Running.Render("● Executing") +
					style.Muted.Render(" • Press ") +
					style.Primary.Bold(true).Render("Ctrl+C") +
					style.Muted.Render(" to cancel")

				b.WriteString(status)

			}

		}

	} else {

		// Render the default divider and text input on default state
		divider := style.Divider.Render(strings.Repeat("─", app.scrollView.Width()))
		b.WriteString(divider)
		b.WriteRune('\n')
		b.WriteString(app.prompter.View())

	}

}

func (app *gonduitApp) renderFileTransferStatus(b *strings.Builder) {

	// Do not render if transfer has not started
	if app.transferTotal == 0 {
		return
	}

	// Determine the unit from the total
	_, unit := util.HumanReadableBytes(app.transferTotal)

	// Convert both using the same unit
	cur := util.HumanReadableBytesWithUnit(app.transferBytes, unit)
	mx := util.HumanReadableBytesWithUnit(app.transferTotal, unit)

	// Build UI strings
	left := fmt.Sprintf(" %s %s ", app.spinner.View(), app.transferName)
	right := fmt.Sprintf(" (%.2f/%.2f %s)", cur, mx, unit)

	// Calculate available width for the progress bar
	totalWidth := app.scrollView.Width()
	usedWidth := lipgloss.Width(left) + lipgloss.Width(right)
	barWidth := max(totalWidth-usedWidth, 5)

	app.progress.SetWidth(barWidth)

	// Compose final line
	status := style.Running.Render(left) +
		app.progress.View() +
		style.Muted.Render(right)

	b.WriteString(status)

}
