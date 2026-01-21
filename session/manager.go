package session

import (
	"bytes"
	"fmt"
	"log"
	"net"
	"shells/command"
	"shells/style"
	"shells/util"
	"strings"
)

type Config struct {
	BindAddr   string
	BindPort   string
	AcceptAddr string
	Password   string
}

func (c *Config) Bind() string {
	return c.BindAddr + ":" + c.BindPort
}

func (c *Config) ShouldAccept(addr string) bool {

	// If no restriction, accept everything
	if c.AcceptAddr == "" || c.AcceptAddr == "0.0.0.0" {
		return true
	}

	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		host = addr
	}

	// Allow multiple comma-separated entries
	entries := strings.Split(c.AcceptAddr, ",")
	ip := net.ParseIP(host)
	if ip == nil {
		return false
	}

	for _, entry := range entries {
		entry = strings.TrimSpace(entry)

		if strings.Contains(entry, "/") {
			_, subnet, err := net.ParseCIDR(entry)
			if err == nil && subnet.Contains(ip) {
				return true
			}
			continue
		}

		if entry == host {
			return true
		}

	}

	return false
}

func (c *Config) CheckPassword(password string) bool {
	return c.Password == password || c.Password == ""
}

func (c *Config) HasPassword() bool {
	return c.Password != ""
}

type Manager struct {
	config *Config
	cmd    *command.Manager
}

func NewManager(config *Config, cmd *command.Manager) *Manager {
	return &Manager{config: config, cmd: cmd}
}

func (m *Manager) CommandManager() *command.Manager {
	return m.cmd
}

func (m *Manager) Run() {

	m.cmd.Freeze()

	bind := m.config.Bind()
	listener, err := net.Listen("tcp", bind)

	if err != nil {
		log.Fatalf("Failed to bind %s: %v", bind, err)
	}

	defer listener.Close()

	log.Printf("Server listening on %s\n", bind)

	for {

		conn, err := listener.Accept()

		if err != nil {
			log.Printf("Failed to accept client: %v", err)
			continue
		}

		clientAddr := conn.RemoteAddr().String()

		log.Printf("%s connected\n", clientAddr)

		go m.handleConnection(conn)

	}

}

func (m *Manager) handleConnection(conn net.Conn) {

	defer util.CloseConn(conn)

	addr := conn.RemoteAddr().String()

	if !m.config.ShouldAccept(addr) {
		log.Printf("Connection rejected: host %s not in %s", addr, m.config.AcceptAddr)
		return
	}

	if m.config.HasPassword() {

		util.WriteConn(conn, fmt.Sprintf("%s> ", style.BoldWhite.Apply("?")))

		buf := make([]byte, 256)
		n, err := conn.Read(buf)
		if err != nil {
			log.Printf("Failed to read password: %v", err)
			return
		}

		// Trim newline/carriage return
		input := string(buf[:n])
		input = string(bytes.TrimRight([]byte(input), "\r\n"))

		if input != m.config.Password {
			log.Printf("Authentication failed from %s: invalid password: %s", conn.RemoteAddr().String(), input)
			return
		}

	}

	// Start interactive command prompt loop
	if err := m.cmd.HandleConnection(conn); err != nil {
		log.Printf("Error handling connection: %v", err)
	} else {
		log.Printf("Connection closed from %s", conn.RemoteAddr().String())
	}

}
