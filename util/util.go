package util

import (
	"errors"
	"fmt"
	"gonduit/style"
	"log"
	"net"
	"unicode"
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

func SplitBash(s string) ([]string, error) {
	const (
		stateNone = iota
		stateWord
		stateSingleQuote
		stateDoubleQuote
	)

	var (
		out   []string
		buf   []rune
		state = stateNone
		esc   bool
	)

	for _, r := range s {
		switch state {

		case stateNone, stateWord:
			if esc {
				buf = append(buf, r)
				esc = false
				state = stateWord
				continue
			}

			switch {
			case r == '\\':
				esc = true
				state = stateWord

			case r == '\'':
				state = stateSingleQuote

			case r == '"':
				state = stateDoubleQuote

			case unicode.IsSpace(r):
				if len(buf) > 0 {
					out = append(out, string(buf))
					buf = buf[:0]
				}
				state = stateNone

			default:
				buf = append(buf, r)
				state = stateWord
			}

		case stateSingleQuote:
			if r == '\'' {
				state = stateWord
			} else {
				buf = append(buf, r)
			}

		case stateDoubleQuote:
			if esc {
				buf = append(buf, r)
				esc = false
				continue
			}

			switch r {
			case '\\':
				esc = true
			case '"':
				state = stateWord
			default:
				buf = append(buf, r)
			}
		}
	}

	if esc {
		return nil, fmt.Errorf("unfinished escape at end of input")
	}
	if state == stateSingleQuote {
		return nil, fmt.Errorf("unterminated single quote")
	}
	if state == stateDoubleQuote {
		return nil, fmt.Errorf("unterminated double quote")
	}

	if len(buf) > 0 {
		out = append(out, string(buf))
	}

	return out, nil
}
