package core

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"server/log"
	"shared/proto"
	"sync"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

func (s *Server) reverse() error {

	// Try to build the TLS config
	tlsConfig, err := s.buildTLSConfig()

	// Return the error on failure
	if err != nil {
		return fmt.Errorf("could not load server identity bundle: %v", err)
	}

	log.Infof("dialing %s (grpc+tls)...", s.config.Bind())

	// Try to connect to the client with a 5s timeout
	conn, err := tls.DialWithDialer(&net.Dialer{Timeout: 5 * time.Second}, "tcp", s.config.Bind(), tlsConfig)

	if err != nil {
		return err
	}

	// Create the gRPC server with our reverse credentials
	s.grpcSrv = grpc.NewServer(grpc.Creds(&reverseCredentials{server: s, inner: credentials.NewTLS(tlsConfig)}))

	// Register the service
	proto.RegisterGonduitServiceServer(s.grpcSrv, s)

	// Start the gRPC server
	return s.grpcSrv.Serve(newConnListener(conn))

}

// reverseCredentials is a credentials.TransportCredentials wrapper used to intercept ServerHandshake calls and
// handle errors to close the grpc server early in reverse mode, preventing it from running forever
type reverseCredentials struct {
	inner  credentials.TransportCredentials
	server *Server
}

func (w *reverseCredentials) ServerHandshake(rawConn net.Conn) (net.Conn, credentials.AuthInfo, error) {
	conn, info, err := w.inner.ServerHandshake(rawConn)
	if err != nil {
		log.Errorf("error during handshake: %v", err)
		w.server.stop()
		return nil, nil, err
	}
	return newReverseConn(conn, w.server), info, nil
}

func (w *reverseCredentials) ClientHandshake(ctx context.Context, s string, rawConn net.Conn) (net.Conn, credentials.AuthInfo, error) {
	return w.inner.ClientHandshake(ctx, s, rawConn)
}

func (w *reverseCredentials) Clone() credentials.TransportCredentials {
	return &reverseCredentials{inner: w.inner.Clone(), server: w.server}
}

func (w *reverseCredentials) Info() credentials.ProtocolInfo {
	return w.inner.Info()
}
func (w *reverseCredentials) OverrideServerName(s string) error {
	return w.inner.OverrideServerName(s)
}

// reverseListener wraps a single outbound conn as a net.Listener to use in reverse mode
type reverseListener struct {
	conn net.Conn
	ch   chan net.Conn
	once sync.Once
}

func newConnListener(conn net.Conn) net.Listener {
	l := &reverseListener{
		conn: conn,
		ch:   make(chan net.Conn, 1),
	}
	l.ch <- conn
	return l
}

func (l *reverseListener) Accept() (net.Conn, error) {
	conn, ok := <-l.ch
	if !ok {
		return nil, errors.New("listener closed")
	}
	return conn, nil
}

func (l *reverseListener) Close() error {
	l.once.Do(func() { close(l.ch) })
	return l.conn.Close()
}

func (l *reverseListener) Addr() net.Addr {
	return l.conn.RemoteAddr()
}

// reverseConn wraps an established net.Conn to catch its Close call
// when the client connection is done in reverse mode
type reverseConn struct {
	conn      net.Conn
	server    *Server
	closeOnce sync.Once
	closeErr  error
}

func newReverseConn(c net.Conn, s *Server) *reverseConn {
	return &reverseConn{conn: c, server: s}
}

func (w *reverseConn) Close() error {

	w.closeOnce.Do(func() {

		// Stop the server
		defer w.server.stop()

		log.Infof("client connection closed. exiting")

		// Close the connection
		w.closeErr = w.conn.Close()

	})

	return w.closeErr

}

func (w *reverseConn) Read(b []byte) (int, error)         { return w.conn.Read(b) }
func (w *reverseConn) Write(b []byte) (int, error)        { return w.conn.Write(b) }
func (w *reverseConn) LocalAddr() net.Addr                { return w.conn.LocalAddr() }
func (w *reverseConn) RemoteAddr() net.Addr               { return w.conn.RemoteAddr() }
func (w *reverseConn) SetDeadline(t time.Time) error      { return w.conn.SetDeadline(t) }
func (w *reverseConn) SetReadDeadline(t time.Time) error  { return w.conn.SetReadDeadline(t) }
func (w *reverseConn) SetWriteDeadline(t time.Time) error { return w.conn.SetWriteDeadline(t) }
