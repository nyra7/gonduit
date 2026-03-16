package command

import (
	"app/color"
	"fmt"
	"strings"
)

// Default terminal width if not available
const defaultTermWidth = 80

func help(ctx *Context) (string, error) {
	m := ctx.Manager()
	parts := ctx.Variadic(0)

	termCols, _ := m.TermSize()
	if termCols == 0 {
		termCols = defaultTermWidth
	}

	if len(parts) == 0 {
		var output strings.Builder

		handlers := m.Handlers()

		// Header matching app UI style
		output.WriteString("\n")
		output.WriteString(color.RGB(color.Primary.ToANSI(), color.BoldText(" AVAILABLE COMMANDS")))
		output.WriteString("\n")
		output.WriteString(color.RGB(color.Border.ToANSI(), strings.Repeat("─", termCols)))
		output.WriteString("\n")

		// Local builtin
		if len(handlers) > 0 {
			for _, handler := range handlers {
				output.WriteString(formatCommandSummary(handler, termCols))
			}
		}

		output.WriteString("\n")
		output.WriteString(color.RGB(color.Muted.ToANSI(), "Tip: Use 'help <command>' for detailed information"))
		output.WriteString("\n")

		return output.String(), nil
	}

	commandName := parts[0]

	handler, err := m.FindHandler(commandName)
	if err != nil {
		return "", NewError(ErrInvalidArgument, "command '%s' not found. Type 'help' for available builtin", commandName)
	}

	return formatDetailedHelp(handler, termCols), nil
}

func formatCommandSummary(handler Handler, termCols int) string {
	const nameWidth = 16
	const padding = 2

	cmdName := color.RGB(color.Blue.ToANSI(), fmt.Sprintf("%-16s", handler.Name()))

	// Calculate available width for description
	descMaxWidth := termCols - nameWidth - padding*2 - 4 // Account for spacing
	if descMaxWidth < 20 {
		descMaxWidth = 20
	}

	descIndent := strings.Repeat(" ", nameWidth+padding*2)
	description := wrapText(handler.Description(), descMaxWidth, descIndent)

	return fmt.Sprintf("  %s  %s\n", cmdName, description)
}

func formatDetailedHelp(handler Handler, termCols int) string {
	var output strings.Builder

	// Header with command name
	output.WriteString("\n")
	output.WriteString(color.RGB(color.Primary.ToANSI(), color.BoldText(fmt.Sprintf(" %s", strings.ToUpper(handler.Name())))))
	output.WriteString("\n")
	output.WriteString(color.RGB(color.Border.ToANSI(), strings.Repeat("─", termCols)))
	output.WriteString("\n\n")

	// Description
	output.WriteString(color.RGB(color.Primary.ToANSI(), color.BoldText("Description")))
	output.WriteString("\n")
	output.WriteString(handler.Description())
	output.WriteString("\n\n")

	// Usage line
	output.WriteString(color.RGB(color.Primary.ToANSI(), color.BoldText("Usage")))
	output.WriteString("\n")
	output.WriteString(fmt.Sprintf("  %s\n\n", formatUsageLine(handler)))

	// Arguments section
	args := handler.Args()
	if len(args) > 0 {
		output.WriteString(color.RGB(color.Primary.ToANSI(), color.BoldText("Arguments")))
		output.WriteString("\n")
		for _, arg := range args {
			output.WriteString(formatArgumentHelp(arg, false, termCols, handler))
		}
		output.WriteString("\n")
	}

	// Flags section
	flags := handler.Flags()
	if len(flags) > 0 {
		output.WriteString(color.RGB(color.Primary.ToANSI(), color.BoldText("Flags")))
		output.WriteString("\n")
		for _, arg := range flags {
			output.WriteString(formatArgumentHelp(arg, true, termCols, handler))
		}
		output.WriteString("\n")
	}

	return output.String()
}

func formatUsageLine(handler Handler) string {
	var usage strings.Builder

	usage.WriteString(color.RGB(color.Blue.ToANSI(), handler.Name()))

	// Add flags
	ch, hasShorthands := handler.(*commandHandler)
	for _, arg := range handler.Flags() {
		var flagName string
		if hasShorthands {
			if shorthand, ok := ch.GetShorthand(arg.Name()); ok {
				flagName = fmt.Sprintf("-%s|--%s", shorthand, arg.Name())
			} else if len(arg.Name()) == 1 {
				flagName = fmt.Sprintf("-%s", arg.Name())
			} else {
				flagName = fmt.Sprintf("--%s", arg.Name())
			}
		} else {
			flagName = fmt.Sprintf("--%s", arg.Name())
		}

		if arg.Required() {
			usage.WriteString(fmt.Sprintf(" %s %s", color.RGB(color.Amber.ToANSI(), flagName), color.RGB(color.Blue.ToANSI(), fmt.Sprintf("<%s>", arg.Name()))))
		} else {
			usage.WriteString(fmt.Sprintf(" %s", color.RGB(color.Muted.ToANSI(), fmt.Sprintf("[%s]", flagName))))
		}
	}

	// Add positional arguments
	for _, arg := range handler.Args() {
		if arg.Type() == ArgTypeVariadic {
			usage.WriteString(fmt.Sprintf(" %s", color.RGB(color.Muted.ToANSI(), arg.Name()+"...")))
		} else if arg.Required() {
			usage.WriteString(color.RGB(color.Blue.ToANSI(), fmt.Sprintf(" <%s>", arg.Name())))
		} else {
			usage.WriteString(fmt.Sprintf(" %s", color.RGB(color.Muted.ToANSI(), fmt.Sprintf("[%s]", arg.Name()))))
		}
	}

	return usage.String()
}

func formatArgumentHelp(arg Argument, flag bool, termCols int, handler Handler) string {
	var line strings.Builder

	// Fixed width for argument/flag name (increased for shorthands)
	const nameWidth = 24

	// Fixed width for metadata (optional/required/default)
	const metadataWidth = 32

	// Left padding before name
	const leftPadding = 2

	// Gap between columns
	const gapBetweenColumns = 2

	// Where description starts
	const descriptionColumn = leftPadding + nameWidth + gapBetweenColumns + metadataWidth + gapBetweenColumns

	// Remaining width for description
	const descriptionWidth = 80

	// Format argument name
	name := arg.Name()
	nameColor := color.Blue.ToANSI()

	if flag {
		ch, hasShorthands := handler.(*commandHandler)
		if hasShorthands {
			if shorthand, ok := ch.GetShorthand(arg.Name()); ok {
				name = fmt.Sprintf("-%s, --%s", shorthand, arg.Name())
			} else if len(arg.Name()) == 1 {
				name = fmt.Sprintf("-%s", arg.Name())
			} else {
				name = fmt.Sprintf("--%s", arg.Name())
			}
		} else {
			name = fmt.Sprintf("--%s", arg.Name())
		}
		nameColor = color.Amber.ToANSI()
	}

	formattedName := color.RGB(nameColor, name)

	// Build metadata string (optional/required/default indicators)
	var metadata []string
	if !arg.Required() {
		metadata = append(metadata, color.RGB(color.Muted.ToANSI(), "optional"))
	} else {
		metadata = append(metadata, color.RGB(color.Blue.ToANSI(), "required"))
	}
	if def, ok := arg.DefaultValue(); ok {
		metadata = append(metadata, color.RGB(color.GreenLight.ToANSI(), fmt.Sprintf("default: %s", formatDefaultValue(def))))
	}

	metaStr := strings.Join(metadata, color.RGB(color.Muted.ToANSI(), " | "))

	// Calculate padding for name column
	namePadding := nameWidth - stripANSILen(formattedName)
	if namePadding < 1 {
		namePadding = 1
	}

	// Calculate padding for metadata column
	metaPadding := metadataWidth - stripANSILen(metaStr)
	if metaPadding < 1 {
		metaPadding = 1
	}

	// Wrap description text
	descIndent := strings.Repeat(" ", descriptionColumn-4)
	wrapped := wrapText(arg.Description(), descriptionWidth, descIndent)

	// Format the line: indent + name + padding + metadata + padding + description
	line.WriteString(strings.Repeat(" ", leftPadding))
	line.WriteString(formattedName)
	line.WriteString(strings.Repeat(" ", namePadding))
	line.WriteString(metaStr)
	line.WriteString(strings.Repeat(" ", metaPadding))
	line.WriteString(wrapped)
	line.WriteString("\n")

	return line.String()
}

func formatDefaultValue(value any) string {
	const maxLen = 14
	formatted := fmt.Sprintf("%v", value)
	if len(formatted) <= maxLen {
		return formatted
	}
	return formatted[:maxLen-3] + "..."
}

// stripANSILen returns the visible length of a string (without ANSI codes)
func stripANSILen(str string) int {
	length := 0
	inEscape := false
	for _, ch := range str {
		if ch == '\033' {
			inEscape = true
		} else if inEscape && ch == 'm' {
			inEscape = false
		} else if !inEscape {
			length++
		}
	}
	return length
}

// wrapText wraps text to maxWidth, with subsequent lines indented
func wrapText(text string, maxWidth int, indent string) string {
	if len(text) <= maxWidth {
		return text
	}

	var result strings.Builder
	words := strings.Fields(text)
	lineLen := 0
	firstLine := true

	for _, word := range words {
		wordLen := len(word)

		if lineLen == 0 {
			// Start of new line
			if !firstLine {
				result.WriteString(indent)
			}
			result.WriteString(word)
			lineLen = wordLen
			firstLine = false
		} else if lineLen+1+wordLen <= maxWidth {
			// Word fits on current line
			result.WriteString(" ")
			result.WriteString(word)
			lineLen += 1 + wordLen
		} else {
			// Word doesn't fit, start new line
			result.WriteString("\n")
			result.WriteString(indent)
			result.WriteString(word)
			lineLen = wordLen
		}
	}

	return result.String()
}

func MakeHelpCommand() Handler {

	helpArgs := []Argument{
		NewArgument("command", "Provides help about a specific command", ArgTypeVariadic, false),
	}

	return NewHandler("help", help, "Show this help message", helpArgs, []Argument{})

}
