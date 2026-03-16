package component

import (
	"app/fmtx"
	"shared/pkg"
	"strings"

	"github.com/charmbracelet/x/cellbuf"
)

// ScrollView represents a scrollable view of text with line wrapping and offset management
type ScrollView struct {
	lines  []string
	offset int
	width  int
	height int
}

func NewScrollView() *ScrollView {
	return &ScrollView{}
}

// WrappedLines returns all output lines wrapped to the given width
func (v *ScrollView) WrappedLines() []string {
	if v.width <= 0 {
		return append([]string(nil), v.lines...)
	}

	wrapped := make([]string, 0, len(v.lines))
	for _, line := range v.lines {
		if line == "" {
			wrapped = append(wrapped, "")
			continue
		}

		wrappedLine := cellbuf.Wrap(line, v.width, "")
		if wrappedLine == "" {
			wrapped = append(wrapped, "")
			continue
		}
		wrapped = append(wrapped, strings.Split(wrappedLine, "\n")...)
	}
	return wrapped
}

// VisibleLines returns the lines that fit in the available space with wrapping
func (v *ScrollView) VisibleLines(available int) []string {
	if available <= 0 {
		return []string{}
	}

	wrapped := v.WrappedLines()
	if len(wrapped) == 0 {
		return []string{}
	}

	start := 0
	if len(wrapped) > available {
		start = len(wrapped) - available - v.offset
		if start < 0 {
			start = 0
		}
	}

	end := start + available
	if end > len(wrapped) {
		end = len(wrapped)
	}

	return wrapped[start:end]
}

// AddLine appends a new line to the output
func (v *ScrollView) AddLine(line string) {
	v.lines = append(v.lines, line)
}

// Clear removes all output lines
func (v *ScrollView) Clear() {
	v.lines = []string{}
}

// Reset resets the scroll offset
func (v *ScrollView) Reset() {
	v.offset = 0
}

// ScrollUp scrolls up by one line
func (v *ScrollView) ScrollUp(available int) {
	totalLines := len(v.WrappedLines())
	maxScroll := totalLines - available
	if maxScroll > 0 && v.offset < maxScroll {
		v.offset += 1
	}
}

// ScrollDown scrolls down by one line
func (v *ScrollView) ScrollDown() {
	if v.offset > 0 {
		v.offset -= 1
	}
}

func (v *ScrollView) Width() int {
	return v.width
}

func (v *ScrollView) Height() int {
	return v.height
}

func (v *ScrollView) SetDimensions(width int, height int) {
	v.width = width
	v.height = height
}

// Debug adds a styled debug line to the output
func (v *ScrollView) Debug(msg string) {
	if pkg.IsDebug() {
		v.Write(fmtx.Debug(msg))
	}
}

// Debugf adds a styled debug line with formatting
func (v *ScrollView) Debugf(format string, args ...any) {
	v.Write(fmtx.Debugf(format, args...))
}

// Error adds a styled error line to the output
func (v *ScrollView) Error(msg string) {
	v.Write(fmtx.Error(msg))
}

// Errorf adds a styled error line with formatting
func (v *ScrollView) Errorf(format string, args ...any) {
	v.Write(fmtx.Errorf(format, args...))
}

// Warn adds a styled warning line to the output
func (v *ScrollView) Warn(msg string) {
	v.Write(fmtx.Warn(msg))
}

// Warnf adds a styled warning line with formatting
func (v *ScrollView) Warnf(format string, args ...any) {
	v.Write(fmtx.Warnf(format, args...))
}

// Success adds a styled success line to the output
func (v *ScrollView) Success(msg string) {
	v.Write(fmtx.Success(msg))
}

// Successf adds a styled success line with formatting
func (v *ScrollView) Successf(format string, args ...any) {
	v.Write(fmtx.Successf(format, args...))
}

// Info adds a styled info line to the output
func (v *ScrollView) Info(msg string) {
	v.Write(fmtx.Info(msg))
}

// Infof adds a styled info line with formatting
func (v *ScrollView) Infof(format string, args ...any) {
	v.Write(fmtx.Infof(format, args...))
}

func (v *ScrollView) Danger(msg string) {
	v.Write(fmtx.Danger(msg))
}

func (v *ScrollView) Dangerf(format string, args ...any) {
	v.Write(fmtx.Dangerf(format, args...))
}

// Write adds a line to the output
func (v *ScrollView) Write(msg string) {
	v.AddLine(msg)
}
