# kosync-go

A lightweight [KOReader](https://koreader.rocks/) sync server written in Go.

Inspired by and compatible with the original [koreader-sync-server](https://github.com/koreader/koreader-sync-server). No Redis, no Docker, no config files — just a single binary that runs anywhere.

## Features

- Full compatibility with KOReader's sync protocol
- Single static binary, zero dependencies
- Data persisted to a JSON file (saved on shutdown and every 5 minutes)
- TLS out of the box with a self-signed certificate
- Bring your own certificate for production use
- Designed to run behind a reverse proxy (nginx, lighttpd, Caddy, Traefik)
- Systemd-ready with graceful shutdown

## Getting started

### Build from source

```bash
git clone https://github.com/fellipec/kosync-go
cd kosync-go
go build -o kosync-go .
```

Requires Go 1.26 or later.

### Run

```bash
# Default: HTTPS on port 7200 with a self-signed certificate
./kosync-go

# Behind a reverse proxy: plain HTTP on port 17200
./kosync-go --insecure --port 17200

# With your own certificate
./kosync-go --cert /path/to/cert.pem --key /path/to/key.pem

# Custom data file location
./kosync-go --store-file /var/lib/kosync/data.json

# Disable new user registration (after initial setup)
./kosync-go --disable-new-users
```

## Options

| Flag | Default | Description |
| ------ | --------- | ------------- |
| `--port` | 7200 (TLS) / 17200 (insecure) | Port to listen on |
| `--listen-addr` | `` (all interfaces) | Address to bind to |
| `--insecure` | false | Disable TLS (for reverse proxy use) |
| `--cert` | — | Path to TLS certificate file |
| `--key` | — | Path to TLS private key file |
| `--store-file` | `kosync.json` | Path to the data file |
| `--disable-new-users` | false | Prevent new user registration |

## Recommended setup

Running behind a reverse proxy with a real certificate is the recommended production setup. It gives you proper TLS, access logs, and easy integration with Let's Encrypt.

```bash
./kosync-go --insecure --listen-addr 127.0.0.1 --port 17200 \
            --store-file /var/lib/kosync/data.json \
            --disable-new-users
```

See the systemd section below for running as a service.

### Self-signed certificate

If you run without `--insecure` and without providing `--cert`/`--key`, the server generates a self-signed ECDSA certificate on startup. The certificate is ephemeral (regenerated on each restart).

## Running as a systemd service

Create `/etc/systemd/system/kosync.service`:

```ini
[Unit]
Description=KOReader Sync Server
After=network.target

[Service]
Type=simple
User=kosync
ExecStart=/usr/local/bin/kosync-go \
    --insecure \
    --listen-addr 127.0.0.1 \
    --port 17200 \
    --store-file /var/lib/kosync/data.json \
    --disable-new-users
Restart=on-failure
RestartSec=10

[Install]
WantedBy=multi-user.target
```

Then:

```bash
useradd --system --no-create-home --shell /usr/sbin/nologin kosync
mkdir -p /var/lib/kosync
chown kosync:kosync /var/lib/kosync
systemctl daemon-reload
systemctl enable --now kosync
```

## Auto update (linux-amd64)

```bash
curl -fsSL https://raw.githubusercontent.com/fellipec/kosync-go/master/kosync-update.sh | sudo bash
```

### What this script does

- Fetches the latest release from GitHub  
- Downloads the correct Linux AMD64 binary  
- Stops the `kosync` systemd service  
- Replaces the binary in `/usr/local/bin/kosync-go`  
- Restarts the service  
- Cleans up temporary files  

## KOReader configuration

In KOReader set:

- **Custom sync server**: your server URL (e.g. `https://k.example.com`)
- **Username** and **password**: credentials you registered with the server

To register a new user, make sure `--disable-new-users` is not active

## Security notes

- Passwords are never stored in plain text. The server stores a bcrypt hash of the MD5 digest sent by KOReader.
- The KOReader client sends an MD5 hash of the password, not the password itself. MD5 is considered insecure, so never use this open to the Internet without TLS encryption.
- For production use, always run behind a reverse proxy with a valid TLS certificate.
- After creating your account, you may want to use use `--disable-new-users` to prevent further registrations.

## Data storage

All data is stored in a single JSON file. The file is written atomically (via a temp file rename) to prevent corruption. It is saved:

- On clean shutdown (SIGTERM or SIGINT)
- Every 5 minutes, if data has changed

## License

This project is licensed under the [GNU General Public License v3.0](LICENSE).

## Acknowledgements

This project implements the sync protocol defined by the [KOReader](https://koreader.rocks/) project and is compatible with the original [koreader-sync-server](https://github.com/koreader/koreader-sync-server). The original server is written in Lua and uses Redis and OpenResty. This implementation reimplements the same HTTP API from scratch in Go, with no shared code.
