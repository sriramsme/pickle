package cli

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/sriramsme/pickle/internal/config"
)

func RunManage(ctx context.Context, command string, args []string, output, errorOutput io.Writer) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	return runManage(ctx, command, args, home, filepath.Dir(config.DefaultPath(home)), output, errorOutput, runSystemctl)
}

func runSystemctl(ctx context.Context, args ...string) (string, error) {
	data, err := exec.CommandContext(ctx, "systemctl", append([]string{"--user"}, args...)...).CombinedOutput()
	return strings.TrimSpace(string(data)), err
}

func runManage(ctx context.Context, command string, args []string, home, configDir string, output, errorOutput io.Writer, run func(context.Context, ...string) (string, error)) error {
	flags := flag.NewFlagSet(command, flag.ContinueOnError)
	flags.SetOutput(errorOutput)
	var purge bool
	if command == "uninstall" {
		flags.BoolVar(&purge, "purge", false, "also remove saved settings and notification subscriptions")
	}
	flags.Usage = func() {
		fmt.Fprintf(errorOutput, "Usage: pickle %s", command)
		if command == "uninstall" {
			fmt.Fprint(errorOutput, " [--purge]")
		}
		fmt.Fprintln(errorOutput)
		flags.PrintDefaults()
	}
	if err := flags.Parse(args); err != nil {
		return err
	}
	if flags.NArg() != 0 {
		return fmt.Errorf("unexpected argument: %s", flags.Arg(0))
	}
	ctx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()

	switch command {
	case "start", "stop", "restart":
		if _, err := os.Stat(filepath.Join(home, ".config/systemd/user/pickle.service")); err != nil {
			return errors.New("Pickle's user service is not installed; run the installer, or use pickle to run in the foreground")
		}
		if data, err := run(ctx, command, "pickle.service"); err != nil {
			return serviceError(data, err)
		}
		if command == "stop" {
			fmt.Fprintln(output, "Pickle stopped.")
		} else {
			fmt.Fprintln(output, "Pickle is running.")
			fmt.Fprintln(output, "Open http://127.0.0.1:8080")
		}
	case "status":
		state, err := run(ctx, "is-active", "pickle.service")
		if err != nil && state != "inactive" && state != "failed" && state != "unknown" {
			fmt.Fprintln(output, "Pickle's user service status is unavailable.")
		} else if state == "active" {
			fmt.Fprintln(output, "Pickle's service is running.")
		} else {
			fmt.Fprintln(output, "Pickle's service is stopped or not installed.")
		}
		if serverReachable(ctx) {
			fmt.Fprintln(output, "Open http://127.0.0.1:8080")
		} else {
			fmt.Fprintln(output, "No Pickle server responded at http://127.0.0.1:8080")
		}
	case "doctor":
		missing := false
		for _, name := range []string{"tmux", "systemctl", "tailscale"} {
			if _, err := exec.LookPath(name); err != nil {
				fmt.Fprintf(output, "%s: not installed\n", name)
				if name == "tmux" {
					missing = true
				}
			} else {
				fmt.Fprintf(output, "%s: installed\n", name)
			}
		}
		settings, err := config.Load(filepath.Join(configDir, "config.json"), filepath.Join(home, "projects"), "")
		if err != nil {
			fmt.Fprintf(output, "Settings: %v\n", err)
			missing = true
		} else if !settings.Current().Configured {
			fmt.Fprintln(output, "Settings: finish setup at http://127.0.0.1:8080/setup")
		} else if info, err := os.Stat(settings.ProjectsDirectory()); err != nil || !info.IsDir() {
			fmt.Fprintln(output, "Projects folder: unavailable; choose a folder in Settings")
			missing = true
		} else {
			fmt.Fprintln(output, "Projects folder: available")
		}
		if serverReachable(ctx) {
			fmt.Fprintln(output, "Server: responding at http://127.0.0.1:8080")
		} else {
			fmt.Fprintln(output, "Server: not responding at http://127.0.0.1:8080")
			missing = true
		}
		if path, err := exec.LookPath("tailscale"); err == nil {
			data, err := exec.CommandContext(ctx, path, "serve", "status").CombinedOutput()
			if err != nil {
				fmt.Fprintln(output, "Tailscale Serve: unavailable; check that Tailscale is running")
			} else {
				fmt.Fprintf(output, "Tailscale Serve:\n%s\n", strings.TrimSpace(string(data)))
			}
		}
		if missing {
			return errors.New("fix the items above and run pickle doctor again")
		}
	case "uninstall":
		servicePath := filepath.Join(home, ".config/systemd/user/pickle.service")
		if _, err := os.Stat(servicePath); err == nil {
			if data, err := run(ctx, "disable", "--now", "pickle.service"); err != nil {
				return serviceError(data, err)
			}
			if err := os.Remove(servicePath); err != nil {
				return err
			}
			if data, err := run(ctx, "daemon-reload"); err != nil {
				return serviceError(data, err)
			}
		} else if !errors.Is(err, os.ErrNotExist) {
			return err
		}
		if purge {
			if err := os.RemoveAll(configDir); err != nil {
				return err
			}
		}
		if err := os.Remove(filepath.Join(home, ".local/bin/pickle")); err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
		fmt.Fprintln(output, "Pickle uninstalled.")
		if !purge {
			fmt.Fprintln(output, "Saved settings kept. tmux sessions and Tailscale routes are unchanged.")
		} else {
			fmt.Fprintln(output, "Saved settings removed. tmux sessions and Tailscale routes are unchanged.")
		}
	}
	return nil
}

func serviceError(data string, err error) error {
	if data == "" {
		return fmt.Errorf("manage Pickle's user service: %w", err)
	}
	return fmt.Errorf("manage Pickle's user service: %s", data)
}

func serverReachable(ctx context.Context) bool {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, defaultServerURL+"/api/settings", nil)
	if err != nil {
		return false
	}
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return false
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return false
	}
	var settings struct {
		Configured *bool `json:"configured"`
	}
	return json.NewDecoder(io.LimitReader(response.Body, 64<<10)).Decode(&settings) == nil && settings.Configured != nil
}
