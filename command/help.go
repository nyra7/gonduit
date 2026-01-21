package command

import (
	"fmt"
	"gonduit/style"
	"gonduit/util"
	"strings"
)

func (cm *Manager) help(ctx *Context) error {

	commandName, err := ctx.Next()

	if err != nil {

		var output strings.Builder

		output.WriteString(fmt.Sprintf("%s\n", "\nAvailable commands:"))
		output.WriteString(strings.Repeat("-", 70) + "\n\n")

		for _, handler := range cm.handlers {
			output.WriteString(formatCommandSummary(handler))
		}

		util.WriteConn(ctx.Conn, output.String()+"\n")

		return nil

	}

	handler, err := cm.FindHandler(commandName)
	if err != nil {
		return fmt.Errorf("command '%s' not found. Type 'help' for available commands", commandName)
	}

	util.WriteConn(ctx.Conn, formatDetailedHelp(handler))

	return nil
}

func formatCommandSummary(handler Handler) string {
	return fmt.Sprintf("  %s  %s\n", style.Cyan.Apply(fmt.Sprintf("%-16s", handler.Name())), handler.Description())
}

func formatDetailedHelp(handler Handler) string {
	var output strings.Builder

	// Usage line
	output.WriteString("\n")
	output.WriteString(formatUsageLine(handler))
	output.WriteString("\n\n")

	// Description
	output.WriteString(fmt.Sprintf("%s\n", style.Bold.Apply("Description:")))
	output.WriteString(fmt.Sprintf("  %s\n\n", handler.Description()))

	// Arguments section
	args := handler.Args()
	if len(args) > 0 {
		output.WriteString(fmt.Sprintf("%s\n", style.Bold.Apply("Arguments:")))
		for _, arg := range args {
			output.WriteString(formatArgumentHelp(arg, false))
		}

		output.WriteString("\n")
	}

	// Flags section
	flags := handler.Flags()
	if len(flags) > 0 {
		output.WriteString(fmt.Sprintf("%s\n", style.Bold.Apply("Flags:")))
		for _, arg := range flags {
			output.WriteString(formatArgumentHelp(arg, true))
		}

		output.WriteString("\n")
	}

	return output.String()
}

func formatUsageLine(handler Handler) string {
	var usage strings.Builder

	usage.WriteString(fmt.Sprintf("%s ", style.Bold.Apply("usage:")))
	usage.WriteString(style.Cyan.Apply(handler.Name()))

	// Add flags
	for _, arg := range handler.Flags() {
		flagName := fmt.Sprintf("--%s", arg.Name())
		if arg.Required() {
			usage.WriteString(fmt.Sprintf(" %s %s", style.Yellow.Apply(flagName), style.Magenta.Apply(fmt.Sprintf("<%s>", arg.Name()))))
		} else {
			usage.WriteString(fmt.Sprintf(" %s", fmt.Sprintf("(%s)", style.Yellow.Apply(flagName))))
		}
	}

	// Add positional arguments
	for _, arg := range handler.Args() {
		if arg.Type() == ArgTypeVariadic {
			usage.WriteString(fmt.Sprintf(" %s", style.Gray.Apply(arg.Name()+"...")))
		} else if arg.Required() {
			usage.WriteString(style.Magenta.Apply(fmt.Sprintf(" <%s>", arg.Name())))
		} else {
			usage.WriteString(fmt.Sprintf(" (%s)", style.Magenta.Apply(fmt.Sprintf("<%s>", arg.Name()))))
		}
	}

	return usage.String()
}

func formatArgumentHelp(arg Argument, flag bool) string {
	var line strings.Builder

	// Format argument name
	name := arg.Name()
	color := style.Magenta

	if flag {
		name = fmt.Sprintf("--%s", name)
		color = style.Yellow
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

	line.WriteString(fmt.Sprintf("  %s %s\n", color.Apply(fmt.Sprintf("%-28s", name)), arg.Description()))

	return line.String()
}

func MakeHelpCommand(manager *Manager) Handler {

	helpArgs := []Argument{
		NewArgument("command", "Provides help about a specific command", ArgTypeString, false),
	}

	return NewHandler("help", manager.help, "Show this help message", helpArgs, []Argument{})

}
