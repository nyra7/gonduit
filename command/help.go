package command

import (
	"fmt"
	"net"
	"shells/style"
	"strings"
)

func (cm *Manager) help(conn net.Conn, args []string) (string, error) {
	var output strings.Builder

	if len(args) == 0 {
		output.WriteString(fmt.Sprintf("%s\n", "\nAvailable commands:"))
		output.WriteString(strings.Repeat("-", 70) + "\n\n")

		for _, handler := range cm.handlers {
			output.WriteString(formatCommandSummary(handler))
		}

		return output.String(), nil
	}

	commandName := args[0]
	handler, err := cm.FindHandler(commandName)
	if err != nil {
		return "", fmt.Errorf("command '%s' not found. Type 'help' for available commands", commandName)
	}

	output.WriteString(formatDetailedHelp(handler))
	return output.String(), nil
}

func formatCommandSummary(handler Handler) string {
	return fmt.Sprintf("  %s  %s\n", style.Cyan.Apply(fmt.Sprintf("%-16s", handler.Name())), handler.Description())
}

func formatDetailedHelp(handler Handler) string {
	var output strings.Builder

	// Usage line
	output.WriteString(formatUsageLine(handler))
	output.WriteString("\n")

	// Description
	output.WriteString(fmt.Sprintf("%s\n", style.Bold.Apply("Description:")))
	output.WriteString(fmt.Sprintf("  %s\n\n", handler.Description()))

	// Arguments section
	args := handler.Args()
	if len(args) > 0 {
		output.WriteString(fmt.Sprintf("%s\n", style.Bold.Apply("Options:")))
		for _, arg := range args {
			output.WriteString(formatArgumentHelp(arg))
		}
	}

	return output.String()
}

func formatUsageLine(handler Handler) string {
	var usage strings.Builder

	usage.WriteString(fmt.Sprintf("%s ", style.Bold.Apply("usage:")))
	usage.WriteString(style.Cyan.Apply(handler.Name()))

	args := handler.Args()

	// Add positional arguments
	for _, arg := range args {
		if arg.IsPositional() {
			if arg.Required() {
				usage.WriteString(fmt.Sprintf(" %s", style.Green.Apply(fmt.Sprintf("<%s>", arg.Name()))))
			} else {
				usage.WriteString(fmt.Sprintf(" (%s)", style.Green.Apply(fmt.Sprintf("<%s>", arg.Name()))))
			}
		}
	}

	// Add optional flags
	for _, arg := range args {
		if !arg.IsPositional() {
			flagName := fmt.Sprintf("--%s", arg.Name())
			if arg.Required() {
				usage.WriteString(fmt.Sprintf(" %s %s", style.Yellow.Apply(flagName), style.Green.Apply(fmt.Sprintf("<%s>", arg.Name()))))
			} else {
				usage.WriteString(fmt.Sprintf(" %s", fmt.Sprintf("(%s)", style.Yellow.Apply(flagName))))
			}
		}
	}

	return usage.String()
}

func formatArgumentHelp(arg Argument) string {
	var line strings.Builder

	// Format argument name
	name := arg.Name()
	if !arg.IsPositional() {
		name = fmt.Sprintf("--%s", name)
	}
	/*
		// Type indicator
		argType := "option"
		if arg.IsPositional() {
			argType = "positional"
		}

		// Required indicator
		var requiredIndicator string
		if arg.Required() {
			requiredIndicator = fmt.Sprintf(" %s", style.BoldMagenta.Apply("[required]"))
		}
	*/

	line.WriteString(fmt.Sprintf("  %s %s\n", style.Cyan.Apply(fmt.Sprintf("%-28s", name)), arg.Description()))

	return line.String()
}
