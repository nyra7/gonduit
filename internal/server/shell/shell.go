package shell

import (
	"context"
	"fmt"
	"server/log"
	"server/shell/pty"
	"shared/util"
	"sync"
	"sync/atomic"
)

const (
	// Maximum history buffer size (4MB)
	maxHistoryBufferSize = 4096 * 1024

	// Size to keep when truncating history (256KB)
	truncatedHistorySize = 256 * 1024
)

var (
	idGen = util.NewIDGenerator()
)

type Shell struct {

	// id is the unique identifier for this shell instance
	id uint64

	// path is the full shell executable path
	path string

	// historyBuffer stores the terminal output history
	historyBuffer []byte

	// lastSent tracks the last position sent to an attached client
	lastSent int

	// term is the underlying pseudo-terminal
	term pty.Pty

	//////// Channels

	// input receives data to be written to the shell
	input chan []byte

	// resize receives terminal size change requests
	resize chan util.TerminalSize

	// done signals shell termination with an optional error
	done chan error

	// outputNotify signals that new output is available
	outputNotify chan struct{}

	//////// Sync variables

	// mu protects shared state access
	mu sync.Mutex

	// ctx is the shell's lifecycle context
	ctx context.Context

	// wg tracks active goroutines
	wg sync.WaitGroup

	// running indicates if the shell process is running
	running atomic.Bool

	// attached indicates if a client is attached
	attached atomic.Bool

	// closed indicates if the shell has been closed
	closed atomic.Bool

	// shutdown cancels the shell's context
	shutdown context.CancelFunc

	// attachCancel cancels the attachment context
	attachCancel context.CancelFunc
}

func NewShell(ctx context.Context, shell string) *Shell {
	newCtx, shutdown := context.WithCancel(ctx)
	return &Shell{
		path:          shell,
		input:         make(chan []byte),
		historyBuffer: make([]byte, 0, 32768),
		resize:        make(chan util.TerminalSize),
		outputNotify:  make(chan struct{}, 1),
		done:          make(chan error, 1),
		ctx:           newCtx,
		shutdown:      shutdown,
		id:            idGen.Next(),
	}
}

func (s *Shell) Run(args ...string) error {

	if !s.running.CompareAndSwap(false, true) {
		return fmt.Errorf("path was already run")
	}

	go s.start(args...)

	return nil

}

func (s *Shell) ID() uint64 {
	return s.id
}

func (s *Shell) Path() string {
	return s.path
}

func (s *Shell) Resize(size util.TerminalSize) {
	if s.closed.Load() {
		return
	}
	s.resize <- size
}

func (s *Shell) Write(data []byte) {
	select {
	case s.input <- data:
	case <-s.ctx.Done():
	}
}

func (s *Shell) WaitChannel() <-chan error {
	return s.done
}

func (s *Shell) Attach(stream *Stream) ([]byte, error) {

	if !s.running.Load() {
		return nil, fmt.Errorf("path is not running")
	}

	if !s.attached.CompareAndSwap(false, true) {
		return nil, fmt.Errorf("already attached")
	}

	// Capture history to return
	s.mu.Lock()
	history := append([]byte(nil), s.historyBuffer...)
	s.lastSent = len(s.historyBuffer)
	s.mu.Unlock()

	ctx, cancel := context.WithCancel(s.ctx)
	s.attachCancel = cancel

	s.wg.Add(1)

	go func() {
		defer s.wg.Done()
		defer s.attached.Store(false)

		for {
			select {
			case <-ctx.Done():
				return
			case <-s.ctx.Done():
				return
			case <-s.outputNotify:
				if err := stream.SendStdout(s.id, s.readNewOutput()); err != nil {
					log.Errorf("failed to send path output: %v", err)
					return
				}
			}
		}
	}()

	log.Infof("shell %d attached to %s", s.id, stream.RemoteAddr())

	return history, nil

}

func (s *Shell) Detach() {

	if !s.attached.Load() {
		return
	}

	s.attached.Store(false)

	if s.attachCancel != nil {
		s.attachCancel()
	}

	s.lastSent = 0

	log.Infof("shell %d detached", s.id)

}

func (s *Shell) Close() {

	// Check if already closed
	if !s.closed.CompareAndSwap(false, true) {
		return
	}

	// Signal shutdown to all goroutines
	s.shutdown()

	// Detach if still attached
	s.Detach()

	// Wait for all goroutines to finish
	s.wg.Wait()

	// Cleanup channels last (after goroutines have exited)
	close(s.input)
	close(s.resize)
	close(s.outputNotify)

	// Cleanup the PTY
	if s.term != nil {
		_ = s.term.Close()
	}

	// Clear references
	s.term = nil

}

func (s *Shell) IsAttached() bool {
	return s.attached.Load()
}

func (s *Shell) start(args ...string) {

	defer s.Close()

	s.term = pty.NewPty()
	err := s.term.Start(s.ctx, s.path, args...)

	if err != nil {
		s.done <- fmt.Errorf("error spawning path: %w", err)
		return
	}

	s.wg.Add(3)
	go s.handleInput()
	go s.handleOutput()
	go s.handleTermSize()

	s.done <- s.term.Wait()

	log.Infof("shell %d process exited", s.id)

}

func (s *Shell) readNewOutput() []byte {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.lastSent >= len(s.historyBuffer) {
		return nil
	}

	newOutput := append([]byte(nil), s.historyBuffer[s.lastSent:]...)
	s.lastSent = len(s.historyBuffer)
	return newOutput
}

func (s *Shell) handleInput() {
	defer s.wg.Done()

	for {
		select {
		case <-s.ctx.Done():
			return
		case data, ok := <-s.input:
			if !ok {
				return
			}
			if _, err := s.term.Write(data); err != nil {
				log.Errorf("failed to write data to path: %v", err)
				return
			}
		}
	}
}

func (s *Shell) handleOutput() {
	defer s.wg.Done()
	tmp := make([]byte, 4096)

	for {

		// Exit routine if shutdown is requested
		select {
		case <-s.ctx.Done():
			return
		default:
		}

		// Try to read input, exit on failure
		n, err := s.term.Read(tmp)

		if err != nil {
			return
		}

		s.mu.Lock()

		// Cap history buffer size to prevent unbounded growth
		if len(s.historyBuffer) > maxHistoryBufferSize {
			// Keep only the most recent data
			truncateBy := len(s.historyBuffer) - truncatedHistorySize
			s.historyBuffer = append([]byte(nil), s.historyBuffer[truncateBy:]...)
			// Adjust lastSent pointer to account for truncation
			s.lastSent = max(0, s.lastSent-truncateBy)
		}

		s.historyBuffer = append(s.historyBuffer, tmp[:n]...)

		s.mu.Unlock()

		// Notify attached client if any
		select {
		case s.outputNotify <- struct{}{}:
		default:
		}

	}
}

func (s *Shell) handleTermSize() {
	defer s.wg.Done()

	for {
		select {
		case <-s.ctx.Done():
			return
		case ws, ok := <-s.resize:
			if !ok {
				return
			}

			// Validate terminal size
			if ws.Rows < 1 || ws.Rows > 1000 || ws.Columns < 1 || ws.Columns > 1000 {
				log.Errorf("invalid terminal size: %dx%d", ws.Columns, ws.Rows)
				continue
			}

			s.mu.Lock()

			if err := s.term.Resize(ws); err != nil {
				log.Errorf("failed to resize PTY: %v", err)
			} else {
				log.Infof("resized PTY %d to %dx%d", s.id, ws.Columns, ws.Rows)
			}

			s.mu.Unlock()

		}
	}
}

func (s *Shell) History() []byte {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]byte(nil), s.historyBuffer...)
}
