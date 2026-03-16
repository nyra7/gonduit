package core

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"server/log"
	"server/shell"
	"shared/pkg"
	"shared/platform"
	"shared/proto"
	"shared/util"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// Hello implements the Hello RPC
func (s *Server) Hello(ctx context.Context, req *proto.HelloRequest) (*proto.HelloResponse, error) {

	p, ok := peer.FromContext(ctx)

	if !ok {
		log.Errorf("failed to get peer from context")
		return nil, status.Errorf(codes.Internal, "failed to get peer from context")
	}

	// Ensure the client version matches the server version
	if req.ClientVersion != pkg.Version {
		log.Errorf("refusing connection for client with mismatched version %s", req.ClientVersion)
		return nil, status.Errorf(codes.InvalidArgument, "version mismatch: %s != %s", req.ClientVersion, pkg.Version)
	}

	// Get host info from server
	hostInfo := platform.GetHostInfo(p.LocalAddr)

	return &proto.HelloResponse{
		HostInfo:      hostInfo.ToProto(),
		ServerVersion: pkg.Version,
		Features:      []string{"shell", "file-transfer"},
	}, nil

}

// ShellStream implements bidirectional shell streaming
func (s *Server) ShellStream(stream proto.GonduitService_ShellStreamServer) error {

	p, ok := peer.FromContext(stream.Context())

	if !ok {
		return status.Errorf(codes.Internal, "failed to get peer from context")
	}

	addr := p.Addr.String()

	// Enforce single stream per client at a time
	if _, exists := s.activeStreams.LoadOrStore(addr, true); exists {
		log.Errorf("client %s already has an active stream", addr)
		return status.Errorf(codes.ResourceExhausted, "stream already active for this client")
	}

	// Clear the stream reference on return
	defer s.activeStreams.Delete(addr)

	shellStream := shell.NewStream(stream, p.Addr)

	// Cleanup resources and log on stream close
	go func() {
		<-stream.Context().Done()
		s.mgr.DetachStream(shellStream)
	}()

	log.Infof("shell stream created for %s", addr)

	for {

		// Read next message and handle errors
		msg, err := stream.Recv()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return status.Errorf(codes.Internal, "failed to receive: %v", err)
		}

		// Route message to appropriate handler
		s.routeMessage(msg, shellStream)

	}
}

// UploadFile implements file upload streaming
func (s *Server) UploadFile(stream proto.GonduitService_UploadFileServer) error {

	// First message must be metadata
	msg, err := stream.Recv()

	if err != nil {
		return status.Errorf(codes.InvalidArgument, "failed to receive metadata: %v", err)
	}

	metadata := msg.GetMetadata()
	if metadata == nil {
		return status.Errorf(codes.InvalidArgument, "first message must contain metadata")
	}

	log.Infof("starting file upload: %s (%d bytes)", metadata.Filename, metadata.Size)

	// Determine the actual file path
	destPath := metadata.Filename

	// Check if the destination is a directory
	if info, err2 := os.Stat(destPath); err2 == nil && info.IsDir() {
		destPath = filepath.Join(destPath, filepath.Base(metadata.Filename))
	}

	// Check if file already exists
	if _, err = os.Stat(destPath); err == nil {
		if !metadata.Force {
			return status.Errorf(codes.AlreadyExists, "file already exists: %s (use --force to overwrite)", destPath)
		}
		log.Infof("overwriting existing file: %s", destPath)
	}

	// Create the output file
	file, err := os.Create(destPath)
	if err != nil {
		return status.Errorf(codes.Internal, "failed to create file: %v", err)
	}
	defer util.CloseFile(file)

	hasher := sha256.New()
	var bytesReceived uint64

	// Receive chunks
	for {
		msg, err = stream.Recv()

		if err == io.EOF {
			break
		}

		if err != nil {
			return status.Errorf(codes.Internal, "failed to receive chunk: %v", err)
		}

		data := msg.GetData()
		if data == nil {
			continue
		}

		n, wErr := file.Write(data)
		if wErr != nil {
			return status.Errorf(codes.Internal, "failed to write chunk: %v", wErr)
		}

		hasher.Write(data)
		bytesReceived += uint64(n)

	}

	// Verify checksum
	actualChecksum := hex.EncodeToString(hasher.Sum(nil))
	if actualChecksum != metadata.Checksum {
		return status.Errorf(codes.DataLoss, "checksum mismatch: expected %s, got %s", metadata.Checksum, actualChecksum)
	}

	log.Infof("file upload completed: %s (%d bytes)", destPath, bytesReceived)

	// Send result
	return stream.SendAndClose(&proto.TransferResult{
		Success:          true,
		BytesTransferred: bytesReceived,
	})

}

// DownloadFile implements file download streaming
func (s *Server) DownloadFile(req *proto.TransferRequest, stream proto.GonduitService_DownloadFileServer) error {

	log.Infof("starting file download: %s", req.Path)

	file, err := os.Open(req.Path)
	if err != nil {
		return status.Errorf(codes.NotFound, "file not found: %v", err)
	}
	defer util.CloseFile(file)

	stat, err := file.Stat()
	if err != nil {
		return status.Errorf(codes.Internal, "failed to stat file: %v", err)
	}

	if stat.IsDir() {
		return status.Errorf(codes.InvalidArgument, "cannot download directory: %s", req.Path)
	}

	// Calculate checksum
	hasher := sha256.New()
	if _, err = io.Copy(hasher, file); err != nil {
		return status.Errorf(codes.Internal, "failed to calculate checksum: %v", err)
	}
	fileChecksum := hex.EncodeToString(hasher.Sum(nil))

	// Reset file pointer
	if _, err = file.Seek(0, 0); err != nil {
		return status.Errorf(codes.Internal, "failed to reset file pointer: %v", err)
	}

	// Send metadata
	err = stream.Send(&proto.TransferChunk{
		Chunk: &proto.TransferChunk_Metadata{
			Metadata: &proto.TransferMetadata{
				Filename: filepath.Base(req.Path),
				Size:     uint64(stat.Size()),
				Checksum: fileChecksum,
			},
		},
	})

	if err != nil {
		return status.Errorf(codes.Internal, "failed to send metadata: %v", err)
	}

	var n int
	// Send chunks
	buf := make([]byte, 1024*1024)
	for {
		n, err = file.Read(buf)
		if n > 0 {
			err = stream.Send(&proto.TransferChunk{
				Chunk: &proto.TransferChunk_Data{
					Data: buf[:n],
				},
			})
			if err != nil {
				return status.Errorf(codes.Internal, "failed to send chunk: %v", err)
			}
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return status.Errorf(codes.Internal, "failed to read file: %v", err)
		}
	}

	log.Infof("file download completed: %s (%d bytes)", req.Path, stat.Size())

	return nil

}

func (s *Server) routeMessage(msg *proto.ShellClientMessage, shellStream *shell.Stream) {

	// Handle different message types
	switch m := msg.Message.(type) {

	case *proto.ShellClientMessage_Create:

		id, err := s.createShell(m)

		if err != nil {
			log.Error(err)
			_ = shellStream.SendError(msg.Id, 0, err)
			return
		}

		_ = shellStream.SendOk(msg.Id, id)

	case *proto.ShellClientMessage_Kill:

		err := s.mgr.KillShell(m.Kill.ShellId)

		if err != nil {
			log.Errorf("failed to kill shell: %v", err)
			_ = shellStream.SendError(msg.Id, m.Kill.ShellId, err)
			return
		}

		_ = shellStream.SendOk(msg.Id, m.Kill.ShellId)

	case *proto.ShellClientMessage_Attach:

		history, mErr := s.attachShell(m, shellStream)

		if mErr != nil {
			log.Error(mErr)
			_ = shellStream.SendError(msg.Id, 0, mErr)
			return
		}

		_ = shellStream.SendAttached(msg.Id, m.Attach.ShellId, history)

	case *proto.ShellClientMessage_Detach:

		s.mgr.DetachStream(shellStream)
		_ = shellStream.SendOk(msg.Id, m.Detach.ShellId)

	case *proto.ShellClientMessage_ListSessions:

		sessions := make([]*proto.ShellSession, 0)

		for _, sh := range s.mgr.All() {
			sessions = append(sessions, &proto.ShellSession{
				Id:         sh.ID(),
				Path:       sh.Path(),
				IsAttached: sh.IsAttached(),
				CreatedAt:  timestamppb.New(time.Now()),
			})
		}

		_ = shellStream.SendSessionList(msg.Id, sessions)

	case *proto.ShellClientMessage_Dump:

		sh, err := s.mgr.ShellByID(m.Dump.ShellId)

		if err != nil {
			log.Errorf("failed to get shell: %v", err)
			_ = shellStream.SendError(msg.Id, m.Dump.ShellId, err)
			return
		}

		session := &proto.ShellSession{
			Id:         sh.ID(),
			Path:       sh.Path(),
			IsAttached: sh.IsAttached(),
			CreatedAt:  timestamppb.New(time.Now()),
		}

		_ = shellStream.SendDump(msg.Id, session, sh.History())

	case *proto.ShellClientMessage_ListTypes:

		shells, err := util.GetValidShells()

		if err != nil {
			log.Errorf("failed to list shells: %v", err)
			_ = shellStream.SendError(msg.Id, 0, err)
			return
		}

		_ = shellStream.SendShellTypes(msg.Id, shells)

	case *proto.ShellClientMessage_Stdin:

		sh, err := s.mgr.ShellByID(m.Stdin.ShellId)

		if err != nil {
			log.Errorf("failed to get shell: %v", err)
			_ = shellStream.SendError(msg.Id, m.Stdin.ShellId, err)
			return
		}

		sh.Write(m.Stdin.Data)

	case *proto.ShellClientMessage_Resize:

		sh, err := s.mgr.ShellByID(m.Resize.ShellId)

		if err != nil {
			log.Errorf("failed to get shell: %v", err)
			return
		}

		sh.Resize(util.TerminalSizeFromProto(m.Resize.Size))

	}

}

func (s *Server) createShell(m *proto.ShellClientMessage_Create) (uint64, error) {

	var path string
	var err error

	if m.Create.Path == "" {
		path, err = util.FindBestShell()
	} else {
		path, err = util.FindShell(m.Create.Path)
	}

	if err != nil {
		return 0, fmt.Errorf("failed to find valid shell: %v", err)
	}

	sh := shell.NewShell(s.ctx, path)

	if err = sh.Run(m.Create.Args...); err != nil {
		return 0, fmt.Errorf("failed to run shell: %v", err)
	}

	s.mgr.RegisterShell(sh)

	return sh.ID(), nil

}

func (s *Server) attachShell(m *proto.ShellClientMessage_Attach, stream *shell.Stream) ([]byte, error) {

	sh, err := s.mgr.ShellByID(m.Attach.ShellId)

	if err != nil {
		return nil, fmt.Errorf("failed to get shell: %v", err)
	}

	history, err := s.mgr.Attach(stream, sh)

	if err != nil {
		return nil, fmt.Errorf("failed to attach shell: %v", err)
	}

	sh.Resize(util.TerminalSizeFromProto(m.Attach.Size))

	return history, nil

}
