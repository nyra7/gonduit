package core

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"server/log"
	"shared/proto"
	"shared/util"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/stats"
)

func (s *Server) listen() error {

	filter, tlsConfig, err := s.createListener()

	if err != nil {
		return err
	}

	// Create gRPC server
	s.grpcSrv = grpc.NewServer(grpc.Creds(credentials.NewTLS(tlsConfig)), grpc.StatsHandler(&statsHandler{}))

	// Register Gonduit service
	proto.RegisterGonduitServiceServer(s.grpcSrv, s)

	if s.config.ServerIdentity == "" && s.config.Fingerprint == "" {

		message := "no identity or fingerprint specified, server will accept any client connection"
		accept := filter.AcceptString()

		if accept != "" {
			message = fmt.Sprintf("%s within [%s]", message, accept)
		}

		log.Warn(message)

	}

	log.Infof("server listening on %s", filter.Addr())

	// Serve (blocks until shutdown)
	if err = s.grpcSrv.Serve(filter); err != nil {
		return err
	}

	return nil

}

func (s *Server) createListener() (*util.FilterListener, *tls.Config, error) {

	bind := s.config.Bind()

	tlsConfig, err := s.buildTLSConfig()

	if err != nil {
		return nil, nil, err
	}

	listener, err := net.Listen("tcp", bind)

	if err != nil {
		return nil, nil, fmt.Errorf("failed to initialize server: %v", err)
	}

	filter, err := util.NewFilterListener(listener, s.config.AcceptAddr, s.connectionRejected)

	if err != nil {
		_ = listener.Close()
		return nil, nil, fmt.Errorf("error creating filter: %v", err)
	}

	return filter, tlsConfig, nil

}

func (s *Server) connectionRejected(_ net.Conn, err error) {
	log.Error(err)
}

// statsHandler is a simple gRPC stats handler to log connect and disconnect events
type statsHandler struct{}

func (h *statsHandler) TagConn(ctx context.Context, info *stats.ConnTagInfo) context.Context {
	log.Infof("connection established (%s)", info.RemoteAddr)
	return ctx
}

func (h *statsHandler) HandleConn(ctx context.Context, s stats.ConnStats) {

	if _, ok := s.(*stats.ConnEnd); ok {

		p, ctxOk := peer.FromContext(ctx)

		if !ctxOk {
			log.Infof("connection closed (unknown peer)")
		}

		log.Infof("connection closed (%s)", p.Addr)

	}

}

func (h *statsHandler) TagRPC(ctx context.Context, _ *stats.RPCTagInfo) context.Context { return ctx }
func (h *statsHandler) HandleRPC(_ context.Context, _ stats.RPCStats)                   {}
