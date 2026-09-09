package cli

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRunNotifySendsConfiguredNotification(t *testing.T) {
	var received sendRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/notifications" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
			return
		}
		if err := json.NewDecoder(r.Body).Decode(&received); err != nil {
			t.Errorf("decode request: %v", err)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"sent":2}`))
	}))
	defer server.Close()

	var output bytes.Buffer
	err := RunNotify(t.Context(), []string{
		"--server", server.URL,
		"--title", "Codex",
		"--url", "/tmux/pickle",
		"--tag", "codex-pickle",
		"--urgency", "high",
		"Finished the task",
	}, strings.NewReader(""), &output, &bytes.Buffer{})
	if err != nil {
		t.Fatalf("run notify: %v", err)
	}
	if received.Title != "Codex" || received.Body != "Finished the task" || received.URL != "/tmux/pickle" || received.Tag != "codex-pickle" || received.Urgency != "high" {
		t.Fatalf("unexpected notification: %+v", received)
	}
	if output.String() != "Notification sent to 2 devices.\n" {
		t.Fatalf("unexpected output: %q", output.String())
	}
}

func TestRunNotifyReadsStdinAndWritesJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"sent":1}`))
	}))
	defer server.Close()

	var output bytes.Buffer
	err := RunNotify(
		t.Context(),
		[]string{"--server", server.URL, "--stdin", "--json"},
		strings.NewReader("  Finished from stdin\n"),
		&output,
		&bytes.Buffer{},
	)
	if err != nil {
		t.Fatalf("run notify: %v", err)
	}
	if output.String() != "{\"sent\":1}\n" {
		t.Fatalf("unexpected output: %q", output.String())
	}
}

func TestRunNotifyRejectsExternalDestination(t *testing.T) {
	err := RunNotify(
		t.Context(),
		[]string{"--url", "https://example.com", "Finished"},
		strings.NewReader(""),
		&bytes.Buffer{},
		&bytes.Buffer{},
	)
	if err == nil || !strings.Contains(err.Error(), "local Pickle path") {
		t.Fatalf("unexpected error: %v", err)
	}
}
