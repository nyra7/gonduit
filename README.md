# Gonduit

A lightweight, cross-platform bind shell server written in Go. Provides shell access over TCP with optional password authentication.

## Features

- Lightweight and fast with minimal dependencies
- Optional password authentication
- Cross-platform support (Linux, macOS, Windows, FreeBSD)
- Interactive shell with PTY support on Unix systems
- Customizable bind address and port

## Quick Start

### Building

```bash
git clone https://github.com/nyra7/gonduit.git
cd gonduit
make
```

### Running

```bash
# Basic usage, binds on 0.0.0.0:1337
./gonduit

# With password protection
./gonduit --bind-port 1337 --password "your-password"

# Accept connections from specific address with password protection
./gonduit --bind-port 1337 --accept-addr "192.168.1.100" --password "your-password"
```

### Connecting

```bash
nc localhost 1337
telnet localhost 1337
socat - TCP:localhost:1337
```

## Configuration

| Flag            | Description                              | Default   |
|-----------------|------------------------------------------|-----------|
| `--bind-addr`   | Address to bind to                       | `0.0.0.0` |
| `--bind-port`   | Port to listen on                        | `1337`    |
| `--password`    | Authentication password (optional)       | none      |
| `--accept-addr` | Restrict connections to specific address | `0.0.0.0` |

## Building

You can quickly build the project for all platforms and architectures using the provided Makefile.

```bash
make build-all
```

