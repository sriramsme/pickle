# Notifications

Pickle can send standard Web Push notifications to devices that opt in.

Open **Settings** and choose **Enable** under Notifications. Use **Send test**
to verify delivery before relying on notifications from future agent
integrations.

Notifications require HTTPS or localhost. On iPhone and iPad, Pickle must be
opened from an installed Home Screen icon before it can ask for notification
permission.

Pickle stores its VAPID keys and device subscriptions at:

```text
~/.config/pickle/notifications.json
```

The file is created with permissions `0600`. Removing it creates a new identity
for the Pickle server and invalidates existing device subscriptions. Turn
notifications off and on again on each device after removing it.

## Send from the command line

`pickle notify` sends through the running Pickle server to every enabled device:

```bash
pickle notify "The task finished"
pickle notify --urgency high "Waiting for your approval"
```

Inside tmux, Pickle identifies the current session, window, pane, and supported
coding agent. It uses them to create the title and replacement tag, and tapping
the notification opens the relevant tmux session. Explicit `--title`, `--url`,
and `--tag` values override these defaults.

Options include:

- `--title` sets the notification title.
- `--url` opens a local Pickle route when the notification is tapped.
- `--tag` groups similar notifications on browsers that support it. By default,
  notifications from the same tmux pane replace one another.
- `--urgency` accepts `low`, `normal`, or `high`.
- `--stdin` reads the message from standard input.
- `--json` prints a machine-readable delivery result.
- `--server` overrides the default `http://127.0.0.1:8080`. The
  `PICKLE_URL` environment variable provides the same default override.
- `--timeout` controls how long the command waits for delivery.

The command exits with a failure status when Pickle is unreachable, no devices
are enabled, the message is invalid, or delivery fails to every device.

Notification action buttons are not currently used because browser support is
inconsistent and Safari does not support them. Use `--url` to open the relevant
Pickle screen and perform approvals with their full context visible.

## Agent instruction

Add this to a root `AGENTS.md` or the equivalent instruction file used by your
coding agents:

```md
## Pickle notifications

If the `pickle` command is available, send one concise notification when:

- you finish a task that took long enough that I may have stepped away
- you are blocked and need my input or approval
- a long-running command fails and needs my attention

Use:

`pickle notify "<outcome or exact input needed>"`

When running outside tmux, add `--title` and `--url` when they provide useful
context. Use `--urgency high` only when work cannot continue without me. Do not
notify for routine progress, every tool call, or while I am actively responding.
Never include secrets, tokens, source code, customer data, or other sensitive
information because notifications may appear on a lock screen. If notification
delivery fails, continue normally and mention it in your final response instead
of retrying repeatedly.
```
