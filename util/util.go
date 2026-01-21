package util

import (
	"errors"
	"log"
	"net"
	"shells/style"
)

func WriteConn(conn net.Conn, data string) {
	_, _ = conn.Write([]byte(data))
}

func WriteSuccess(conn net.Conn, message string) {
	WriteConn(conn, style.Green.Apply(message))
}

func WriteError(conn net.Conn, message string) {
	WriteConn(conn, style.Red.Apply(message))
}

func WriteWarning(conn net.Conn, message string) {
	WriteConn(conn, style.Yellow.Apply(message))
}

func WriteInfo(conn net.Conn, message string) {
	WriteConn(conn, style.Cyan.Apply(message))
}

func CloseConn(conn net.Conn) {
	err := conn.Close()
	if err != nil && !errors.Is(err, net.ErrClosed) {
		log.Printf("Failed to close connection %v: %v\n", conn.RemoteAddr(), err)
	}
}
