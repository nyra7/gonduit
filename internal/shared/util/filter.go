package util

import (
	"fmt"
	"net"
	"strings"
)

type RejectedFunc func(conn net.Conn, err error)

// FilterListener wraps a net.Listener to restrict accepted connections based on allowed IPs or subnets
type FilterListener struct {
	net.Listener
	ips          []net.IP
	subnets      []*net.IPNet
	allowAll     bool
	rejected     RejectedFunc
	acceptString string
}

func NewFilterListener(l net.Listener, acceptAddr string, onReject RejectedFunc) (*FilterListener, error) {

	fl := &FilterListener{Listener: l, rejected: onReject}

	if acceptAddr == "" || acceptAddr == "0.0.0.0" {
		fl.allowAll = true
		return fl, nil
	}

	for _, raw := range strings.Split(acceptAddr, ",") {
		entry := strings.TrimSpace(raw)
		if entry == "" {
			continue
		}

		if strings.Contains(entry, "/") {
			_, subnet, err := net.ParseCIDR(entry)
			if err != nil {
				return nil, fmt.Errorf("invalid CIDR %q: %w", entry, err)
			}
			fl.subnets = append(fl.subnets, subnet)
		} else {
			ip := net.ParseIP(entry)
			if ip == nil {
				return nil, fmt.Errorf("invalid IP %q", entry)
			}
			fl.ips = append(fl.ips, ip)
		}
	}

	var s []string

	for _, ip := range fl.ips {
		s = append(s, ip.String())
	}

	for _, subnet := range fl.subnets {
		s = append(s, subnet.String())
	}

	fl.acceptString = strings.Join(s, ", ")

	return fl, nil

}

func (f *FilterListener) ShouldAccept(conn net.Conn) error {

	if f.allowAll {
		return nil
	}

	addr := conn.RemoteAddr().String()
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		host = addr
	}

	ip := net.ParseIP(host)

	// Should never happen, but just in case
	if ip == nil {
		return fmt.Errorf("invalid remote IP: %s", host)
	}

	for _, subnet := range f.subnets {
		if subnet.Contains(ip) {
			return nil
		}
	}

	for _, allowed := range f.ips {
		if allowed.Equal(ip) || allowed.IsLoopback() && ip.IsLoopback() {
			return nil
		}
	}

	return fmt.Errorf("connection not allowed: %s is not in [%s]", host, f.acceptString)

}

func (f *FilterListener) Accept() (net.Conn, error) {

	for {

		conn, err := f.Listener.Accept()

		if err != nil {
			return nil, err
		}

		if err = f.ShouldAccept(conn); err == nil {
			return conn, nil
		}

		f.rejected(conn, err)

		_ = conn.Close()

	}

}

func (f *FilterListener) AcceptString() string {
	return f.acceptString
}
