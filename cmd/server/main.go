package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/sriramsme/pickle/internal/cli"
	"github.com/sriramsme/pickle/internal/config"
	"github.com/sriramsme/pickle/internal/notifications"
	"github.com/sriramsme/pickle/internal/server"
)

func main() {
	args := os.Args[1:]
	if len(args) == 2 && args[0] == "help" {
		args = []string{args[1], "--help"}
	}
	if len(args) > 0 {
		switch args[0] {
		case "start", "stop", "restart", "status", "doctor", "uninstall":
			if err := cli.RunManage(context.Background(), args[0], args[1:], os.Stdout, os.Stderr); err != nil && !errors.Is(err, flag.ErrHelp) {
				fmt.Fprintf(os.Stderr, "pickle %s: %v\n", args[0], err)
				os.Exit(1)
			}
			return
		}
	}
	if len(args) > 0 && args[0] == "notify" {
		if err := cli.RunNotify(context.Background(), args[1:], os.Stdin, os.Stdout, os.Stderr); err != nil {
			if errors.Is(err, flag.ErrHelp) {
				return
			}
			fmt.Fprintf(os.Stderr, "pickle notify: %v\n", err)
			os.Exit(1)
		}
		return
	}

	home, err := os.UserHomeDir()
	if err != nil {
		log.Fatal(err)
	}

	addr := flag.String("addr", "127.0.0.1:8080", "HTTP listen address")
	configPath := flag.String("config", config.DefaultPath(home), "configuration file")
	projectsDir := flag.String("projects-dir", "", "override the configured projects directory")
	flag.Usage = func() {
		fmt.Fprintln(flag.CommandLine.Output(), "Pickle is a self-hosted browser terminal for this Linux machine.")
		fmt.Fprintln(flag.CommandLine.Output(), "It connects your devices to persistent tmux sessions over a private network.")
		fmt.Fprintln(flag.CommandLine.Output(), "Use pickle notify to send notifications to devices enabled in Pickle's Settings.")
		fmt.Fprintln(flag.CommandLine.Output())
		fmt.Fprintln(flag.CommandLine.Output(), "Usage: pickle [server options]")
		fmt.Fprintln(flag.CommandLine.Output(), "       pickle notify [options] <message>")
		fmt.Fprintln(flag.CommandLine.Output(), "       pickle <command> [options]")
		fmt.Fprintln(flag.CommandLine.Output(), "       pickle help <command>")
		fmt.Fprintln(flag.CommandLine.Output(), "\nCommands:")
		fmt.Fprintln(flag.CommandLine.Output(), "  start / stop / restart  Manage the installed user service")
		fmt.Fprintln(flag.CommandLine.Output(), "  status                  Show service and server status")
		fmt.Fprintln(flag.CommandLine.Output(), "  doctor                  Check setup and connectivity")
		fmt.Fprintln(flag.CommandLine.Output(), "  notify                  Send a notification to your devices")
		fmt.Fprintln(flag.CommandLine.Output(), "  uninstall [--purge]     Remove Pickle; keep settings by default")
		fmt.Fprintln(flag.CommandLine.Output())
		flag.PrintDefaults()
	}
	if len(args) > 0 && args[0] == "help" {
		flag.Usage()
		return
	}
	if err := flag.CommandLine.Parse(args); err != nil {
		os.Exit(2)
	}
	if flag.NArg() != 0 {
		fmt.Fprintf(os.Stderr, "Unknown command: %s. Run pickle help for available commands.\n", flag.Arg(0))
		os.Exit(1)
	}

	settings, err := config.Load(*configPath, filepath.Join(home, "projects"), *projectsDir)
	if err != nil {
		log.Fatal(err)
	}
	notificationStore, err := notifications.Load(filepath.Join(filepath.Dir(*configPath), "notifications.json"))
	if err != nil {
		log.Fatal(err)
	}

	httpServer := &http.Server{
		Addr:              *addr,
		Handler:           server.New(settings, notificationStore),
		ReadHeaderTimeout: 5 * time.Second,
	}

	log.Printf("pickle listening on http://%s", *addr)
	if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}
}
