package cli

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/sriramsme/pickle/internal/agents"
	"github.com/sriramsme/pickle/internal/notifications"
)

const tmuxContextFormat = "#{session_name}\t#{window_index}\t#{pane_index}\t#{pane_id}"

type tmuxContext struct {
	Session   string
	Window    int
	Pane      int
	PaneID    string
	AgentKind string
}

func currentTmuxContext(ctx context.Context) (tmuxContext, bool) {
	paneID := strings.TrimSpace(os.Getenv("TMUX_PANE"))
	if !validPaneID(paneID) {
		return tmuxContext{}, false
	}

	output, err := exec.CommandContext(
		ctx,
		"tmux", "display-message", "-p", "-t", paneID, tmuxContextFormat,
	).Output()
	if err != nil {
		return tmuxContext{}, false
	}
	current, err := parseTmuxContext(output)
	if err != nil || current.PaneID != paneID {
		return tmuxContext{}, false
	}

	runningAgents, err := agents.List(ctx, "")
	if err == nil {
		for _, agent := range runningAgents {
			if agent.ID == current.PaneID {
				current.AgentKind = agent.Kind
				break
			}
		}
	}
	return current, true
}

func parseTmuxContext(output []byte) (tmuxContext, error) {
	fields := strings.Split(strings.TrimSpace(string(output)), "\t")
	if len(fields) != 4 || fields[0] == "" || !validPaneID(fields[3]) {
		return tmuxContext{}, fmt.Errorf("invalid tmux context")
	}
	window, err := strconv.Atoi(fields[1])
	if err != nil || window < 0 {
		return tmuxContext{}, fmt.Errorf("invalid tmux window")
	}
	pane, err := strconv.Atoi(fields[2])
	if err != nil || pane < 0 {
		return tmuxContext{}, fmt.Errorf("invalid tmux pane")
	}
	return tmuxContext{
		Session: fields[0],
		Window:  window,
		Pane:    pane,
		PaneID:  fields[3],
	}, nil
}

func validPaneID(value string) bool {
	if !strings.HasPrefix(value, "%") || len(value) == 1 {
		return false
	}
	_, err := strconv.ParseUint(value[1:], 10, 64)
	return err == nil
}

func (current tmuxContext) title() string {
	title := fmt.Sprintf(
		"%s · %s:%d.%d",
		agentTitle(current.AgentKind), current.Session, current.Window, current.Pane,
	)
	if len(title) > notifications.MaxTitleBytes {
		return agentTitle(current.AgentKind)
	}
	return title
}

func (current tmuxContext) url() string {
	return "/tmux/" + url.PathEscape(current.Session)
}

func (current tmuxContext) tag() string {
	return "pickle-pane-" + strings.TrimPrefix(current.PaneID, "%")
}

func agentTitle(kind string) string {
	switch kind {
	case "codex":
		return "Codex"
	case "claude":
		return "Claude Code"
	case "opencode":
		return "OpenCode"
	case "pi":
		return "Pi"
	case "hermes":
		return "Hermes"
	default:
		return "Pickle"
	}
}
