package util

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"regexp"
	"runtime"
	"sync/atomic"
	"unicode"

	"google.golang.org/grpc/status"
)

var (
	descRe            = regexp.MustCompile(`desc\s*=\s*"([^"]*)"`)
	certFingerprintRe = regexp.MustCompile(`^[0-9A-Fa-f]{64}$`)
)

// IDGenerator is a helper object allowing to generate unique IDs
type IDGenerator struct {
	n uint64
}

// NewIDGenerator creates a new IDGenerator instance
func NewIDGenerator() *IDGenerator {
	return &IDGenerator{n: 0}
}

// Next generates the next unique ID
func (g *IDGenerator) Next() uint64 {
	return atomic.AddUint64(&g.n, 1)
}

// QuotedSplit splits a quoted string into words
func QuotedSplit(s string) ([]string, error) {
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

// ClearTerminal clears the native terminal screen
func ClearTerminal() {
	switch runtime.GOOS {
	case "windows":
		cmd := exec.Command("cmd", "/c", "cls")
		cmd.Stdout = os.Stdout
		_ = cmd.Run()

	default:
		cmd := exec.Command("clear")
		cmd.Stdout = os.Stdout
		_ = cmd.Run()
	}
}

// DupSlice duplicates a slice
func DupSlice[T any](s []T) []T {
	return append([]T(nil), s...)
}

// CloseFile closes a file and ignores any errors
func CloseFile(f *os.File) {
	_ = f.Close()
}

func HumanReadableBytes(b uint64) (float64, string) {
	const (
		KB = 1 << 10
		MB = 1 << 20
		GB = 1 << 30
	)

	switch {
	case b < KB:
		return float64(b), "bytes"
	case b < MB:
		return float64(b) / KB, "KB"
	case b < GB:
		return float64(b) / MB, "MB"
	default:
		return float64(b) / GB, "GB"
	}
}

func HumanReadableBytesWithUnit(b uint64, unit string) float64 {
	const (
		KB = 1 << 10
		MB = 1 << 20
		GB = 1 << 30
	)

	switch unit {
	case "bytes":
		return float64(b)
	case "KB":
		return float64(b) / KB
	case "MB":
		return float64(b) / MB
	case "GB":
		return float64(b) / GB
	default:
		return float64(b)
	}
}

func LocalIPAndInterface(addr net.Addr) (net.IP, *net.Interface, error) {

	tcpAddr, ok := addr.(*net.TCPAddr)
	if !ok {
		return nil, nil, fmt.Errorf("not a TCP connection")
	}
	ip := tcpAddr.IP

	ifaces, err := net.Interfaces()
	if err != nil {
		return ip, nil, err
	}

	for _, iface := range ifaces {
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}
		for _, addr = range addrs {
			var ipNet *net.IPNet
			switch v := addr.(type) {
			case *net.IPNet:
				ipNet = v
			case *net.IPAddr:
				ipNet = &net.IPNet{IP: v.IP, Mask: v.IP.DefaultMask()}
			}
			if ipNet != nil && ipNet.IP.To16().Equal(ip) {
				return ip, &iface, nil
			}
		}
	}

	return ip, nil, fmt.Errorf("interface not found for IP %v", ip)
}

// CalculateFileChecksum computes SHA256 hash of entire file
func CalculateFileChecksum(file *os.File) (string, error) {
	hasher := sha256.New()
	if _, err := io.Copy(hasher, file); err != nil {
		return "", err
	}
	return hex.EncodeToString(hasher.Sum(nil)), nil
}

func ParseGrpcError(err error) error {
	e, ok := status.FromError(err)
	if ok {
		return errors.New(stripDesc(e.Message()))
	}
	return err
}

func IsFingerprint(s string) bool {
	return certFingerprintRe.MatchString(s)
}

func stripDesc(s string) string {
	return descRe.ReplaceAllStringFunc(s, func(match string) string {
		// Extract the inner quoted text
		m := descRe.FindStringSubmatch(match)
		if len(m) == 2 {
			return m[1] // replace whole `desc = "..."` with inner text
		}
		return match
	})
}
