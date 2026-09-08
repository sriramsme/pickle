package tmux

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os/exec"
	"sort"
	"strconv"
	"strings"
	"time"
)

const sessionFormat = "#{session_name}\t#{session_windows}\t#{session_attached}\t#{session_activity}"

type Session struct {
	Name         string    `json:"name"`
	Windows      int       `json:"windows"`
	Attached     int       `json:"attached"`
	LastActivity time.Time `json:"lastActivity"`
}

func ListSessions(ctx context.Context) ([]Session, error) {
	output, err := exec.CommandContext(ctx, "tmux", "list-sessions", "-F", sessionFormat).CombinedOutput()
	if err != nil {
		var exitErr *exec.ExitError
		if !isNoServerOutput(output) || !errors.As(err, &exitErr) {
			return nil, fmt.Errorf("list tmux sessions: %w", err)
		}
		return []Session{}, nil
	}

	sessions, err := parseSessions(output)
	if err != nil {
		return nil, err
	}
	sort.Slice(sessions, func(i, j int) bool {
		if sessions[i].LastActivity.Equal(sessions[j].LastActivity) {
			return sessions[i].Name < sessions[j].Name
		}
		return sessions[i].LastActivity.After(sessions[j].LastActivity)
	})
	return sessions, nil
}

func isNoServerOutput(output []byte) bool {
	return bytes.Contains(output, []byte("no server running")) ||
		(bytes.Contains(output, []byte("error connecting to")) &&
			bytes.Contains(output, []byte("No such file or directory")))
}

func SessionExists(ctx context.Context, name string) (bool, error) {
	sessions, err := ListSessions(ctx)
	if err != nil {
		return false, err
	}
	for _, session := range sessions {
		if session.Name == name {
			return true, nil
		}
	}
	return false, nil
}

func parseSessions(output []byte) ([]Session, error) {
	text := strings.TrimSpace(string(output))
	if text == "" {
		return []Session{}, nil
	}

	lines := strings.Split(text, "\n")
	sessions := make([]Session, 0, len(lines))
	for _, line := range lines {
		fields := strings.Split(line, "\t")
		if len(fields) != 4 || fields[0] == "" {
			return nil, fmt.Errorf("parse tmux session: invalid output %q", line)
		}

		windows, err := strconv.Atoi(fields[1])
		if err != nil {
			return nil, fmt.Errorf("parse tmux window count: %w", err)
		}
		attached, err := strconv.Atoi(fields[2])
		if err != nil {
			return nil, fmt.Errorf("parse tmux attached count: %w", err)
		}
		activity, err := strconv.ParseInt(fields[3], 10, 64)
		if err != nil {
			return nil, fmt.Errorf("parse tmux activity: %w", err)
		}

		sessions = append(sessions, Session{
			Name:         fields[0],
			Windows:      windows,
			Attached:     attached,
			LastActivity: time.Unix(activity, 0),
		})
	}
	return sessions, nil
}
