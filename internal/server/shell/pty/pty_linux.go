//go:build linux

package pty

import "golang.org/x/sys/unix"

func termiosIoCtl() (get, set uint) {
	return unix.TCGETS, unix.TCSETS
}
