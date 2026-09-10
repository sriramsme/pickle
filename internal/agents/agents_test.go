package agents

import (
	"reflect"
	"testing"
)

func TestParsePanes(t *testing.T) {
	panes, err := parsePanes([]byte("work\t1\t2\t%7\t4200\tnode\t/home/user/projects/pickle/internal\n"))
	if err != nil {
		t.Fatalf("parse panes: %v", err)
	}
	if len(panes) != 1 || panes[0].ID != "%7" || panes[0].PID != 4200 || panes[0].Window != 1 || panes[0].Pane != 2 {
		t.Fatalf("unexpected panes: %+v", panes)
	}
}

func TestParseProcessStatHandlesSpacesInName(t *testing.T) {
	process, err := parseProcessStat([]byte("4200 (agent worker) S 4100 4200 4100 34816 4200 0 0"))
	if err != nil {
		t.Fatalf("parse process stat: %v", err)
	}
	if process.Name != "agent worker" || process.ParentPID != 4100 {
		t.Fatalf("unexpected process: %+v", process)
	}
}

func TestDetectFindsDirectAndWrappedAgents(t *testing.T) {
	panes := []tmuxPane{
		{ID: "%1", Session: "pickle", PID: 100, CurrentCommand: "opencode", CurrentPath: "/home/user/projects/pickle"},
		{ID: "%2", Session: "work", Window: 1, PID: 200, CurrentCommand: "node", CurrentPath: "/home/user/projects/site/src"},
		{ID: "%3", Session: "shell", PID: 300, CurrentCommand: "bash", CurrentPath: "/home/user"},
	}
	processes := []process{
		{PID: 200},
		{PID: 210, ParentPID: 200, Name: "node", Arguments: []string{"node", "/modules/node_modules/@openai/codex/bin/codex.js"}},
		{PID: 300, Name: "bash", Arguments: []string{"bash"}},
	}

	got := detect(panes, processes, "/home/user/projects")
	want := []Agent{
		{ID: "%1", Kind: "opencode", Project: "pickle", Session: "pickle"},
		{ID: "%2", Kind: "codex", Project: "site", Session: "work", Window: 1},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected agents:\n got: %+v\nwant: %+v", got, want)
	}
}

func TestDetectKeepsAgentWhileItRunsAChildProcess(t *testing.T) {
	panes := []tmuxPane{{ID: "%1", Session: "work", PID: 100, CurrentCommand: "bash"}}
	processes := []process{
		{PID: 100, Name: "bash"},
		{PID: 110, ParentPID: 100, Name: "node", Arguments: []string{"node", "/node_modules/@anthropic-ai/claude-code/cli.js"}},
		{PID: 120, ParentPID: 110, Name: "bash", Arguments: []string{"bash", "-c", "go test ./..."}},
	}

	agents := detect(panes, processes, "/projects")
	if len(agents) != 1 || agents[0].Kind != "claude" {
		t.Fatalf("unexpected agents: %+v", agents)
	}
}

func TestIdentifyProcessRejectsEvalText(t *testing.T) {
	process := process{Name: "node", Arguments: []string{"node", "--eval", "console.log('codex')"}}
	if kind := identifyProcess(process); kind != "" {
		t.Fatalf("unexpected agent: %q", kind)
	}
}

func TestIdentifyName(t *testing.T) {
	tests := map[string]string{
		"codex":        "codex",
		"claude-code":  "claude",
		"opencode2":    "opencode",
		"pi":           "pi",
		"hermes-agent": "hermes",
		"bash":         "",
	}
	for name, want := range tests {
		t.Run(name, func(t *testing.T) {
			if got := identifyName(name); got != want {
				t.Fatalf("identifyName(%q) = %q, want %q", name, got, want)
			}
		})
	}
}

func TestProjectNameRejectsSiblingPrefix(t *testing.T) {
	if project := projectName("/home/user/projects", "/home/user/projects-old/pickle"); project != "" {
		t.Fatalf("unexpected project: %q", project)
	}
}
