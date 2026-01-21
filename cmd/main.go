package main

import (
	"flag"
	"shells/command"
	"shells/session"
)

func main() {

	cfg := &session.Config{}

	flag.StringVar(&cfg.BindAddr, "bind-addr", "0.0.0.0", "Address to bind the server")
	flag.StringVar(&cfg.BindPort, "bind-port", "1337", "Port to bind the server")
	flag.StringVar(&cfg.AcceptAddr, "accept-addr", "0.0.0.0", "Address, addresses (comma-separated) or subnetwork allowed to connect")
	flag.StringVar(&cfg.Password, "password", "", "Optional password")

	flag.Parse()

	commandManager := command.NewManager()
	sessionManager := session.NewManager(cfg, commandManager)
	sessionManager.Run()

}
