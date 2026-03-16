package completion

import "app/command"

// TokenType describes what kind of token is being completed
type TokenType int

const (
	TokenCommand TokenType = iota
	TokenFlag
	TokenFlagValue
	TokenPositional
)

// Context is the result obtained after tokenizing an input
type Context struct {
	// Command is the command name of the completing line
	Command string

	// Handler is the resolved handler for Command
	Handler command.Handler

	// Type is the kind of token the cursor is on
	Type TokenType

	// Word is the word being completed
	Word string

	// FlagName represents the flag whose value is being completed (valid when Type is TokenFlag)
	FlagName string

	// PositionalIndex is the index of the positional argument being completed (valid when Type is TokenPositional)
	PositionalIndex int
}
