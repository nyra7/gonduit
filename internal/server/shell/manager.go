package shell

import (
	"errors"
	"fmt"
	"server/log"
	"server/shell/pty"
	"sync"
)

type Manager struct {
	mu          sync.Mutex
	shells      map[uint64]*Shell
	attachments map[*Stream]*Shell
}

func NewManager() *Manager {
	return &Manager{
		shells:      make(map[uint64]*Shell),
		attachments: make(map[*Stream]*Shell),
	}
}

func (m *Manager) RegisterShell(sh *Shell) {
	m.mu.Lock()
	m.shells[sh.ID()] = sh
	m.mu.Unlock()

	go m.awaitShell(sh)
}

func (m *Manager) ShellByID(id uint64) (*Shell, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	sh, ok := m.shells[id]

	if !ok {
		return nil, fmt.Errorf("no running shell with id %d", id)
	}

	return sh, nil
}

func (m *Manager) Attach(s *Stream, sh *Shell) ([]byte, error) {
	var prev *Shell

	m.mu.Lock()
	if existing, ok := m.attachments[s]; ok {

		if existing == sh {
			log.Warnf("client %s already attached to shell %d", s.RemoteAddr(), sh.ID())
			m.mu.Unlock()
			return nil, nil
		}

		prev = existing
		delete(m.attachments, s)

	}
	m.mu.Unlock()

	if prev != nil {
		prev.Detach()
	}

	history, err := sh.Attach(s)

	if err != nil {
		return nil, err
	}

	m.mu.Lock()
	m.attachments[s] = sh
	m.mu.Unlock()

	return history, nil
}

func (m *Manager) DetachStream(s *Stream) {
	var sh *Shell

	m.mu.Lock()
	if attached, ok := m.attachments[s]; ok {
		sh = attached
		delete(m.attachments, s)
	}
	m.mu.Unlock()

	if sh != nil {
		sh.Detach()
	}

}

func (m *Manager) KillShell(id uint64) error {

	m.mu.Lock()
	defer m.mu.Unlock()

	if sh, ok := m.shells[id]; ok {
		sh.Close()
		delete(m.shells, id)
		return nil
	}

	return fmt.Errorf("no running shell with id %d", id)

}

func (m *Manager) All() []*Shell {
	m.mu.Lock()
	defer m.mu.Unlock()
	var shs []*Shell
	for _, sh := range m.shells {
		shs = append(shs, sh)
	}
	return shs
}

func (m *Manager) awaitShell(sh *Shell) {
	err := <-sh.WaitChannel()

	var code = -1
	var exitErr *pty.ExitError
	var target *Stream
	var reason string

	if errors.As(err, &exitErr) {
		code = exitErr.ExitCode
		err = nil
	}

	if err != nil {
		log.Warnf("shell %d died: %v", sh.ID(), err)
	}

	m.mu.Lock()

	delete(m.shells, sh.ID())

	for stream, test := range m.attachments {

		if sh != test {
			continue
		}

		delete(m.attachments, stream)
		target = stream
		break

	}

	m.mu.Unlock()

	if target != nil {

		if err != nil {
			reason = err.Error()
		}

		_ = target.SendExited(sh.ID(), int32(code), reason)

	}

}
