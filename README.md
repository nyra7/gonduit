# Gonduit

Gonduit is a Go tool for managing fully interactive shells across multiple hosts and operating systems at the same time.
It runs over gRPC and TLS 1.3, supports optional mTLS authentication and handles file transfers in both directions, all 
managed from a single TUI client. It is intended for use in penetration testing engagements, lab setups, and CTF 
challenges where you need reliable shell access across different platforms.

## Features

- Fully interactive shells across Windows, Linux, and macOS
- Multiple concurrent sessions with per-host isolation
- Bidirectional file transfer
- Bind and reverse connection modes
- TLS 1.3 with optional mTLS and certificate fingerprint validation
- Built-in identity and certificate management

## Build

```bash
git clone https://github.com/nyra7/gonduit.git
cd gonduit
make all
```

Cross-compilation is supported via `OS` and `ARCH`:

```bash
make server OS=linux ARCH=amd64
make app OS=darwin ARCH=arm64
```

## Basic Usage (Bind Mode)

**Server:**
```bash
# Run the server in bind mode on 0.0.0.0:1337
gonduit-server bind
```

**App:**
```bash
# Run the TUI
gonduit-app

# Connect to a host on default port (1337)
> connect <host>
```

## Documentation

See the [Wiki](../../wiki) for full usage reference, including server flags, identity management, and quickstart scenarios.

## License

MIT. See [LICENSE](LICENSE).

## Future Ideas

- Run shells as other users
- Add retries for reverse connections
- Implement simple linux privilege escalation checks (e.g. gtfobins)
- Implement windows privilege abuse (e.g. SeTcbPrivilege, SeImpersonatePrivilege)

## Known Issues

The `app` does not run properly on Windows for now. Window resize messages are not dispatched in shell view and the output is not displayed properly when connecting to linux hosts.