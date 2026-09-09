# Development services

Pickle finds TCP listeners owned by processes running inside configured
projects. It also finds containers started by Docker Compose projects in those
directories. Docker is optional.

Service details can include the process or container name, runtime, health,
image, and published ports. Pickle probes published ports over loopback to find
likely HTTP apps.

## Tailscale Serve actions

Pickle can expose an HTTP service to the tailnet, open its private URL, and
remove it from Serve again. Give your user one-time permission to update Serve
rules:

```bash
sudo tailscale set --operator=$USER
```

The app itself should remain on localhost. Tailscale's
[Serve documentation](https://tailscale.com/docs/features/tailscale-serve)
explains how private routing and tailnet access rules work.

## Vite host checks

Vite-based apps may reject the tailnet hostname with a 403 response. Allow this
machine's exact Tailscale hostname when starting the development server:

```bash
export __VITE_ADDITIONAL_SERVER_ALLOWED_HOSTS="$(
  tailscale status --json | jq -r '.Self.DNSName | rtrimstr(".")'
)"
pnpm dev
```

## Ending services

Ending a host process sends it `SIGTERM`. Ending a container runs `docker stop`.
Pickle asks for confirmation before either action.
