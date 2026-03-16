package grpc

import (
	"app/core/grpc/session"
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"net"
	"shared/crypto"
	"shared/util"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

func (m *Manager) Listen(addr string, port int, acceptAddr string, insecure bool) error {

	m.mu.Lock()
	defer m.mu.Unlock()

	if m.listener != nil {
		return fmt.Errorf("already listening on %s", m.listener.Addr())
	}

	bind := fmt.Sprintf("%s:%d", addr, port)

	tlsConfig, err := m.getTLSConfig()

	if err != nil {
		return err
	}

	tlsConfig = crypto.WithVerification(tlsConfig, func(cert *x509.Certificate, fingerprint string, err error) error {

		if insecure || m.selfSigned {
			return nil
		}

		return err

	})

	listener, err := tls.Listen("tcp", bind, tlsConfig)

	if err != nil {
		return err
	}

	filter, err := util.NewFilterListener(listener, acceptAddr, m.connectionRejected)

	if err != nil {
		_ = listener.Close()
		return fmt.Errorf("error creating filter: %v", err)
	}

	m.listener = filter

	go m.listenLoop(tlsConfig)

	return nil
}

func (m *Manager) connectionRejected(_ net.Conn, err error) {
	m.logger.Errorf("rejecting server connection: %v", err)
}

func (m *Manager) listenLoop(config *tls.Config) {

	for {
		conn, err := m.listener.Accept()

		if err != nil {
			select {
			case <-m.ctx.Done():
				return
			default:
			}

			if errors.Is(err, net.ErrClosed) {
				return
			}

			m.logger.Errorf("failed to accept connection: %v", err)
			continue
		}

		go m.accept(conn, config)

	}

}

func (m *Manager) StopListening() error {

	m.mu.Lock()
	defer m.mu.Unlock()

	if m.listener == nil {
		return fmt.Errorf("not listening")
	}

	_ = m.listener.Close()
	m.listener = nil

	return nil

}

// accept takes an inbound TCP connection and establishes a gRPC session on it
func (m *Manager) accept(conn net.Conn, config *tls.Config) {

	// Instead of opening a new TCP connection, hand gRPC the conn that already dialed us
	opts := []grpc.DialOption{
		grpc.WithTransportCredentials(credentials.NewTLS(config)),
		grpc.WithContextDialer(func(ctx context.Context, _ string) (net.Conn, error) { return conn, nil }),
	}

	grpcConn, err := grpc.NewClient("passthrough://", opts...)

	if err != nil {
		m.logger.Errorf("could not create grpc client for inbound conn %s: %v", conn.RemoteAddr(), err)
		_ = conn.Close()
		return
	}

	sess, err := session.NewSessionFromConn(m.ctx, grpcConn, conn.RemoteAddr(), m.logger, m.state, m)

	if err != nil {
		m.logger.Errorf("connection refused for %s: %v", conn.RemoteAddr(), util.ParseGrpcError(err))
		_ = grpcConn.Close()
		_ = conn.Close()
		return
	}

	m.registerSession(sess)

}
