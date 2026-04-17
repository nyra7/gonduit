//go:build darwin || freebsd || netbsd || openbsd

package pty

import "golang.org/x/sys/unix"

func termiosIoCtl() (get, set uint) {
	return unix.TIOCGETA, unix.TIOCSETA
}
