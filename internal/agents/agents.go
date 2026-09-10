package agents

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

const paneFormat = "#{session_name}\t#{window_index}\t#{pane_index}\t#{pane_id}\t#{pane_pid}\t#{pane_current_command}\t#{pane_current_path}"

type Agent struct {
	ID      string `json:"id"`
	Kind    string `json:"kind"`
	Project string `json:"project,omitempty"`
	Session string `json:"session"`
	Window  int    `json:"window"`
	Pane    int    `json:"pane"`
}

type tmuxPane struct {
	ID             string
	Session        string
	Window         int
	Pane           int
	PID            int
	CurrentCommand string
	CurrentPath    string
}

type process struct {
	PID       int
	ParentPID int
	Name      string
	Arguments []string
}

func List(ctx context.Context, projectsDirectory string) ([]Agent, error) {
	output, err := exec.CommandContext(ctx, "tmux", "list-panes", "-a", "-F", paneFormat).CombinedOutput()
	if err != nil {
		if noTmuxServer(output) {
			return []Agent{}, nil
		}
		return nil, fmt.Errorf("list tmux panes: %w", err)
	}

	panes, err := parsePanes(output)
	if err != nil {
		return nil, err
	}
	if len(panes) == 0 {
		return []Agent{}, nil
	}
	processes, err := readProcesses("/proc")
	if err != nil {
		return nil, err
	}
	return detect(panes, processes, projectsDirectory), nil
}

func parsePanes(output []byte) ([]tmuxPane, error) {
	text := strings.TrimSpace(string(output))
	if text == "" {
		return []tmuxPane{}, nil
	}

	lines := strings.Split(text, "\n")
	panes := make([]tmuxPane, 0, len(lines))
	for _, line := range lines {
		fields := strings.Split(line, "\t")
		if len(fields) != 7 || fields[0] == "" || fields[3] == "" {
			return nil, fmt.Errorf("parse tmux pane: invalid output %q", line)
		}
		window, err := strconv.Atoi(fields[1])
		if err != nil {
			return nil, fmt.Errorf("parse tmux window: %w", err)
		}
		pane, err := strconv.Atoi(fields[2])
		if err != nil {
			return nil, fmt.Errorf("parse tmux pane index: %w", err)
		}
		pid, err := strconv.Atoi(fields[4])
		if err != nil {
			return nil, fmt.Errorf("parse tmux pane PID: %w", err)
		}
		panes = append(panes, tmuxPane{
			ID:             fields[3],
			Session:        fields[0],
			Window:         window,
			Pane:           pane,
			PID:            pid,
			CurrentCommand: fields[5],
			CurrentPath:    fields[6],
		})
	}
	return panes, nil
}

func readProcesses(procRoot string) ([]process, error) {
	entries, err := os.ReadDir(procRoot)
	if err != nil {
		return nil, fmt.Errorf("read processes: %w", err)
	}

	processes := make([]process, 0)
	for _, entry := range entries {
		pid, err := strconv.Atoi(entry.Name())
		if err != nil || !entry.IsDir() {
			continue
		}
		processDirectory := filepath.Join(procRoot, entry.Name())
		stat, err := os.ReadFile(filepath.Join(processDirectory, "stat"))
		if err != nil {
			continue
		}
		parsed, err := parseProcessStat(stat)
		if err != nil {
			continue
		}
		parsed.PID = pid
		if commandLine, err := os.ReadFile(filepath.Join(processDirectory, "cmdline")); err == nil {
			parsed.Arguments = parseCommandLine(commandLine)
		}
		processes = append(processes, parsed)
	}
	return processes, nil
}

func parseProcessStat(data []byte) (process, error) {
	text := strings.TrimSpace(string(data))
	open := strings.IndexByte(text, '(')
	close := strings.LastIndexByte(text, ')')
	if open < 1 || close <= open || close+1 >= len(text) {
		return process{}, errors.New("invalid process stat")
	}
	fields := strings.Fields(text[close+1:])
	if len(fields) < 2 {
		return process{}, errors.New("invalid process stat")
	}
	parentPID, err := strconv.Atoi(fields[1])
	if err != nil {
		return process{}, err
	}
	return process{
		ParentPID: parentPID,
		Name:      text[open+1 : close],
	}, nil
}

func parseCommandLine(data []byte) []string {
	parts := strings.Split(strings.TrimRight(string(data), "\x00"), "\x00")
	if len(parts) == 1 && parts[0] == "" {
		return nil
	}
	return parts
}

func detect(panes []tmuxPane, processes []process, projectsDirectory string) []Agent {
	byPID := make(map[int]process, len(processes))
	children := make(map[int][]process)
	for _, candidate := range processes {
		byPID[candidate.PID] = candidate
		children[candidate.ParentPID] = append(children[candidate.ParentPID], candidate)
	}

	agents := make([]Agent, 0)
	for _, pane := range panes {
		kind := identifyName(pane.CurrentCommand)
		if kind == "" {
			if paneProcess, ok := byPID[pane.PID]; ok {
				kind = identifyProcess(paneProcess)
			}
		}
		if kind == "" {
			kind = identifyDescendants(pane.PID, children)
		}
		if kind == "" {
			continue
		}
		agents = append(agents, Agent{
			ID:      pane.ID,
			Kind:    kind,
			Project: projectName(projectsDirectory, pane.CurrentPath),
			Session: pane.Session,
			Window:  pane.Window,
			Pane:    pane.Pane,
		})
	}

	sort.Slice(agents, func(i, j int) bool {
		if agents[i].Session != agents[j].Session {
			return strings.ToLower(agents[i].Session) < strings.ToLower(agents[j].Session)
		}
		if agents[i].Window != agents[j].Window {
			return agents[i].Window < agents[j].Window
		}
		return agents[i].Pane < agents[j].Pane
	})
	return agents
}

func identifyDescendants(rootPID int, children map[int][]process) string {
	pending := append([]process(nil), children[rootPID]...)
	seen := map[int]struct{}{rootPID: {}}
	for len(pending) > 0 {
		candidate := pending[0]
		pending = pending[1:]
		if _, ok := seen[candidate.PID]; ok {
			continue
		}
		seen[candidate.PID] = struct{}{}
		if kind := identifyProcess(candidate); kind != "" {
			return kind
		}
		pending = append(pending, children[candidate.PID]...)
	}
	return ""
}

func identifyProcess(candidate process) string {
	if kind := identifyName(candidate.Name); kind != "" {
		return kind
	}
	if len(candidate.Arguments) == 0 {
		return ""
	}
	if kind := identifyName(candidate.Arguments[0]); kind != "" {
		return kind
	}

	runtime := executableName(candidate.Arguments[0])
	program := wrappedProgram(runtime, candidate.Arguments[1:])
	return identifyProgram(program)
}

func wrappedProgram(runtime string, arguments []string) string {
	switch runtime {
	case "node", "bun":
		for _, argument := range arguments {
			if argument == "-e" || argument == "--eval" || argument == "-p" || argument == "--print" {
				return ""
			}
			if !strings.HasPrefix(argument, "-") {
				return argument
			}
		}
	case "python", "python3":
		for index := 0; index < len(arguments); index++ {
			argument := arguments[index]
			if argument == "-c" {
				return ""
			}
			if argument == "-m" && index+1 < len(arguments) {
				return arguments[index+1]
			}
			if !strings.HasPrefix(argument, "-") {
				return argument
			}
		}
	}
	return ""
}

func identifyProgram(program string) string {
	if kind := identifyName(program); kind != "" {
		return kind
	}
	normalized := strings.ToLower(filepath.ToSlash(program))
	switch {
	case strings.Contains(normalized, "/node_modules/@openai/codex/"):
		return "codex"
	case strings.Contains(normalized, "/node_modules/@anthropic-ai/claude-code/"):
		return "claude"
	case strings.Contains(normalized, "/node_modules/@mariozechner/pi-coding-agent/"):
		return "pi"
	case strings.Contains(normalized, "hermes_cli"):
		return "hermes"
	default:
		return ""
	}
}

func identifyName(value string) string {
	name := executableName(value)
	name = strings.TrimSuffix(name, ".js")
	name = strings.TrimSuffix(name, ".mjs")
	switch name {
	case "codex":
		return "codex"
	case "claude", "claude-code":
		return "claude"
	case "opencode", "opencode2", "open-code":
		return "opencode"
	case "pi":
		return "pi"
	case "hermes", "hermes-agent":
		return "hermes"
	default:
		return ""
	}
}

func executableName(value string) string {
	name := strings.ToLower(filepath.Base(strings.TrimSpace(value)))
	for _, suffix := range []string{".exe", ".cmd", ".bat"} {
		name = strings.TrimSuffix(name, suffix)
	}
	return name
}

func projectName(projectsDirectory, currentPath string) string {
	relative, err := filepath.Rel(filepath.Clean(projectsDirectory), filepath.Clean(currentPath))
	if err != nil || relative == "." || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return ""
	}
	return strings.Split(relative, string(filepath.Separator))[0]
}

func noTmuxServer(output []byte) bool {
	return strings.Contains(string(output), "no server running") ||
		(strings.Contains(string(output), "error connecting to") && strings.Contains(string(output), "No such file or directory"))
}
