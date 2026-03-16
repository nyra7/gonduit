package command

import (
	"shared/util"
	"slices"
)

type Executor func(ctx *Context) (string, error)

type Handler interface {
	Name() string
	Executor() Executor
	Description() string
	Args() []Argument
	Flags() []Argument
	GetFlag(name string) (Argument, error)
	NumArgs() int
	NumFlags() int
}

type commandHandler struct {
	name        string
	executor    Executor
	description string
	args        []Argument
	flags       []Argument
	shorthands  map[string]string
}

func NewHandler(name string, executor Executor, description string, args []Argument, flags []Argument) Handler {

	flagsCopy := util.DupSlice(flags)
	helpArg := NewArgument("help", "Show this help message", ArgTypeBool, false)
	flagsCopy = slices.DeleteFunc[[]Argument](flagsCopy, func(arg Argument) bool { return arg.Name() == "help" })
	flagsCopy = append(flagsCopy, helpArg)

	handler := &commandHandler{
		name:        name,
		executor:    executor,
		description: description,
		args:        util.DupSlice(args),
		flags:       flagsCopy,
		shorthands:  make(map[string]string),
	}

	// Generate shorthands for flags with length > 1
	handler.generateShorthands()

	return handler
}

// generateShorthands creates automatic shorthands for flags
func (ch *commandHandler) generateShorthands() {

	// Track which first letters are used
	usedLetters := make(map[string]bool)

	// First pass: mark single-char flags as used
	for _, flag := range ch.flags {
		if len(flag.Name()) == 1 {
			usedLetters[flag.Name()] = true
		}
	}

	// Second pass: check for conflicts among multi-char flags
	firstLetterCounts := make(map[string]int)
	for _, flag := range ch.flags {
		if len(flag.Name()) > 1 {
			firstLetter := string(flag.Name()[0])
			firstLetterCounts[firstLetter]++
		}
	}

	// Third pass: assign shorthands where no conflicts exist
	for _, flag := range ch.flags {
		flagName := flag.Name()
		if len(flagName) > 1 {
			firstLetter := string(flagName[0])
			// Only create shorthand if no single-char flag and no conflict
			if !usedLetters[firstLetter] && firstLetterCounts[firstLetter] == 1 {
				ch.shorthands[firstLetter] = flagName
			}
		}
	}

}

func (ch *commandHandler) GetFlag(name string) (Argument, error) {

	// First check if it's a direct flag name
	for _, f := range ch.flags {
		if f.Name() == name {
			return f, nil
		}
	}

	// Check if it's a shorthand
	if fullName, ok := ch.shorthands[name]; ok {
		for _, f := range ch.flags {
			if f.Name() == fullName {
				return f, nil
			}
		}
	}

	return nil, NewError(ErrUnknownHandler, "unknown flag: %s", name)

}

// GetShorthand returns the shorthand for a flag name, if one exists
func (ch *commandHandler) GetShorthand(flagName string) (string, bool) {
	for short, full := range ch.shorthands {
		if full == flagName {
			return short, true
		}
	}
	return "", false
}

func (ch *commandHandler) Name() string        { return ch.name }
func (ch *commandHandler) Executor() Executor  { return ch.executor }
func (ch *commandHandler) Description() string { return ch.description }
func (ch *commandHandler) IsRemote() bool      { return ch.executor == nil }
func (ch *commandHandler) Flags() []Argument   { return ch.flags }
func (ch *commandHandler) NumArgs() int        { return len(ch.args) }
func (ch *commandHandler) NumFlags() int       { return len(ch.flags) }
func (ch *commandHandler) Args() []Argument    { return util.DupSlice(ch.args) }
