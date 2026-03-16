package core

import (
	"context"
	"os/signal"
	"server/shell"
	"shared/proto"
	"sync"
	"syscall"

	"google.golang.org/grpc"
)

type Server struct {
	proto.UnimplementedGonduitServiceServer

	// config Stores the configuration for the server
	config Config

	// ctx is the server context used for graceful shutdown
	ctx context.Context

	// stop is the server context cancellation function
	stop context.CancelFunc

	// mgr is the server shell manager
	mgr *shell.Manager

	// grpcSrv is the gRPC server
	grpcSrv *grpc.Server

	// activeStreams is a map that keeps track of active shell streams, keyed by peer address (map[string]bool)
	activeStreams sync.Map
}

func NewServer(config Config) *Server {

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)

	return &Server{
		config: config,
		ctx:    ctx,
		stop:   cancel,
		mgr:    shell.NewManager(),
	}

}

// Serve starts the server
func (s *Server) Serve() error {

	defer s.stop()
	go s.waitForShutdown()

	var err error

	if s.config.Reverse {
		err = s.reverse()
	} else {
		err = s.listen()
	}

	return err

}

func (s *Server) waitForShutdown() {
	<-s.ctx.Done()
	if s.grpcSrv != nil {
		s.grpcSrv.Stop()
	}
}
