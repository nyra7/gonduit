package completion

import (
	"app/command"
	"strings"
)

// ReplaceLastToken replaces the last whitespace-delimited token in rawInput with chosen
func ReplaceLastToken(rawInput string, chosen string) string {
	if len(rawInput) == 0 {
		return chosen
	}

	endsWithSpace := rawInput[len(rawInput)-1] == ' '

	parts := parseFields(rawInput)

	if len(parts) == 0 {
		return chosen
	}

	if endsWithSpace {
		return rawInput + chosen
	}

	parts[len(parts)-1] = chosen
	return strings.Join(parts, " ")
}

// parseFields splits s on unquoted spaces, stripping double-quotes
func parseFields(s string) []string {
	var fields []string
	var cur strings.Builder
	inQuote := false

	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '"':
			inQuote = !inQuote
		case ' ':
			if inQuote {
				cur.WriteByte(s[i])
			} else if cur.Len() > 0 {
				fields = append(fields, cur.String())
				cur.Reset()
			}
		default:
			cur.WriteByte(s[i])
		}
	}
	if cur.Len() > 0 {
		fields = append(fields, cur.String())
	}
	return fields
}

// extractFlagName strips leading dashes from a flag token
func extractFlagName(flagStr string) string {
	s := strings.TrimLeft(flagStr, "-")
	return s
}

// nonBoolFlagType returns the argument type of a flag if it is non-boolean
func nonBoolFlagType(h command.Handler, flagToken string) (command.ArgumentType, bool) {
	name := extractFlagName(flagToken)
	flag, err := h.GetFlag(name)
	if err != nil {
		return "", false
	}
	if flag.Type() == command.ArgTypeBool {
		return "", false
	}
	return flag.Type(), true
}

// countPositionalArgs counts how many positional arguments appear in parts[1:]
func countPositionalArgs(h command.Handler, parts []string, endsWithSpace bool) int {
	limit := len(parts)
	if !endsWithSpace && limit > 1 {
		// don't count the word being typed
		limit--
	}

	count := 0
	for i := 1; i < limit; i++ {
		part := parts[i]
		if strings.HasPrefix(part, "-") {
			// If the flag expects a value, skip the next token too
			if _, ok := nonBoolFlagType(h, part); ok {
				i++
			}
			continue
		}
		// Verify this token is not the value of the preceding flag
		if i > 0 && strings.HasPrefix(parts[i-1], "-") {
			if _, ok := nonBoolFlagType(h, parts[i-1]); ok {
				// already consumed as flag value above
				continue
			}
		}
		count++
	}
	return count
}
