package main

import (
	"flag"
	"fmt"
	"os"
	"server/core"
	"server/log"
	"shared/pkg"
	"shared/util"
	"strings"
)

func main() {

	// Check version flag before any subcommand parsing
	for _, arg := range os.Args[1:] {
		if arg == "-v" || arg == "--version" {
			fmt.Println(pkg.Version)
			return
		}
	}

	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	var cfg core.Config

	switch os.Args[1] {

	case "bind":
		fs := flag.NewFlagSet("bind", flag.ExitOnError)
		fs.StringVar(&cfg.BindAddr, "addr", "0.0.0.0", "Address to bind the server")
		fs.IntVar(&cfg.BindPort, "port", 1337, "Port to bind the server")
		fs.StringVar(&cfg.AcceptAddr, "accept", "0.0.0.0", "Allowed IP addresses (comma-separated or CIDR)")
		fs.StringVar(&cfg.LogFile, "log-file", "", "Write log to file")
		fs.BoolVar(&cfg.Silent, "silent", false, "Suppresses all output including errors")
		fs.StringVar(&cfg.ServerIdentity, "identity", "", "A server identity to use for mTLS")
		fs.StringVar(&cfg.Fingerprint, "fingerprint", "", "The certificate fingerprint to accept when receiving connections")

		_ = fs.Parse(reorder(os.Args[2:]))

		if err := run(cfg); err != nil {
			log.Fatal(err)
		}

	case "reverse":
		fs := flag.NewFlagSet("reverse", flag.ExitOnError)
		fs.IntVar(&cfg.BindPort, "port", 1337, "Port of the remote server to connect to")
		fs.StringVar(&cfg.LogFile, "log-file", "", "Write log to file")
		fs.BoolVar(&cfg.Silent, "silent", false, "Suppresses all output including errors")
		fs.StringVar(&cfg.ServerIdentity, "identity", "", "A server identity to use for mTLS")
		fs.StringVar(&cfg.Fingerprint, "fingerprint", "", "The certificate fingerprint to verify when connecting")

		_ = fs.Parse(reorder(os.Args[2:]))

		if fs.NArg() < 1 {
			log.Fatal("reverse requires an address argument")
		}

		cfg.BindAddr = fs.Arg(0)
		cfg.Reverse = true

		if err := run(cfg); err != nil {
			log.Fatal(err)
		}

	default:
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	_, _ = fmt.Fprintf(os.Stderr, "Usage: gonduit <command> [flags]\n\n"+
		"Commands:\n"+
		"  bind	   Start the gonduit server in bind mode\n"+
		"  reverse  Start the gonduit server in reverse mode\n")
}

func run(cfg core.Config) error {

	log.InitLogger(log.LevelDebug, cfg.LogFile)

	defer log.CloseLogger()

	if cfg.Silent {
		log.SilenceConsole()
	}

	log.Infof("running version %s", pkg.Version)

	if cfg.Fingerprint != "" {

		if !util.IsFingerprint(cfg.Fingerprint) {
			return fmt.Errorf("invalid fingerprint: %s", cfg.Fingerprint)
		}

		log.Infof("restricting connections to client with fingerprint %s", cfg.Fingerprint)

	}

	if err := core.NewServer(cfg).Serve(); err != nil {
		log.Fatal(err)
	} else {
		log.Info("server exited")
	}

	return nil

}

func reorder(args []string) []string {

	var positional []string
	var flagArgs []string

	for i := 0; i < len(args); i++ {
		if strings.HasPrefix(args[i], "-") {
			flagArgs = append(flagArgs, args[i])
			if i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") {
				i++
				flagArgs = append(flagArgs, args[i])
			}
		} else {
			positional = append(positional, args[i])
		}
	}

	return append(flagArgs, positional...)

}
