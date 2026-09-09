# Local development

Pickle's server currently targets Linux. Development requires:

- Go 1.24 or newer
- Node.js 22 or newer
- pnpm 11.9.0
- tmux

## Run the development servers

Install frontend packages:

```bash
cd web
pnpm install
```

From the repository root, start Go:

```bash
go run ./cmd/server
```

In another terminal, start Vite:

```bash
cd web
pnpm dev
```

Open <http://127.0.0.1:5173>. Vite proxies `/api` and `/ws` to Go on port 8080.

A fresh config redirects to `/setup`. To use a temporary projects folder without
writing config, start Go with:

```bash
go run ./cmd/server -projects-dir /path/to/projects
```

## Build the embedded server

```bash
cd web
pnpm build
cd ..
go build -o pickle ./cmd/server
```

The frontend is embedded in the Go binary. Run it with:

```bash
./pickle
```

## Checks

```bash
gofmt -w cmd internal
go test -count=1 ./...
cd web
pnpm typecheck
pnpm build
```

## Release files

Build Linux x86-64 and ARM64 binaries, the systemd unit, and checksums:

```bash
scripts/package-release.sh
```

Files are written under `build/`. Pushing a tag beginning with `v` runs the same
build and publishes a GitHub release.

## Architecture

```text
Browser -> xterm.js -> WebSocket -> Go -> PTY -> tmux
```

React, TypeScript, Vite, TanStack Router, TanStack Query, Tailwind CSS, and
`@xterm/xterm` make up the frontend. The backend uses Go's `net/http`, a
WebSocket connection, a Unix PTY, tmux, and Linux `/proc` service discovery.
