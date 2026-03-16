package fmtx

import (
	"app/style"
	"fmt"
)

// Error returns a styled error message
func Error(msg string) string {
	return style.ErrorLabel.Render("✗ Error: ") + style.ErrorText.Render(msg)
}

// Errorf returns a styled error message with formatting
func Errorf(format string, args ...any) string {
	return Error(fmt.Sprintf(format, args...))
}

// Warn returns a styled warning message
func Warn(msg string) string {
	return style.WarningLabel.Render("⚠ Warning: ") + style.WarningText.Render(msg)
}

// Warnf returns a styled warning message with formatting
func Warnf(format string, args ...any) string {
	return Warn(fmt.Sprintf(format, args...))
}

// Success returns a styled success message
func Success(msg string) string {
	return style.SuccessLabel.Render("✓ Success: ") + style.SuccessText.Render(msg)
}

// Successf returns a styled success message with formatting
func Successf(format string, args ...any) string {
	return Success(fmt.Sprintf(format, args...))
}

// Debug returns a styled debug message
func Debug(msg string) string {
	return style.DebugLabel.Render("• Debug: ") + style.DebugStyle.Render(msg)
}

// Debugf returns a styled debug message with formatting
func Debugf(format string, args ...any) string {
	return Debug(fmt.Sprintf(format, args...))
}

// Info returns a styled info message
func Info(msg string) string {
	return style.InfoLabel.Render("ℹ Info: ") + style.InfoStyle.Render(msg)
}

// Infof returns a styled info message with formatting
func Infof(format string, args ...any) string {
	return Info(fmt.Sprintf(format, args...))
}

// Danger returns a danger info message
func Danger(msg string) string {
	return style.DangerLabel.Render("☢ Danger: ") + style.DangerStyle.Render(msg)
}

// Dangerf returns a styled danger message with formatting
func Dangerf(format string, args ...any) string {
	return Danger(fmt.Sprintf(format, args...))
}
