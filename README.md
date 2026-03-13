# webnetd

A web-based terminal server. Like telnetd, but the client is your browser.

webnetd spawns a shell in a pseudo-terminal and connects it to a web browser over WebSockets. The embedded web UI uses [xterm.js](https://xtermjs.org/) for terminal rendering and includes a chat-style command input bar and file upload support.

## Features

- **Browser-based terminal** — full PTY session accessible from any modern browser
- **Command input bar** — type commands in a fixed bottom bar (or click the terminal for direct input)
- **File upload** — drag-and-drop or click to upload files to the server
- **PIN authentication** — optional 6-digit PIN printed to the server log
- **Single binary** — all static assets are embedded at compile time via `go:embed`
- **Zero configuration** — sensible defaults, just run it

## Install

```sh
go install github.com/FrancoAA/webnetd@latest
```

Or build from source:

```sh
git clone https://github.com/FrancoAA/webnetd.git
cd webnetd
go build -o webnetd .
```

## Usage

```sh
# Start with defaults (port 8080, user's $SHELL)
./webnetd

# Custom port and shell
./webnetd --addr :2323 --shell /bin/bash

# Enable PIN authentication
./webnetd --auth

# Set upload directory
./webnetd --upload-dir /tmp/uploads

# Combine options
./webnetd --addr :9090 --auth --shell /bin/zsh --upload-dir ~/uploads
```

Then open `http://localhost:8080` in your browser.

When `--auth` is enabled, the server prints a 6-digit PIN to the log:

```
2024/01/15 10:30:00 auth: PIN is 482731
2024/01/15 10:30:00 webnetd listening on :8080 (shell: /bin/bash, auth: true, upload-dir: .)
```

Enter this PIN in the browser login screen to start a session.

## Options

| Flag | Default | Description |
|------|---------|-------------|
| `--addr` | `:8080` | Listen address (host:port) |
| `--shell` | `$SHELL` or `/bin/sh` | Shell to execute |
| `--auth` | `false` | Enable PIN authentication |
| `--upload-dir` | `.` | Directory for file uploads |
| `-v`, `--version` | | Print version and exit |

## Testing

```sh
go test -v ./...
```

## License

Apache 2.0
