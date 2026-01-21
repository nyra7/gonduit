package command

import (
	"fmt"
)

type ArgumentType string

var (
	ArgTypeString   ArgumentType = "string"
	ArgTypeInt      ArgumentType = "int"
	ArgTypeFloat    ArgumentType = "float"
	ArgTypeBool     ArgumentType = "bool"
	ArgTypeVariadic ArgumentType = "variadic"
)

type Error struct {
	code    int
	message string
}

const (
	ErrNilHandler = iota
	ErrDuplicateHandler
	ErrDuplicateArgument
	ErrInvalidHandlerName
	ErrUnknownHandler
	ErrBadUsage
)

type Executor func(ctx *Context) error

type Argument interface {
	Name() string
	Description() string
	Type() ArgumentType
	Required() bool
}

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

type commandArgument struct {
	name        string
	description string
	argType     ArgumentType
	required    bool
}

func NewArgument(name, description string, argType ArgumentType, required bool) Argument {
	return &commandArgument{name, description, argType, required}
}

func (ca *commandArgument) Name() string        { return ca.name }
func (ca *commandArgument) Description() string { return ca.description }
func (ca *commandArgument) Type() ArgumentType  { return ca.argType }
func (ca *commandArgument) Required() bool      { return ca.required }

type commandHandler struct {
	name        string
	executor    Executor
	description string
	args        []Argument
	flags       []Argument
}

func NewHandler(name string, executor Executor, description string, args []Argument, flags []Argument) Handler {
	argsCopy := make([]Argument, len(args))
	copy(argsCopy, args)
	flagsCopy := make([]Argument, len(flags))
	copy(flagsCopy, flags)

	return &commandHandler{name, executor, description, argsCopy, flagsCopy}
}

func (ch *commandHandler) Name() string        { return ch.name }
func (ch *commandHandler) Executor() Executor  { return ch.executor }
func (ch *commandHandler) Description() string { return ch.description }
func (ch *commandHandler) Flags() []Argument   { return ch.flags }
func (ch *commandHandler) NumArgs() int        { return len(ch.args) }
func (ch *commandHandler) NumFlags() int       { return len(ch.flags) }
func (ch *commandHandler) Args() []Argument {
	result := make([]Argument, len(ch.args))
	copy(result, ch.args)
	return result
}

func (ch *commandHandler) GetFlag(name string) (Argument, error) {

	for _, f := range ch.flags {
		if f.Name() == name {
			return f, nil
		}
	}

	return nil, NewError(ErrUnknownHandler, "unknown flag: %s", name)

}

func NewError(code int, message string, args ...interface{}) *Error {
	return &Error{
		code:    code,
		message: fmt.Sprintf(message, args...),
	}
}

func (e Error) Error() string {
	return e.message
}
func (e Error) Code() int {
	return e.code
}
