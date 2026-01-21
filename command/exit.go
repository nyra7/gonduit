package command

import (
	"io"
	"net"
)

func exit(_ net.Conn, args []string) (string, error) {
	return "", io.EOF
}
