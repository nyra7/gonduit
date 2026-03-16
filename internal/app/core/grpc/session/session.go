package session

import (
	"app/log"
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net"
	"shared/pkg"
	"shared/platform"
	"shared/proto"
	"shared/util"
	"sync"

	"golang.org/x/term"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

var (
	idGen = util.NewIDGenerator()
)

type Listener interface {
	OnAttach(s *Session, attached bool)
	OnClosed(s *Session, err error)
	UpdateTransfer(filename string, current uint64, total uint64)
}

type Session struct {

	// id is the unique ID of the session
	id uint64

	// logger is the logger for the session
	logger log.Logger

	// addr is the remote address of the session
	addr *net.TCPAddr

	// conn is the gRPC connection to the remote server
	conn *grpc.ClientConn

	// hostInfo is the host info from the server
	hostInfo platform.HostInfo

	// client is the gRPC client for the server
	client proto.GonduitServiceClient

	// stream is the gRPC stream for the shell
	stream proto.GonduitService_ShellStreamClient

	// initialState is the initial terminal state
	initialState *term.State

	// readClosed indicates whether the read loop is active
	readClosed chan struct{}

	// activeShell indicates the index of the shell that is currently being interacted with
	activeShell uint64

	// mu guards request tracking state (pending)
	mu sync.Mutex

	// pending maps request IDs to response channels awaiting replies
	pending map[string]chan *proto.ShellServerMessage

	// messages delivers async, non-response messages from the peer
	messages chan *proto.ShellServerMessage

	// closeOnce ensures shutdown happens once
	closeOnce sync.Once

	// wg tracks background goroutines launched by the session
	wg sync.WaitGroup

	// closed is closed to broadcast shutdown to all waiters
	closed chan struct{}

	// closeErr records the error that triggered shutdown (if any)
	closeErr error

	// listener is the callback listener for the session
	listener Listener
}

// NewSession creates a new session by establishing a gRPC connection to a running server
func NewSession(
	ctx context.Context,
	addr string,
	tlsConfig *tls.Config,
	logger log.Logger,
	state *term.State,
	listener Listener) (*Session, error) {

	tcpAddr, err := net.ResolveTCPAddr("tcp", addr)

	if err != nil {
		return nil, err
	}

	logger.Info(fmt.Sprintf("dialing %s (grpc+tls)...", addr))

	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(credentials.NewTLS(tlsConfig)))

	if err != nil {
		return nil, err
	}

	client := proto.NewGonduitServiceClient(conn)

	session := &Session{
		addr:         tcpAddr,
		conn:         conn,
		client:       client,
		logger:       logger,
		hostInfo:     platform.HostInfo{},
		pending:      make(map[string]chan *proto.ShellServerMessage),
		messages:     make(chan *proto.ShellServerMessage, 128),
		closed:       make(chan struct{}),
		initialState: state,
		listener:     listener,
	}

	err = session.init(ctx, pkg.Version)

	if err != nil {
		_ = session.Close()
		return nil, err
	}

	// Generate a unique ID for the session on success
	session.id = idGen.Next()

	return session, nil

}

func NewSessionFromConn(
	ctx context.Context,
	grpcConn *grpc.ClientConn,
	remoteAddr net.Addr,
	logger log.Logger,
	state *term.State,
	listener Listener,
) (*Session, error) {

	tcpAddr, _ := net.ResolveTCPAddr("tcp", remoteAddr.String())

	client := proto.NewGonduitServiceClient(grpcConn)

	session := &Session{
		addr:         tcpAddr,
		conn:         grpcConn,
		client:       client,
		logger:       logger,
		hostInfo:     platform.HostInfo{},
		pending:      make(map[string]chan *proto.ShellServerMessage),
		messages:     make(chan *proto.ShellServerMessage, 128),
		closed:       make(chan struct{}),
		initialState: state,
		listener:     listener,
	}

	if err := session.init(ctx, pkg.Version); err != nil {
		_ = session.Close()
		return nil, err
	}

	session.id = idGen.Next()

	return session, nil
}

func (s *Session) init(ctx context.Context, clientVersion string) error {

	// Send the initial message
	resp, err := s.client.Hello(ctx, &proto.HelloRequest{ClientVersion: clientVersion})

	// If the connection failed, return the error
	if err != nil {
		return err
	}

	// Save the host info
	s.hostInfo = platform.HostInfoFromProto(resp.HostInfo)

	// Start the shell stream
	stream, err := s.client.ShellStream(context.Background())

	// If the stream failed, return the error
	if err != nil {
		return err
	}

	// Save the stream
	s.stream = stream

	// Increment the wait group
	s.wg.Add(2)

	// Start the input, resize and read loops
	go s.messageLoop()
	go s.readLoop()

	return nil

}

func (s *Session) close(err error) error {

	s.closeOnce.Do(func() {

		// If the shutdown was caused by an error other than ErrClosed or EOF, record it.
		if err != nil && !errors.Is(err, net.ErrClosed) && !errors.Is(err, io.EOF) {
			s.closeErr = fmt.Errorf("connection died: %w", err)
		}

		// Close the read loop
		s.closeRead("session closed")

		// Close the stream
		_ = s.closeStream()

		// Signal shutdown early to unblock read/write paths.
		close(s.closed)

		// Close the messages channel
		close(s.messages)

		// Wait for all goroutines to finish and release resources
		s.wg.Wait()

		// Clear pending requests
		s.pending = nil

		// Close the connection
		_ = s.conn.Close()

		// Call the onClosed callback
		s.listener.OnClosed(s, s.closeErr)

	})

	return s.closeErr

}

func (s *Session) Resize(size util.TerminalSize) {

	if s.activeShell != 0 {
		_ = s.sendTerminalResize(s.activeShell, size)
	}

}

func (s *Session) RemoteAddr() net.Addr {
	return s.addr
}

func (s *Session) Close() error {
	return s.close(nil)
}

func (s *Session) HostInfo() platform.HostInfo {
	return s.hostInfo
}

func (s *Session) ID() uint64 {
	return s.id
}
