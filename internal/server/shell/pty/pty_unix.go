//go:build !windows

package pty

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path"
	"shared/util"
	"sync"

	"github.com/creack/pty"
	"golang.org/x/sys/unix"
)

type UnixPty struct {
	cmd  *exec.Cmd
	term *os.File
	mu   sync.Mutex
}

func (p *UnixPty) Start(ctx context.Context, command string, args ...string) error {

	shell := path.Base(command)

	switch shell {
	case "bash", "zsh", "sh":
		args = append([]string{"-i"}, args...)
	}

	cmd := exec.CommandContext(ctx, command, args...)
	cmd.Env = append(os.Environ(), "TERM=xterm-256color")
	term, err := pty.StartWithSize(cmd, &pty.Winsize{Rows: 24, Cols: 80})

	if err != nil {
		return err
	}

	p.cmd = cmd
	p.term = term

	return nil

}

func (p *UnixPty) Close() error {

	p.mu.Lock()
	defer p.mu.Unlock()

	// Kill the process
	if p.cmd != nil && p.cmd.Process != nil {
		_ = p.cmd.Process.Kill()
	}

	// Cleanup the PTY
	if p.term != nil {
		_ = p.term.Close()
	}

	p.cmd = nil
	p.term = nil

	return nil

}

func (p *UnixPty) Resize(size util.TerminalSize) error {
	return pty.Setsize(p.term, &pty.Winsize{Rows: uint16(size.Rows), Cols: uint16(size.Columns)})
}

func (p *UnixPty) Read(b []byte) (n int, err error) {
	return p.term.Read(b)
}

func (p *UnixPty) Write(b []byte) (n int, err error) {
	return p.term.Write(b)
}

func (p *UnixPty) Wait() error {

	err := p.cmd.Wait()

	var exitErr *exec.ExitError

	if errors.As(err, &exitErr) {
		return &ExitError{ExitCode: exitErr.ExitCode()}
	}

	return err

}

func (p *UnixPty) SetEcho(enabled bool) error {

	fd := int(p.term.Fd())
	get, set := termiosIoCtl()

	t, err := unix.IoctlGetTermios(fd, get)
	if err != nil {
		return err
	}

	if enabled {
		t.Lflag |= unix.ECHO | unix.ECHONL
	} else {
		t.Lflag &^= unix.ECHO | unix.ECHONL
	}

	return unix.IoctlSetTermios(fd, set, t)

}

func NewPty() Pty {
	return &UnixPty{}
}
