# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

AppUpdateManager is a software auto-update system with two parts:

- **Client**: Windows GUI program written in Go + Fyne v2. It manages a target `.exe`, communicates with the server over WebSocket, and supports self-update, system tray, and Windows autostart.
- **Server**: Ubuntu web service with a Go + Gin backend, SQLite database, and a Vue 3 + TypeScript management console.

The repository has two independent Go modules:

- `server/backend` — module `example.com/appupdatemanager/server`
- `client` — module `example.com/appupdatemanager/client`

There is also a Node/Vite frontend under `server/frontend`.

## Common Commands

### Server backend

```bash
cd server/backend
go mod tidy
go run ./cmd/server
```

Run flags:

- `-addr :8080` — HTTP listen address
- `-config config/accounts.txt` — accounts file path
- `-data ./data` — directory for SQLite DB and uploaded files

### Server frontend

```bash
cd server/frontend
npm install
npm run dev
```

`vite.config.ts` proxies `/api`, `/ws`, and `/files` to `http://127.0.0.1:8080` during development. Build output goes to `../backend/static`.

### Full server build

```bash
./scripts/build-server.sh
```

Produces `dist/server/` with the backend binary, frontend static assets, and `accounts.txt`.

### Server package

```bash
./scripts/package-server.sh
```

Builds the server and creates `dist/appUpdateManager-server.tar.gz`.

### Client

On Windows:

```bash
cd client
go run .
```

On macOS, cross-compile to Windows (requires `mingw-w64`):

```bash
./scripts/build-client.sh
```

Produces `dist/client/appUpdateManager-client.exe` with the icon embedded. Requires `client/assets/icon.ico` and `client/assets/icon.rc` to exist.

### Tests

There are currently no test files in the repository. To add or run Go tests:

```bash
cd server/backend
go test ./...

# or for a single package
go test ./internal/store

# or for a single test
go test ./internal/store -run TestName
```

```bash
cd client
go test ./...
```

## High-Level Architecture

### Server backend (`server/backend`)

- **`cmd/server/main.go`** — entry point. Parses flags, loads `accounts.txt`, opens SQLite with WAL mode, runs database migrations, syncs users, starts the WebSocket hub, and wires the Gin router.
- **`config/config.go`** — reads `server/backend/config/accounts.txt` (`username:password` per line, `#` comments) and syncs bcrypt-hashed credentials into the DB on startup.
- **`internal/store/store.go`** — all SQLite access: migrations, users, software/client versions, clients, and pending commands.
- **`internal/model/models.go`** — shared structs for DB rows, heartbeat data, WebSocket messages, and command payloads.
- **`internal/api/`** — Gin HTTP handlers for login, software versions, client versions, client management, and command triggers.
- **`internal/ws/hub.go`** — WebSocket hub. Clients register by name, send heartbeats, and receive commands. Pending commands are stored in `client_commands` and flushed to the client on each heartbeat.
- **`internal/middleware/auth.go`** — custom JWT implementation with a hardcoded secret. Reads token from `Authorization: Bearer ...` header or `token` cookie.
- **`static/`** — embedded frontend production build.
- **`data/files/`** — uploaded software and client binaries, served under `/files`.

### Server frontend (`server/frontend`)

- Vue 3 + TypeScript + Vite + Element Plus + Pinia + Vue Router.
- Views: `LoginView`, `OverviewView`, `DashboardView`, `SoftwareView`, `ClientsView`.
- `src/api/index.ts` configures Axios with the JWT token and base URL.
- `src/stores/auth.ts` manages login state.

### Client (`client`)

- **`main.go`** — Fyne app lifecycle: load config, embed `assets/icon.svg` for the app/window icon and `assets/icon.ico` for the system-tray icon, build settings/status tabs, create the server client, set up system tray, start heartbeat/status updater, auto-connect to server. `assets/icon.ico` is also compiled into the Windows executable resource via `windres`.
- **`internal/config/config.go`** — local JSON config stored in the OS user config dir (`appUpdateManager/client.json`). Fields: server host/port, client name/version, autostart flag.
- **`internal/server/client.go`** — WebSocket client: connect, register by name, heartbeat loop, read loop, download files via HTTP.
- **`internal/software/manager.go`** — manages `software/` subdirectory per version, starts/stops/restarts the managed executable, tracks runtime.
- **`internal/updater/self.go`** — Windows-only self-update: downloads `client.exe.new`, writes `updater.bat`, starts it, and exits. The batch file replaces the running exe and self-deletes.
- **`internal/autostart/autostart_windows.go`** — Windows registry-based autostart under `HKCU\SOFTWARE\Microsoft\Windows\CurrentVersion\Run`.
- **`internal/sysinfo/info.go`** — collects IP, OS version, memory, and CPU info using `gopsutil`.
- **`internal/systray/systray.go`** — Fyne desktop system tray menu.

## Communication Flow

1. Client opens WebSocket to `/ws` and sends a `register` message with its name.
2. Client sends `heartbeat` every 10 seconds; the server upserts the client row in SQLite.
3. Web console calls REST APIs (e.g. `POST /api/clients/:id/update-software`).
4. The handler persists a pending command and pushes it through the hub to the named client.
5. The client receives the `command` message, downloads the file from `/files/...`, and performs the requested action (`update_software`, `update_self`, `start`, `stop`, `restart`).

## Default Accounts

`server/backend/config/accounts.txt`:

```
admin:123456
operator:operator123
```

The server hashes these with bcrypt and syncs them into the DB on startup. Changing the file requires a server restart to take effect.

## Deployment Notes

- Server is deployed on Ubuntu as a systemd service using `scripts/appupdatemanager.service`. See `docs/deploy.md` for full steps.
- Client is distributed as `appUpdateManager-client.exe` on Windows.
- Client self-update and registry autostart may require administrator privileges on Windows.
- The JWT secret is hardcoded in `server/backend/internal/middleware/auth.go`. Change it for production.

## Build Progress Tracking

`build-progress.json` tracks the construction phases from `target.md`. Update it when starting or completing tasks.
