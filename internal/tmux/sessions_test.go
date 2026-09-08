package tmux

import "testing"

func TestParseSessions(t *testing.T) {
	sessions, err := parseSessions([]byte(
		"pickle\t2\t1\t1700000000\nwork\t4\t0\t1700000100\n",
	))
	if err != nil {
		t.Fatalf("parse sessions: %v", err)
	}
	if len(sessions) != 2 {
		t.Fatalf("unexpected session count: %d", len(sessions))
	}
	if sessions[0].Name != "pickle" || sessions[0].Windows != 2 || sessions[0].Attached != 1 {
		t.Fatalf("unexpected first session: %+v", sessions[0])
	}
	if sessions[1].LastActivity.Unix() != 1700000100 {
		t.Fatalf("unexpected activity: %v", sessions[1].LastActivity)
	}
}

func TestParseSessionsRejectsInvalidOutput(t *testing.T) {
	if _, err := parseSessions([]byte("pickle\tnot-a-number\t1\t1700000000\n")); err == nil {
		t.Fatal("expected an error")
	}
}

func TestIsNoServerOutput(t *testing.T) {
	for _, output := range [][]byte{
		[]byte("no server running on /tmp/tmux/default"),
		[]byte("error connecting to /tmp/tmux/default (No such file or directory)"),
	} {
		if !isNoServerOutput(output) {
			t.Fatalf("expected no-server output: %q", output)
		}
	}

	if isNoServerOutput([]byte("error connecting to /tmp/tmux/default (Permission denied)")) {
		t.Fatal("unexpected no-server match")
	}
}
