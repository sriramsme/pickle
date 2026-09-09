# Pickle

Pickle turns a Linux development machine into a private remote dev lab. Open it
from a laptop, tablet, or phone and use the tmux sessions, shells, Neovim, Codex,
and other tools that already run on that machine.

Your work stays on the host. tmux keeps it alive when the browser closes or the
network drops.

## What it does

- Runs a real tmux terminal in the browser
- Reconnects to persistent sessions
- Lists tmux sessions and local projects
- Finds project development servers and Docker Compose services
- Opens HTTP services privately through Tailscale Serve
- Installs as a home-screen app on iPhone and iPad

## Supported setup

Install the Pickle server on a Linux x86-64 or ARM64 machine with tmux. Open it
from any modern browser on Linux, macOS, Windows, iOS, or Android.

Docker is optional. Tailscale is strongly recommended for private remote access.
Windows hosts can run Pickle inside
[WSL2](https://tailscale.com/docs/install/windows/wsl2). Native macOS and Windows
hosts are not supported yet.

## Install

Download and run the installer as your normal user:

```bash
curl -fsSL https://raw.githubusercontent.com/sriramsme/pickle/main/install.sh -o install.sh
sh install.sh
```

The script verifies the downloaded release, installs Pickle under
`~/.local/bin`, and starts a systemd user service when available. Open the setup
address it prints and choose the folder that contains your projects.

Run the installer again later to update to the latest release.

## Private access with Tailscale

Install Tailscale using its [Linux guide](https://tailscale.com/docs/install/linux),
connect the host to your tailnet, and run:

```bash
tailscale serve --bg 8080
```

Tailscale prints a private HTTPS address. Open it from another device on the
same tailnet. Its [Serve guide](https://tailscale.com/docs/features/tailscale-serve)
explains the access rules and HTTPS setup.

Do not publish Pickle with Tailscale Funnel. Pickle provides shell and process
access to its host, so treat access to it like SSH access.

## Install on a phone or tablet

Open the private Tailscale address in Safari on an iPhone or iPad, use the Share
menu, and choose **Add to Home Screen**. Pickle opens as a standalone app and
reconnects to the same tmux session when possible.

## More

- [Configuration](docs/configuration.md)
- [Development services](docs/services.md)
- [Local development](docs/development.md)
- [Security](SECURITY.md)

Pickle is available under the [MIT License](LICENSE).
