# Configuration

Pickle asks for a projects folder the first time it opens. Change it later from
**Settings** in the web app.

The path belongs to the Linux host running Pickle. Pickle lists the direct child
directories inside it as projects.

Settings are stored at:

```text
~/.config/pickle/config.json
```

If `XDG_CONFIG_HOME` is set to an absolute path, Pickle stores the file under
`$XDG_CONFIG_HOME/pickle/config.json` instead. The file currently contains:

```json
{
  "projectsDirectory": "/home/user/code"
}
```

Pickle validates that a new projects path exists and is a directory before
saving it. The new path takes effect immediately.

## Command-line options

With the installed systemd user service:

```bash
pickle start
pickle stop
pickle restart
pickle status
pickle doctor
```

`status` and `doctor` check the server at its default address,
`http://127.0.0.1:8080`. `doctor` also checks tmux, your projects folder,
and Tailscale Serve. systemd and Tailscale are optional for foreground use.

On a VPS or another host that should start Pickle before you log in, enable
lingering for your user:

```bash
sudo loginctl enable-linger "$USER"
```

Remove the installed binary and service:

```bash
pickle uninstall
```

Settings are kept for the next installation. Add `--purge` to remove settings
and notification subscriptions too. Neither command removes tmux sessions,
projects, the source checkout, or Tailscale routes. Stop any manually launched
Pickle server before uninstalling; these commands manage the installed service.

Run plain `pickle` to start a server in the foreground. For that mode, use the
options below.

Listen on another local port:

```bash
pickle -addr 127.0.0.1:9000
```

Use another configuration file:

```bash
pickle -config /path/to/config.json
```

Temporarily override the projects folder:

```bash
pickle -projects-dir /path/to/projects
```

The command-line override takes priority and locks the projects-folder field in
the web app for that run.
