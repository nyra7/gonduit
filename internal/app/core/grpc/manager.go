package grpc

import (
	"app/component"
	"app/core/grpc/session"
	"app/log"
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"os"
	"shared/platform"
	"shared/util"
	"sync"

	"golang.org/x/term"
)

var (
	ErrNoActiveSession = errors.New("no active session")
)

type AltViewFunc func(bool)

type PromptFunc func(ctx context.Context, promptText string, secure bool) (string, error)

type TransferUpdateFunc func(filename string, current, total uint64)

type Manager struct {

	// logger is the logger used by the manager
	logger log.Logger

	// sessions is a map of the active sessions
	sessions map[uint64]*session.Session

	// activeSession is the ID of the active session
	activeSession uint64

	// state is the initial terminal state upon Manager creation
	state *term.State

	// listener is the listener for incoming connections
	listener net.Listener

	// tlsConfig is the TLS configuration for the server
	tlsConfig *tls.Config

	// selfSigned indicates whether the server is using a self-signed identity
	selfSigned bool

	// altView is called when the user switches between the main and session views
	altView AltViewFunc

	// transferUpdate is called when a file transfer state is updated
	transferUpdate TransferUpdateFunc

	// verifiedCerts is a map of peer certificates that have been verified/accepted
	verifiedCerts map[string]bool

	// verifiedCertsMu is used to synchronize access to verifiedCerts
	verifiedCertsMu sync.Mutex

	// prompter is the component used to prompt the user for unknown connections
	prompter *component.Prompter

	// ctx is the manager context used for graceful shutdown
	ctx context.Context

	// cancel is the manager context cancellation function
	cancel context.CancelFunc

	// mu synchronizes accesses to the manager
	mu sync.Mutex

	// closeOnce guards the call to Close()
	closeOnce sync.Once
}

func NewManager(logger log.Logger, prompter *component.Prompter, altView AltViewFunc, updateFunc TransferUpdateFunc) *Manager {

	// Save the initial terminal state
	state, err := term.GetState(int(os.Stdin.Fd()))

	// Should never fail as tea app exits early if stdin is not a terminal
	if err != nil {
		panic(err)
	}

	// Create a context for the manager
	ctx, cancel := context.WithCancel(context.Background())

	mgr := &Manager{
		logger:         logger,
		sessions:       make(map[uint64]*session.Session),
		activeSession:  0,
		state:          state,
		altView:        altView,
		transferUpdate: updateFunc,
		verifiedCerts:  make(map[string]bool),
		prompter:       prompter,
		ctx:            ctx,
		cancel:         cancel,
	}

	// Try to load identity from the default location
	_ = mgr.LoadIdentity(".gonduit/app.pem")

	return mgr

}

func (m *Manager) ActiveSession() (*session.Session, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.activeSession != 0 {
		return m.sessions[m.activeSession], nil
	}
	return nil, ErrNoActiveSession
}

func (m *Manager) Sessions() map[uint64]platform.HostInfo {
	m.mu.Lock()
	defer m.mu.Unlock()
	info := make(map[uint64]platform.HostInfo, len(m.sessions))

	for id, s := range m.sessions {
		info[id] = s.HostInfo()
	}

	return info
}

func (m *Manager) NumSessions() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.sessions)
}

func (m *Manager) Close() {

	m.closeOnce.Do(func() {

		m.cancel()

		sessions := make(map[uint64]*session.Session, len(m.sessions))

		for id, sess := range m.sessions {
			sessions[id] = sess
		}

		// Close sessions
		for id, sess := range sessions {
			if err := sess.Close(); err != nil {
				m.logger.Errorf("failed to close session %d: %v", id, err)
			}
		}

		if m.listener != nil {
			_ = m.listener.Close()
		}

		m.activeSession = 0
		m.sessions = nil
		m.logger = nil
		m.listener = nil

	})

}

func (m *Manager) UseSession(id uint64) error {

	m.mu.Lock()
	defer m.mu.Unlock()

	if m.activeSession == id {
		return fmt.Errorf("session %d is already active", id)
	}

	_, ok := m.sessions[id]
	if !ok {
		return fmt.Errorf("session %d not found", id)
	}

	m.activeSession = id

	return nil

}

func (m *Manager) CloseSession(id uint64) error {
	m.mu.Lock()
	s, ok := m.sessions[id]
	m.mu.Unlock()

	if !ok {
		return fmt.Errorf("session %d not found", id)
	}

	return s.Close()
}

func (m *Manager) OnAttach(_ *session.Session, value bool) {
	m.altView(value)
}

func (m *Manager) UpdateTransfer(filename string, current, total uint64) {
	m.transferUpdate(filename, current, total)
}

func (m *Manager) OnClosed(s *session.Session, err error) {

	m.mu.Lock()
	defer m.mu.Unlock()

	id := s.ID()

	current, ok := m.sessions[id]

	if !ok || current != s {
		return
	}

	delete(m.sessions, id)

	if m.activeSession == id {
		m.activeSession = 0
	}

	if err != nil {
		m.logger.Warnf("session %d disconnected (%s): %v", id, current.RemoteAddr(), err)
	}

}

func (m *Manager) WindowResized(size util.TerminalSize) {

	s, err := m.ActiveSession()

	if err != nil {
		return
	}

	s.Resize(size)

}
