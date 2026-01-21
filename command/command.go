package command

import (
	"fmt"
	"net"
)

type Error struct {
	code    int
	message string
}

const (
	ErrNilHandler = iota
	ErrDuplicateHandler
	ErrDuplicateArgument
	ErrNotLowercaseName
	ErrUnknownHandler
	ErrBadUsage
)

type Executor func(net.Conn, []string) (string, error)

type Argument interface {
	Name() string
	Description() string
	IsPositional() bool
	Required() bool
}

type Handler interface {
	Name() string
	Executor() Executor
	Description() string
	Args() []Argument
}

type commandArgument struct {
	name        string
	description string
	positional  bool
	required    bool
}

func NewArgument(name, description string, positional, required bool) Argument {
	return &commandArgument{name, description, positional, required}
}

func (ca *commandArgument) Name() string        { return ca.name }
func (ca *commandArgument) Description() string { return ca.description }
func (ca *commandArgument) IsPositional() bool  { return ca.positional }
func (ca *commandArgument) Required() bool      { return ca.required }

type commandHandler struct {
	name        string
	executor    Executor
	description string
	args        []Argument
}

func NewHandler(name string, executor Executor, description string, args []Argument) Handler {
	argsCopy := make([]Argument, len(args))
	copy(argsCopy, args)

	return &commandHandler{name, executor, description, argsCopy}
}

func (ch *commandHandler) Name() string        { return ch.name }
func (ch *commandHandler) Executor() Executor  { return ch.executor }
func (ch *commandHandler) Description() string { return ch.description }
func (ch *commandHandler) Args() []Argument {
	result := make([]Argument, len(ch.args))
	copy(result, ch.args)
	return result
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
