package core

import "time"

// TickMsg is sent periodically to update animations
type TickMsg time.Time

// ErrMsg represents an error message
type ErrMsg error

// CommandResultMsg contains the result of a command execution
type CommandResultMsg struct {
	Output string
	Err    error
}

// CommandCancelMsg requests cancellation of the running command
type CommandCancelMsg struct{}

type ExitMsg struct{}
