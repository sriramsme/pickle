# AGENTS.md

## What Pickle is

Pickle is a small, self-hosted control surface for a Linux development
machine. It lets someone open a browser on a laptop, tablet, or phone and use
the terminal environment that already lives on the host.

The browser is only the control surface. Shells, tmux, Neovim, Codex, and other
tools still run on the Linux machine.

Pickle should feel like a terminal appliance, not a browser IDE or a cloud
development platform. It complements normal terminal workflows instead of
replacing them.

## Current shape

```text
Browser
  |
  | HTTPS / WebSocket
  v
Tailscale Serve
  |
  v
Go server on localhost
  |
  +-- embedded React frontend
  |
  +-- PTY
        |
        v
       tmux
```

The frontend uses React, TypeScript, Vite, TanStack Router, TanStack Query,
Tailwind CSS, and `@xterm/xterm`. The backend uses Go, `net/http`, a WebSocket
connection, and a Unix PTY. A production build embeds the frontend in the Go
binary.

During development, Vite and Go run separately. Vite proxies `/ws` to the Go
server.

## Terminal rules

tmux owns session persistence. Pickle must not invent another session layer or
interfere with normal tmux use.

The default browser session is a general-purpose session named `pickle`. Start
or attach to it directly:

```go
exec.Command("tmux", "new-session", "-A", "-s", "pickle")
```

Never build shell commands from request data or invoke this through `sh -c`.
Validate any identifier that later comes from a client.

For each browser connection:

1. accept the WebSocket
2. create a PTY
3. attach to the `pickle` tmux session
4. copy browser input into the PTY
5. copy PTY output back to the browser
6. apply terminal resize messages
7. clean up the PTY, process, goroutines, and socket on disconnect

Closing the browser may kill its tmux client. It must not kill the tmux session.
A reconnect creates another PTY and returns to the tmux session used by the
previous Pickle client.

Do not build a Mosh-like protocol. WebSocket reconnection plus tmux persistence
is enough.

Existing tmux workflows must keep working, including normal attach commands,
tmux-sessionizer, and custom keybindings. The expected prefix is `Ctrl-b`.

## Security boundary

Pickle provides a shell on the host, so remote access is highly privileged.

- Bind to `127.0.0.1` by default.
- Use Tailscale Serve for private HTTPS access.
- Treat tailnet grants and device identity as the current access control.
- Do not enable Tailscale Funnel or another public route by default.
- Never commit tokens, tailnet names, hostnames, email addresses, or machine-specific paths.
- Keep trusted proxy headers safe by ensuring the backend cannot be reached remotely without the proxy.

Application login and public temporary access are separate security projects.
Do not add a quick password screen or persistent browser auth as a shortcut.

## Priorities

When choices conflict, use this order:

1. terminal correctness
2. connection reliability
3. tmux session persistence
4. low latency
5. keyboard usability
6. simplicity
7. maintainability
8. visual polish
9. new features

A polished UI with unreliable terminal behavior is a failed change.

## Backend style

Keep the Go server small and explicit.

- Prefer the standard library.
- Keep process ownership and cancellation visible.
- Handle errors directly.
- Use small packages only when they separate real responsibilities.
- Avoid interfaces until there is more than one useful implementation.
- Run `gofmt`.
- Test parsing, validation, and non-trivial lifecycle behavior where practical.

Do not add generic session managers, plugin systems, databases, queues, or
service abstractions before the product needs them.

## Frontend style

The terminal should occupy almost the entire viewport.

- Dark background
- Orange accent near `#e68e0d`
- Very little chrome
- Tailwind utilities backed by a small set of semantic theme tokens
- Strict TypeScript
- Functional React components
- No global state library without a demonstrated need
- Keep the xterm.js lifecycle in one focused component or hook
- Do not add subtitles or helper copy that repeats a clear label

Use TanStack Router for routes and TanStack Query for server data. Keep
feature-specific components and API definitions together. Leave xterm rules,
touch behavior, and other selectors that depend on third-party markup in CSS.

The UI should feel calm and purpose-built. Avoid dashboard cards, generic SaaS
styling, or browser-IDE furniture.

The touch toolbar includes Esc, Ctrl, Tab, `Ctrl-b`, `Ctrl-f`, and arrow keys.
Keep it compact and hidden from desktop users. Do not build a large virtual
keyboard. Clipboard helpers can wait until they solve a concrete problem.

## Repository shape

Keep the layout close to the current structure:

```text
.
├── cmd/server/
├── internal/config/
├── internal/server/
├── internal/terminal/
├── web/src/
├── AGENTS.md
├── README.md
└── go.mod
```

Add another layer only when it removes real complexity.

Use:

- `pnpm` for frontend packages
- Go modules for backend packages

The first target is Arch Linux, but application code should remain portable to
other Linux hosts where practical.

## Working on the project

Before changing code:

1. inspect the current implementation and working tree
2. state any assumption that changes behavior or scope
3. choose the smallest coherent patch

While changing code:

- Touch only files required by the task.
- Match the style already in the repository.
- Keep code flat and readable.
- Remove only the unused code created by the change.
- Do not add speculative configuration or extension points.

Before handing work back:

1. run focused tests
2. run the relevant builds and type checks
3. run `git diff --check`
4. inspect the important diff and repository status
5. report what was tested and what still needs a real browser or device

## Scope

The working foundation includes:

- one localhost Go server
- one embedded frontend
- one terminal WebSocket
- one PTY per browser connection
- tmux session discovery and selected-session attachment
- project discovery under a configurable directory
- one-tap project session creation using tmux-sessionizer naming
- read-only discovery of project-owned TCP listeners through Linux `/proc`
- read-only discovery of project-owned Docker Compose containers
- service details with runtime, health, image, ports, and HTTP detection
- Tailscale Serve controls for HTTP services and confirmed service termination
- compact home previews with full lists at `/services`, `/sessions`, and `/projects`
- a default persistent `pickle` tmux session
- resize propagation
- simple reconnect behavior
- a minimal responsive terminal page
- an installable PWA shell
- a compact toolbar on touch devices
- private access through Tailscale Serve
- first-run host configuration and in-app project-folder settings
- opt-in Web Push notifications with host-side subscription storage

Near-term work should make this comfortable as a PWA on laptops, iPhones, and
iPads. Keep testing viewport behavior, hardware keyboard behavior, and the touch
toolbar against real terminal workflows.

The control-plane foundation lists tmux sessions, local projects, host
development servers, and Docker Compose services. Projects open in a matching
session at `/tmux/<session>`. Possible later work includes session controls,
agent status, logs, notifications, host status, and multiple hosts.

Do not add a database, account system, OAuth framework, Docker requirement,
Kubernetes, Redis, message queue, component library, or plugin architecture
until a concrete feature requires it.

## Open-source direction

Keep Pickle useful for someone cloning it onto an ordinary Linux machine:

- sensible defaults
- few dependencies
- a single production binary
- no hosted service requirement
- no personal paths or private network details in the repository
- clear setup commands
- licensed assets only

Personal taste should shape the defaults, but other users should be able to run
the project without reconstructing one machine's dotfiles.
