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
