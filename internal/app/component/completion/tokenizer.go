package completion

import (
	"app/command"
	"strings"
)

// Tokenize parses rawInput and returns a Context describing what the cursor is currently positioned on
func Tokenize(rawInput string, mgr *command.Manager) Context {
	endsWithSpace := len(rawInput) > 0 && rawInput[len(rawInput)-1] == ' '
	parts := parseFields(rawInput)

	// No input yet, or still typing the command name
	if len(parts) == 0 {
		return Context{Type: TokenCommand, Word: ""}
	}

	if len(parts) == 1 && !endsWithSpace {
		return Context{Type: TokenCommand, Word: parts[0]}
	}

	commandName := parts[0]
	handler, err := mgr.FindHandler(commandName)
	if err != nil {
		// Unknown command — no further completion possible
		return Context{Command: commandName, Type: TokenCommand, Word: ""}
	}

	ctx := Context{
		Command: commandName,
		Handler: handler,
	}

	// Determine the word currently being typed
	var word string
	if !endsWithSpace && len(parts) > 1 {
		word = parts[len(parts)-1]
	}
	ctx.Word = word

	// Currently typing flag
	if strings.HasPrefix(word, "-") {
		ctx.Type = TokenFlag
		return ctx
	}

	// Possibly typing a flag value
	// e.g. "--flag <cursor>" (space after flag, value not yet started)
	if endsWithSpace && len(parts) > 0 {
		last := parts[len(parts)-1]
		if strings.HasPrefix(last, "-") {
			if _, ok := nonBoolFlagType(handler, last); ok {
				ctx.Type = TokenFlagValue
				ctx.FlagName = extractFlagName(last)
				ctx.Word = ""
				return ctx
			}
		}
	}

	// Typing a flag value, but not yet finished
	// e.g. "--flag partial<cursor>" (no space, still typing value)
	if !endsWithSpace && len(parts) >= 2 {
		prev := parts[len(parts)-2]
		if strings.HasPrefix(prev, "-") {
			if _, ok := nonBoolFlagType(handler, prev); ok {
				ctx.Type = TokenFlagValue
				ctx.FlagName = extractFlagName(prev)
				return ctx
			}
		}
	}

	// Otherwise, positional argument
	ctx.Type = TokenPositional
	ctx.PositionalIndex = countPositionalArgs(handler, parts, endsWithSpace)
	return ctx
}
