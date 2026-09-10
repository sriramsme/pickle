package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/sriramsme/pickle/internal/notifications"
)

const defaultServerURL = "http://127.0.0.1:8080"

type sendRequest struct {
	Title   string       `json:"title"`
	Body    string       `json:"body"`
	URL     string       `json:"url"`
	Tag     string       `json:"tag,omitempty"`
	Urgency string       `json:"urgency"`
	Context *sendContext `json:"context,omitempty"`
}

type sendContext struct {
	PaneID  string `json:"paneId"`
	Kind    string `json:"kind,omitempty"`
	Session string `json:"session"`
	Window  int    `json:"window"`
	Pane    int    `json:"pane"`
}

type sendResponse struct {
	Sent int `json:"sent"`
}

func RunNotify(ctx context.Context, args []string, input io.Reader, output, errorOutput io.Writer) error {
	flags := flag.NewFlagSet("notify", flag.ContinueOnError)
	flags.SetOutput(errorOutput)
	title := flags.String("title", "Pickle", "notification title; defaults to tmux context when available")
	destination := flags.String("url", "/", "local Pickle path; defaults to the current tmux session")
	tag := flags.String("tag", "", "replacement tag; defaults to the current tmux pane")
	urgency := flags.String("urgency", string(notifications.UrgencyNormal), "delivery urgency: low, normal, or high")
	serverURL := flags.String("server", serverURLFromEnvironment(), "running Pickle server URL")
	readStdin := flags.Bool("stdin", false, "read the message from standard input")
	jsonOutput := flags.Bool("json", false, "print a machine-readable result")
	timeout := flags.Duration("timeout", 15*time.Second, "maximum delivery time")
	flags.Usage = func() {
		fmt.Fprintln(errorOutput, "Usage: pickle notify [options] <message>")
		fmt.Fprintln(errorOutput)
		flags.PrintDefaults()
	}
	if err := flags.Parse(args); err != nil {
		return err
	}
	if *readStdin && flags.NArg() != 0 {
		return errors.New("message arguments cannot be used with --stdin")
	}
	if *timeout <= 0 {
		return errors.New("timeout must be greater than zero")
	}
	message, err := notificationMessage(flags.Args(), *readStdin, input)
	if err != nil {
		return err
	}

	explicit := make(map[string]bool)
	flags.Visit(func(option *flag.Flag) {
		explicit[option.Name] = true
	})
	lookupContext, cancel := context.WithTimeout(ctx, time.Second)
	tmuxContext, hasTmuxContext := currentTmuxContext(lookupContext)
	cancel()
	if hasTmuxContext {
		if !explicit["title"] {
			*title = tmuxContext.title()
		}
		if !explicit["url"] {
			*destination = tmuxContext.url()
		}
		if !explicit["tag"] {
			*tag = tmuxContext.tag()
		}
	}

	notification := notifications.Notification{
		Title:   strings.TrimSpace(*title),
		Body:    message,
		URL:     strings.TrimSpace(*destination),
		Tag:     strings.TrimSpace(*tag),
		Urgency: notifications.Urgency(strings.TrimSpace(*urgency)),
	}
	if err := notifications.Validate(notification); err != nil {
		return err
	}

	endpoint, err := notificationEndpoint(*serverURL)
	if err != nil {
		return err
	}
	payload, err := json.Marshal(sendRequest{
		Title:   notification.Title,
		Body:    notification.Body,
		URL:     notification.URL,
		Tag:     notification.Tag,
		Urgency: string(notification.Urgency),
		Context: tmuxContext.sendContext(hasTmuxContext),
	})
	if err != nil {
		return fmt.Errorf("encode notification: %w", err)
	}

	requestContext, cancel := context.WithTimeout(ctx, *timeout)
	defer cancel()
	request, err := http.NewRequestWithContext(requestContext, http.MethodPost, endpoint.String(), bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("create notification request: %w", err)
	}
	request.Header.Set("Content-Type", "application/json")

	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return fmt.Errorf("contact Pickle server: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		message, _ := io.ReadAll(io.LimitReader(response.Body, 4<<10))
		detail := strings.TrimSpace(string(message))
		if detail == "" {
			detail = response.Status
		}
		return fmt.Errorf("Pickle server: %s", detail)
	}

	var result sendResponse
	decoder := json.NewDecoder(io.LimitReader(response.Body, 4<<10))
	if err := decoder.Decode(&result); err != nil {
		return fmt.Errorf("decode Pickle response: %w", err)
	}
	if *jsonOutput {
		return json.NewEncoder(output).Encode(result)
	}
	device := "devices"
	if result.Sent == 1 {
		device = "device"
	}
	_, err = fmt.Fprintf(output, "Notification sent to %d %s.\n", result.Sent, device)
	return err
}

func notificationMessage(arguments []string, readStdin bool, input io.Reader) (string, error) {
	if readStdin {
		data, err := io.ReadAll(io.LimitReader(input, notifications.MaxBodyBytes+1))
		if err != nil {
			return "", fmt.Errorf("read notification message: %w", err)
		}
		if len(data) > notifications.MaxBodyBytes {
			return "", fmt.Errorf("%w: message is too long", notifications.ErrInvalidNotification)
		}
		return strings.TrimSpace(string(data)), nil
	}
	if len(arguments) == 0 {
		return "", errors.New("notification message is required")
	}
	return strings.TrimSpace(strings.Join(arguments, " ")), nil
}

func notificationEndpoint(value string) (*url.URL, error) {
	value = strings.TrimSpace(value)
	if !strings.Contains(value, "://") {
		value = "http://" + value
	}
	base, err := url.Parse(value)
	if err != nil || (base.Scheme != "http" && base.Scheme != "https") || base.Host == "" || base.User != nil {
		return nil, errors.New("server must be an HTTP or HTTPS URL")
	}
	base.Path = strings.TrimRight(base.Path, "/") + "/api/notifications"
	base.RawPath = ""
	base.RawQuery = ""
	base.Fragment = ""
	return base, nil
}

func serverURLFromEnvironment() string {
	if value := strings.TrimSpace(os.Getenv("PICKLE_URL")); value != "" {
		return value
	}
	return defaultServerURL
}
