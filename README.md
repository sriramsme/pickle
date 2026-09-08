# Pickle

Pickle puts a real tmux terminal in your browser. It is meant for reaching a
Linux development machine from a laptop, iPad, or phone without turning the
browser into an IDE.

```text
Browser -> xterm.js -> WebSocket -> Go -> PTY -> tmux
```

The web app uses TanStack Router for its pages, TanStack Query for live server
data, and Tailwind CSS for a small set of shared visual tokens and utilities.
Terminal connection state remains local to the terminal component.

Your shell, Neovim, Codex, and other terminal tools keep running on the host.
tmux keeps the session alive when the browser closes or the connection drops.
Pickle also remembers which tmux session its client was viewing and returns to
it after reconnecting.

Pickle lists the tmux sessions already running on the host and opens one as a
full-screen browser terminal. If tmux has no sessions yet, it can start the
general-purpose `pickle` session. It also lists projects and can start a tmux
session in a project's directory.

## What you need

- Linux
- Go 1.24 or newer
- Node.js 22 or newer
- pnpm
- tmux
- Tailscale, if you want private remote access

## Run it locally

Install the frontend packages once:

```bash
cd web
pnpm install
```

Start the Go server from the repository root:

```bash
go run ./cmd/server
```

In another terminal, start Vite:

```bash
cd web
pnpm dev
```

Open <http://127.0.0.1:5173>.

Vite serves the frontend and proxies `/api` and `/ws` to the Go server on
`127.0.0.1:8080`. Connecting runs:

```bash
tmux new-session -A -s pickle
```

That creates the session once and attaches to it afterward. Closing the browser
does not end the session.

## Build one binary

Build the frontend first, then compile Go:

```bash
cd web
pnpm build
cd ..
go build -o pickle ./cmd/server
```

Run it:

```bash
./pickle
```

Open <http://127.0.0.1:8080>. The React frontend is embedded in the binary.

After a Go-only rebuild and server restart, an open browser or home-screen app
reconnects on its own. Reload the page when the frontend changes; fully closing
and reopening the iOS app is one way to do that.

Pickle listens on localhost by default. The address can be changed with
`-addr`, but a private reverse proxy is the preferred way to reach it remotely.

## Reach it through Tailscale

Keep Pickle running on `127.0.0.1:8080`, then open another terminal and run:

```bash
tailscale serve --bg 8080
```

Tailscale prints a private HTTPS address for the machine. Open that address on
any device connected to the same tailnet. HTTP and WebSocket traffic are both
proxied to Pickle.

Check the active mapping with:

```bash
tailscale serve status
```

Remove Pickle from the default HTTPS port with:

```bash
tailscale serve --https=443 off
```

The Serve configuration runs in the background, but the Pickle process still
needs to be running. Process supervision and install packaging will come after
the basic workflow settles.

## Install it on a device

Open the private Tailscale HTTPS address in Safari on an iPhone or iPad, use the
Share menu, and choose **Add to Home Screen**. On a laptop, use the browser's
install action when it is available.

Pickle opens as a standalone app with its own icon. It still needs a live
connection to the host because the terminal itself cannot work offline.

## Tmux sessions

The home page previews active tmux sessions. The full list lives at `/sessions`
and shows window counts, attached clients, and recent activity. Selecting one
opens it at:

```text
/tmux/<session-name>
```

The list is read-only for now. Creating, renaming, and deleting sessions still
happens through tmux and the user's existing shell workflow.

## Projects

The home page previews projects, with the full list at `/projects`. Pickle lists
the direct child directories under `~/projects`. Selecting a project opens its
matching tmux session or creates one in that project directory. For consistency
with common tmux-sessionizer scripts, periods become underscores in tmux session
names.

Use a different projects directory with:

```bash
go run ./cmd/server -projects-dir /path/to/projects
```

## Development services

Services appear first on the home page, with the full list at `/services`.
Pickle lists TCP listeners owned by processes running inside discovered
projects. Each row shows the project, process name, and port. Pickle reads this
from Linux `/proc`, so unrelated host services remain out of the list.

Service discovery is read-only. The listed ports are not automatically exposed
through Tailscale yet.

## Security

Pickle gives the browser an interactive shell on the host. Treat access to it
like SSH access.

There is no application login yet. The current security boundary is:

- Pickle listens only on localhost.
- Tailscale Serve provides private HTTPS access.
- Tailnet policy controls which users and devices can connect.
- The service is not exposed through Tailscale Funnel or a public port.

Do not expose Pickle directly to the public internet. Public temporary access
needs a separate, carefully designed authentication flow.

## Current scope

The current build supports terminal input and output, ANSI applications, tmux,
Neovim, Codex, resizing, reconnecting to the same tmux session, and installation
as a PWA from a browser. Touch devices also get a compact toolbar for Esc, Ctrl,
Tab, `Ctrl-b`, `Ctrl-f`, and arrow keys, plus touch scrolling in terminal
scrollback and full-screen terminal apps. The home page discovers existing tmux
sessions, discovers local projects, and opens both in session-specific
terminals. It also shows development services running inside those projects.

Development-server links, session controls, container status, clipboard helpers,
and public authentication are later work.

## License

[MIT](LICENSE)
