package shell

import (
	"net"
	"shared/proto"
)

// Stream is a wrapper around the gRPC shell stream
type Stream struct {
	stream proto.GonduitService_ShellStreamServer
	addr   net.Addr
}

func NewStream(stream proto.GonduitService_ShellStreamServer, addr net.Addr) *Stream {
	return &Stream{stream: stream, addr: addr}
}

func (s *Stream) RemoteAddr() net.Addr {
	return s.addr
}

func (s *Stream) SendStdout(id uint64, stdout []byte) error {
	return s.stream.Send(&proto.ShellServerMessage{
		Message: &proto.ShellServerMessage_Stdout{Stdout: &proto.ShellStdout{ShellId: id, Data: stdout}},
	})
}

func (s *Stream) SendAttached(id string, shellId uint64, stdout []byte) error {
	return s.stream.Send(&proto.ShellServerMessage{
		Id:      id,
		Message: &proto.ShellServerMessage_Attached{Attached: &proto.ShellAttached{ShellId: shellId, Data: stdout}},
	})
}

func (s *Stream) SendExited(id uint64, exitCode int32, reason string) error {
	return s.stream.Send(&proto.ShellServerMessage{
		Message: &proto.ShellServerMessage_Exited{Exited: &proto.ShellExited{ShellId: id, ExitCode: exitCode, Reason: reason}},
	})
}

func (s *Stream) SendError(id string, shellId uint64, err error) error {
	return s.stream.Send(&proto.ShellServerMessage{
		Id: id,
		Message: &proto.ShellServerMessage_Error{
			Error: &proto.ShellError{
				Error:   err.Error(),
				ShellId: shellId,
			},
		},
	})
}

func (s *Stream) SendOk(id string, shellId uint64) error {
	return s.stream.Send(&proto.ShellServerMessage{
		Id: id,
		Message: &proto.ShellServerMessage_Ok{
			Ok: &proto.ShellOk{
				ShellId: shellId,
			},
		},
	})
}

func (s *Stream) SendSessionList(id string, list []*proto.ShellSession) error {
	return s.stream.Send(&proto.ShellServerMessage{
		Id: id,
		Message: &proto.ShellServerMessage_Sessions{
			Sessions: &proto.ShellSessions{
				Sessions: list,
			},
		},
	})
}

func (s *Stream) SendDump(id string, session *proto.ShellSession, data []byte) error {
	return s.stream.Send(&proto.ShellServerMessage{
		Id: id,
		Message: &proto.ShellServerMessage_Dump{
			Dump: &proto.ShellDumpData{
				Session: session,
				Data:    data,
			},
		},
	})
}

func (s *Stream) SendShellTypes(id string, names []string) error {
	return s.stream.Send(&proto.ShellServerMessage{
		Id: id,
		Message: &proto.ShellServerMessage_Types{
			Types: &proto.ShellTypes{
				Names: names,
			},
		},
	})
}
