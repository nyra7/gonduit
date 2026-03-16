package command

import "fmt"

const (
	ErrNilHandler = iota
	ErrDuplicateHandler
	ErrDuplicateArgument
	ErrInvalidHandlerName
	ErrUnknownHandler
	ErrBadUsage
	ErrInvalidArgument
	ErrUnimplemented
)

type Error struct {
	code   int
	reason string
}

func NewError(code int, message string, args ...any) *Error {
	return &Error{code: code, reason: fmt.Sprintf(message, args...)}
}

func (e Error) Error() string {
	return e.reason
}
func (e Error) Code() int {
	return e.code
}
