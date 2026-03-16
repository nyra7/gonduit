package pty

import (
	"context"
	"fmt"
	"shared/util"
)

type ExitError struct {
	ExitCode int
}

func (e *ExitError) Error() string {
	return fmt.Sprintf("exit code %d", e.ExitCode)
}

type Pty interface {
	Start(ctx context.Context, cmd string, args ...string) error
	Close() error
	Resize(size util.TerminalSize) error
	Read(b []byte) (n int, err error)
	Write(b []byte) (n int, err error)
	Wait() error
	SetEcho(mode bool) error
}
