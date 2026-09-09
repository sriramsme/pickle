# Security

Pickle provides terminal and process access to its host. Treat access to it like
SSH access.

Keep Pickle bound to localhost and put Tailscale Serve in front of it. Do not
publish it with Tailscale Funnel or a public reverse proxy. Pickle does not have
application-level authentication yet.

If you find a vulnerability, use **Report a vulnerability** in this
repository's GitHub Security tab. Please do not include security details,
tokens, hostnames, or other private data in a public issue.
If that form is unavailable, open an issue asking for private contact without
describing the vulnerability.

Security fixes target the latest release and the current `main` branch while
Pickle is pre-1.0.
