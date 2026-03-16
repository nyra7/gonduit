package grpc

import (
	"app/core/grpc/session"
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"shared/crypto"
	"shared/util"
	"strings"
	"time"
)

// Connect establishes a gRPC connection to the server
func (m *Manager) Connect(ctx context.Context, addr string, port int) error {

	tlsConfig, err := m.getTLSConfig()

	if err != nil {
		return err
	}

	// Create a new session
	sess, err := session.NewSession(
		ctx,
		fmt.Sprintf("%s:%d", addr, port),
		crypto.WithVerification(tlsConfig, m.promptUntrustedConnect),
		m.logger,
		m.state,
		m,
	)

	if err != nil {
		return fmt.Errorf("could not create session: %v", util.ParseGrpcError(err))
	}

	m.registerSession(sess)

	return nil

}

func (m *Manager) registerSession(s *session.Session) {

	id := s.ID()

	m.mu.Lock()

	// Store the new session
	m.sessions[id] = s

	// Set the session as active if none is
	if m.activeSession == 0 {
		m.activeSession = id
	}

	m.mu.Unlock()

	hostInfo := s.HostInfo()

	// Log the session
	m.logger.Success(fmt.Sprintf(
		"session %d opened | %s@%s (%s via %s at %s) → %s",
		id,
		hostInfo.Username,
		hostInfo.Hostname,
		hostInfo.Ip,
		hostInfo.NetworkAdapter,
		hostInfo.WorkingDir,
		hostInfo.OsInfo.String(),
	))

}

func (m *Manager) getTLSConfig() (*tls.Config, error) {

	// Return the config if valid
	if m.tlsConfig != nil {
		return m.tlsConfig, nil
	}

	// Otherwise, automatically generate a self-signed identity
	if err := m.UseSelfSignedIdentity(); err != nil {
		return nil, err
	}

	// Log a warning message to inform user
	m.logger.Warn("no identity loaded, using self-signed certificate for future operations (use 'identity' command to manage)")

	return m.tlsConfig, nil

}

func (m *Manager) promptUntrustedConnect(_ *x509.Certificate, fingerprint string, _ error) error {

	m.verifiedCertsMu.Lock()
	already, exists := m.verifiedCerts[fingerprint]
	if exists {
		m.verifiedCertsMu.Unlock()
		if !already {
			return fmt.Errorf("user rejected certificate")
		}
		return nil
	}

	m.verifiedCerts[fingerprint] = false
	m.verifiedCertsMu.Unlock()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	m.logger.Warn(fmt.Sprintf("accept untrusted server connection? fingerprint: %s", fingerprint))

	s, err := m.prompter.PromptInput(ctx, "Accept? (y/N)", false)

	m.verifiedCertsMu.Lock()
	defer m.verifiedCertsMu.Unlock()

	s = strings.ToLower(s)

	// this is a bit of a hack, the problem is that when the user tries to connect using a host that has multiple ip
	// addresses, the above warning and prompt will trigger multiple times. To avoid this, use a mutex to force the
	// subsequent connection attempts (for next ips, which happens simultaneously after the first) to fail and return
	// immediately if the user denied the first request. then, clean the denied fingerprint entry after 100 ms so the
	// user can be reprompted for validation
	if err != nil || (s != "y" && s != "yes") {
		time.AfterFunc(100*time.Millisecond, func() {
			m.verifiedCertsMu.Lock()
			defer m.verifiedCertsMu.Unlock()
			delete(m.verifiedCerts, fingerprint)
		})
		if err != nil {
			return err
		}
		return fmt.Errorf("user rejected certificate")
	}

	m.verifiedCerts[fingerprint] = true
	return nil

}
