package session

import (
	"app/fmtx"
	"bytes"
	"context"
	"fmt"
	"os"
	"shared/platform"
	"shared/proto"
	"shared/util"

	"golang.org/x/term"
)

func (s *Session) messageLoop() {

	defer s.wg.Done()

	for message := range s.messages {

		switch msg := message.Message.(type) {

		case *proto.ShellServerMessage_Stdout:

			if msg.Stdout.ShellId != s.activeShell {

				// Log an error
				s.logger.Errorf("received output for inactive session %d.", msg.Stdout.ShellId)

				// Request detach to avoid subsequent messages
				_ = s.DetachShell(context.Background(), msg.Stdout.ShellId)

				continue
			}

			// Write the output to stdout
			_, _ = os.Stdout.Write(msg.Stdout.Data)

		case *proto.ShellServerMessage_Exited:

			if msg.Exited.ShellId != s.activeShell {
				continue
			}

			if msg.Exited.Reason == "" {
				s.logger.Infof("shell exited with code %d", msg.Exited.ExitCode)
			} else {
				s.logger.Warnf("shell exited with code %d: %s", msg.Exited.ExitCode, msg.Exited.Reason)
			}

			s.closeRead("shell exited")

		default:
			s.logger.Errorf("unexpected message %T", msg)
		}

	}

}

func (s *Session) inputLoop() {
	defer s.wg.Done()
	defer s.cleanupShellState()

	s.readClosed = make(chan struct{})
	buf := make([]byte, 4096)

	for {

		n, err := os.Stdin.Read(buf)

		if err != nil {
			return
		}

		select {
		case <-s.closed:
			return
		case <-s.readClosed:
			return
		default:
		}

		chunk := buf[:n]

		// Check for detach key
		if idx := bytes.IndexByte(chunk, platform.DetachKey); idx != -1 {
			if idx > 0 {
				_ = s.sendShellInput(s.activeShell, chunk[:idx])
			}
			_ = s.DetachShell(context.Background(), s.activeShell)
			return
		}

		_ = s.sendShellInput(s.activeShell, chunk)

	}

}

func (s *Session) termSize() util.TerminalSize {

	c, r, err := term.GetSize(int(os.Stdout.Fd()))

	if err != nil {
		s.logger.Warnf("failed to get terminal size: %v", err)
		return util.TerminalSize{Rows: 24, Columns: 80}
	}

	return util.TerminalSize{Rows: int32(r), Columns: int32(c)}

}

func (s *Session) closeRead(reason string) {

	if s.readClosed == nil {
		return
	}

	select {
	case <-s.readClosed:
		return
	default:
		close(s.readClosed)
	}

	if reason != "" {
		_, _ = fmt.Fprint(os.Stderr, "\r\n"+fmtx.Warnf("shell session terminated: %s. Press any key to continue", reason))
	}

}

func (s *Session) cleanupShellState() {

	if s.activeShell == 0 {
		return
	}

	s.activeShell = 0
	util.ClearTerminal()
	_ = term.Restore(int(os.Stdin.Fd()), s.initialState)
	s.listener.OnAttach(s, false)

}

func (s *Session) closeStream() error {

	var err error

	if s.stream != nil {
		err = s.stream.CloseSend()
	}

	s.stream = nil

	return err

}
