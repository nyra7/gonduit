package session

import (
	"app/style"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"shared/platform"
	"shared/proto"
	"shared/util"
	"strings"

	"golang.org/x/term"
)

type TransferResult struct {
	LocalPath  string
	RemotePath string
	Size       uint64
	Checksum   string
}

func (s *Session) DownloadSync(ctx context.Context, localPath, remotePath string, force bool) (*TransferResult, error) {

	// Treat empty localPath as current directory
	if localPath == "" {
		localPath = "."
	}

	// If localPath is a directory, append the remote filename
	if info, err := os.Stat(localPath); err == nil && info.IsDir() {
		localPath = filepath.Join(localPath, filepath.Base(remotePath))
	}

	// Check if local file already exists
	if _, err := os.Stat(localPath); err == nil && !force {
		return nil, fmt.Errorf("file already exists: %s (use --force to overwrite)", localPath)
	}

	stream, err := s.client.DownloadFile(ctx, &proto.TransferRequest{Path: remotePath})

	if err != nil {
		return nil, err
	}

	res, err := stream.Recv()

	if err != nil {
		return nil, err
	}

	header, ok := res.GetChunk().(*proto.TransferChunk_Metadata)

	if !ok {
		return nil, fmt.Errorf("invalid chunk: expected metadata, got %T", header)
	}

	f, err := os.OpenFile(localPath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0o644)

	if err != nil {
		return nil, err
	}

	defer util.CloseFile(f)

	hasher := sha256.New()
	filename := filepath.Base(header.Metadata.Filename)

	var bytesReceived uint64

	for {

		res, err = stream.Recv()

		if err == io.EOF {
			break
		}

		if err != nil {
			return nil, err
		}

		data, dataOk := res.GetChunk().(*proto.TransferChunk_Data)

		if !dataOk {
			return nil, fmt.Errorf("invalid chunk: expected data, got %T", data)
		}

		_, _ = hasher.Write(data.Data)
		n, _ := f.Write(data.Data)

		bytesReceived += uint64(n)

		s.listener.UpdateTransfer(filename, bytesReceived, header.Metadata.Size)

	}

	fileChecksum := hex.EncodeToString(hasher.Sum(nil))

	if fileChecksum != header.Metadata.Checksum {
		return nil, fmt.Errorf("invalid checksum: expected %s, got %s", header.Metadata.Checksum, fileChecksum)
	}

	return &TransferResult{
		LocalPath:  localPath,
		RemotePath: remotePath,
		Size:       bytesReceived,
		Checksum:   fileChecksum,
	}, nil

}

func (s *Session) UploadSync(ctx context.Context, localPath, remotePath string, force bool) (*TransferResult, error) {

	// If remotePath looks like a directory (trailing slash or empty), append local filename
	if remotePath == "" || strings.Contains(remotePath, "/") || strings.Contains(remotePath, "\\") {
		remotePath = path.Join(remotePath, filepath.Base(localPath))
	}

	file, err := os.Open(localPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer util.CloseFile(file)

	stat, err := file.Stat()
	if err != nil {
		return nil, fmt.Errorf("failed to stat file: %w", err)
	}

	if stat.IsDir() {
		return nil, fmt.Errorf("cannot upload directory: %s", localPath)
	}

	s.logger.Info("computing checksum...")

	fileChecksum, err := util.CalculateFileChecksum(file)

	if err != nil {
		return nil, fmt.Errorf("failed to calculate checksum: %w", err)
	}

	if _, err = file.Seek(0, 0); err != nil {
		return nil, fmt.Errorf("failed to reset file pointer: %w", err)
	}

	stream, err := s.client.UploadFile(ctx)

	if err != nil {
		return nil, fmt.Errorf("failed to start upload stream: %w", err)
	}

	err = stream.Send(&proto.TransferChunk{
		Chunk: &proto.TransferChunk_Metadata{
			Metadata: &proto.TransferMetadata{
				Filename: remotePath,
				Size:     uint64(stat.Size()),
				Checksum: fileChecksum,
				Force:    force,
			},
		},
	})

	if err != nil {
		return nil, fmt.Errorf("failed to send metadata: %v", err)
	}

	var n int
	var totalSent uint64
	buf := make([]byte, 1024*1024)
	filename := filepath.Base(localPath)

	for {

		n, err = file.Read(buf)

		if n > 0 {

			err = stream.Send(&proto.TransferChunk{
				Chunk: &proto.TransferChunk_Data{
					Data: buf[:n],
				},
			})

			if err != nil {
				return nil, stream.RecvMsg(nil)
			}

		}

		if err == io.EOF {
			break
		}

		if err != nil {
			return nil, fmt.Errorf("failed to read file: %v", err)
		}

		totalSent += uint64(n)

		s.listener.UpdateTransfer(filename, totalSent, uint64(stat.Size()))

	}

	_ = stream.CloseSend()
	var res proto.TransferResult
	err = stream.RecvMsg(&res)

	if err != nil {
		return nil, err
	}

	if !res.Success {
		return nil, fmt.Errorf("upload did not succeed: %v", res.Error)
	}

	return &TransferResult{
		LocalPath:  localPath,
		RemotePath: remotePath,
		Size:       uint64(stat.Size()),
		Checksum:   fileChecksum,
	}, nil

}

func (s *Session) ListShells(ctx context.Context) ([]string, error) {

	r, err := s.sendRequestWithContext(ctx, &proto.ShellClientMessage{
		Message: &proto.ShellClientMessage_ListTypes{ListTypes: &proto.ListShellTypes{}},
	})

	if err != nil {
		return nil, err
	}

	res, err := expect[*proto.ShellServerMessage_Types](r.Message)

	if err != nil {
		return nil, err
	}

	return res.Types.Names, nil

}

func (s *Session) ListSessions(ctx context.Context) (*proto.ShellSessions, error) {

	r, err := s.sendRequestWithContext(ctx, &proto.ShellClientMessage{
		Message: &proto.ShellClientMessage_ListSessions{ListSessions: &proto.ListShellSessions{}},
	})

	if err != nil {
		return nil, err
	}

	res, err := expect[*proto.ShellServerMessage_Sessions](r.Message)

	if err != nil {
		return nil, err
	}

	return res.Sessions, nil

}

func (s *Session) DumpShellHistory(ctx context.Context, id uint64) (*proto.ShellDumpData, error) {

	r, err := s.sendRequestWithContext(ctx, &proto.ShellClientMessage{
		Message: &proto.ShellClientMessage_Dump{Dump: &proto.ShellDump{ShellId: id}},
	})

	if err != nil {
		return nil, err
	}

	res, err := expect[*proto.ShellServerMessage_Dump](r.Message)

	if err != nil {
		return nil, err
	}

	return res.Dump, nil

}

func (s *Session) CreateShell(ctx context.Context, path string, args []string) (uint64, error) {

	create := &proto.ShellCreate{
		Path: path,
		Args: args,
		Size: s.termSize().ToProto(),
	}

	id, err := s.sendCreateShell(ctx, create)

	if err != nil {
		return 0, fmt.Errorf("failed to spawn shell: %w", err)
	}

	return id, s.AttachShell(ctx, id)

}

func (s *Session) KillShell(ctx context.Context, shellID uint64) error {

	r, err := s.sendRequestWithContext(ctx, &proto.ShellClientMessage{
		Message: &proto.ShellClientMessage_Kill{Kill: &proto.ShellKill{ShellId: shellID}},
	})

	if err != nil {
		return err
	}

	_, err = expect[*proto.ShellServerMessage_Ok](r.Message)

	if err != nil {
		return err
	}

	return nil

}

func (s *Session) AttachShell(ctx context.Context, id uint64) error {

	if s.activeShell != 0 {
		if err := s.DetachShell(ctx, s.activeShell); err != nil {
			return err
		}
	}

	s.activeShell = id

	history, err := s.sendAttachShell(ctx, id, s.termSize())

	if err != nil {
		s.activeShell = 0
		return fmt.Errorf("failed to attach to shell: %w", err)
	}

	// Call the onAttach callback
	s.listener.OnAttach(s, true)

	_, err = term.MakeRaw(int(os.Stdin.Fd()))

	if err != nil {
		s.activeShell = 0
		s.listener.OnAttach(s, false)
		return fmt.Errorf("failed to enable raw mode: %w", err)
	}

	util.ClearTerminal()

	_, _ = fmt.Fprintf(os.Stdout, "%s %s %s\r\n",
		style.SuccessLabel.Render("✓ Connected to"),
		style.Header.Render(fmt.Sprintf("shell %d", id)),
		style.Muted.Render(fmt.Sprintf("(%s to detach)", platform.DetachKeyStr)))

	_, _ = os.Stdout.Write(history)

	s.wg.Add(1)
	go s.inputLoop()

	return nil

}

func (s *Session) DetachShell(ctx context.Context, id uint64) error {

	err := s.sendDetachShell(ctx, id)

	if err != nil {
		return fmt.Errorf("failed to detach shell: %w", err)
	}

	if s.activeShell == id {
		s.closeRead("")
	}

	return nil

}

func (s *Session) sendCreateShell(ctx context.Context, req *proto.ShellCreate) (uint64, error) {

	r, err := s.sendRequestWithContext(ctx, &proto.ShellClientMessage{
		Message: &proto.ShellClientMessage_Create{Create: req},
	})

	if err != nil {
		return 0, err
	}

	res, err := expect[*proto.ShellServerMessage_Ok](r.Message)

	if err != nil {
		return 0, err
	}

	return res.Ok.ShellId, nil

}

func (s *Session) sendAttachShell(ctx context.Context, id uint64, size util.TerminalSize) ([]byte, error) {

	r, err := s.sendRequestWithContext(ctx, &proto.ShellClientMessage{
		Message: &proto.ShellClientMessage_Attach{Attach: &proto.ShellAttach{ShellId: id, Size: size.ToProto()}},
	})

	if err != nil {
		return nil, err
	}

	res, err := expect[*proto.ShellServerMessage_Attached](r.Message)

	if err != nil {
		return nil, err
	}

	return res.Attached.Data, nil

}

func (s *Session) sendDetachShell(ctx context.Context, id uint64) error {

	r, err := s.sendRequestWithContext(ctx, &proto.ShellClientMessage{
		Message: &proto.ShellClientMessage_Detach{Detach: &proto.ShellDetach{ShellId: id}},
	})

	if err != nil {
		return err
	}

	_, err = expect[*proto.ShellServerMessage_Ok](r.Message)

	if err != nil {
		return err
	}

	return nil

}

func (s *Session) sendShellInput(id uint64, data []byte) error {
	return s.stream.Send(&proto.ShellClientMessage{
		Message: &proto.ShellClientMessage_Stdin{Stdin: &proto.ShellStdin{ShellId: id, Data: data}},
	})
}

func (s *Session) sendTerminalResize(id uint64, size util.TerminalSize) error {
	return s.stream.Send(&proto.ShellClientMessage{
		Message: &proto.ShellClientMessage_Resize{Resize: &proto.ShellResize{ShellId: id, Size: size.ToProto()}},
	})
}
