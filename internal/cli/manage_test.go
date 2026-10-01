package cli

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func installedFixture(t *testing.T) (string, string) {
	t.Helper()
	home := t.TempDir()
	configDir := filepath.Join(home, ".config/pickle")
	for _, path := range []string{filepath.Join(home, ".local/bin/pickle"), filepath.Join(home, ".config/systemd/user/pickle.service"), filepath.Join(configDir, "config.json"), filepath.Join(configDir, "notifications.json"), filepath.Join(home, "projects/example/file")} {
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte("fixture"), 0644); err != nil {
			t.Fatal(err)
		}
	}
	return home, configDir
}

func TestUninstallPreservesSettingsUnlessPurged(t *testing.T) {
	for _, purge := range []bool{false, true} {
		t.Run(map[bool]string{false: "keep", true: "purge"}[purge], func(t *testing.T) {
			home, configDir := installedFixture(t)
			var calls [][]string
			run := func(_ context.Context, args ...string) (string, error) { calls = append(calls, args); return "", nil }
			var args []string
			if purge {
				args = []string{"--purge"}
			}
			if err := runManage(context.Background(), "uninstall", args, home, configDir, io.Discard, io.Discard, run); err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(calls, [][]string{{"disable", "--now", "pickle.service"}, {"daemon-reload"}}) {
				t.Fatalf("calls: %v", calls)
			}
			for _, path := range []string{filepath.Join(home, ".local/bin/pickle"), filepath.Join(home, ".config/systemd/user/pickle.service")} {
				if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
					t.Fatalf("not removed: %s (%v)", path, err)
				}
			}
			for _, name := range []string{"config.json", "notifications.json"} {
				_, err := os.Stat(filepath.Join(configDir, name))
				if purge && !errors.Is(err, os.ErrNotExist) || !purge && err != nil {
					t.Fatalf("settings purge=%v: %v", purge, err)
				}
			}
			if _, err := os.Stat(filepath.Join(home, "projects/example/file")); err != nil {
				t.Fatal("project removed:", err)
			}
		})
	}
}

func TestUninstallStopsOnServiceFailure(t *testing.T) {
	home, configDir := installedFixture(t)
	run := func(context.Context, ...string) (string, error) {
		return "Failed to connect to bus", errors.New("exit status 1")
	}
	err := runManage(context.Background(), "uninstall", []string{"--purge"}, home, configDir, io.Discard, io.Discard, run)
	if err == nil || !strings.Contains(err.Error(), "Failed to connect to bus") {
		t.Fatalf("error: %v", err)
	}
	for _, path := range []string{filepath.Join(home, ".local/bin/pickle"), filepath.Join(configDir, "config.json")} {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("removed after failure: %s", path)
		}
	}
}

func TestServiceCommands(t *testing.T) {
	for _, command := range []string{"start", "stop", "restart"} {
		t.Run(command, func(t *testing.T) {
			home, configDir := installedFixture(t)
			var calls [][]string
			run := func(_ context.Context, args ...string) (string, error) { calls = append(calls, args); return "", nil }
			var output bytes.Buffer
			if err := runManage(context.Background(), command, nil, home, configDir, &output, io.Discard, run); err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(calls, [][]string{{command, "pickle.service"}}) {
				t.Fatalf("calls: %v", calls)
			}
			if output.Len() == 0 {
				t.Fatal("missing feedback")
			}
		})
	}
}

func TestManageRejectsArgumentsBeforeChangingService(t *testing.T) {
	home, configDir := installedFixture(t)
	run := func(context.Context, ...string) (string, error) {
		t.Fatal("unexpected service mutation")
		return "", nil
	}
	for _, args := range [][]string{{"--unknown"}, {"extra"}} {
		if err := runManage(context.Background(), "uninstall", args, home, configDir, io.Discard, io.Discard, run); err == nil {
			t.Fatalf("accepted %v", args)
		}
	}
}
