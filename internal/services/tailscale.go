package services

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/url"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

type tailscaleServeStatus struct {
	Web map[string]struct {
		Handlers map[string]struct {
			Proxy string `json:"Proxy"`
		} `json:"Handlers"`
	} `json:"Web"`
}

type tailscaleExposure struct {
	URL  string
	Port int
}

func markTailscaleExposures(ctx context.Context, services []Service) {
	tailscalePath, err := exec.LookPath("tailscale")
	if err != nil {
		return
	}

	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	output, err := exec.CommandContext(ctx, tailscalePath, "serve", "status", "--json").Output()
	if err != nil {
		return
	}

	exposures, err := parseTailscaleServeStatus(output)
	if err != nil {
		return
	}
	for serviceIndex := range services {
		for portIndex := range services[serviceIndex].Ports {
			port := &services[serviceIndex].Ports[portIndex]
			exposure := exposures[port.Host]
			port.ExposedURL = exposure.URL
			port.ExposedPort = exposure.Port
		}
	}
}

func parseTailscaleServeStatus(data []byte) (map[int]tailscaleExposure, error) {
	var status tailscaleServeStatus
	if err := json.Unmarshal(data, &status); err != nil {
		return nil, fmt.Errorf("parse tailscale serve status: %w", err)
	}

	exposures := make(map[int]tailscaleExposure)
	for address, web := range status.Web {
		for path, handler := range web.Handlers {
			proxyURL, err := url.Parse(handler.Proxy)
			if err != nil {
				continue
			}
			port, err := strconv.Atoi(proxyURL.Port())
			if err != nil {
				continue
			}
			host := address
			exposedPort := 443
			if _, portText, err := net.SplitHostPort(address); err == nil {
				exposedPort, err = strconv.Atoi(portText)
				if err != nil {
					continue
				}
			}
			if exposedPort == 443 {
				host = strings.TrimSuffix(address, ":443")
			}
			exposures[port] = tailscaleExposure{URL: "https://" + host + path, Port: exposedPort}
		}
	}
	return exposures, nil
}

func setTailscaleServe(ctx context.Context, port int, enabled bool) error {
	tailscalePath, err := exec.LookPath("tailscale")
	if err != nil {
		return fmt.Errorf("tailscale is not installed")
	}

	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	if enabled {
		output, err := exec.CommandContext(ctx, tailscalePath, "serve", "status", "--json").Output()
		if err == nil {
			exposures, parseErr := parseTailscaleServeStatus(output)
			if parseErr == nil && tailscalePortInUse(exposures, port, port) {
				return fmt.Errorf("Tailscale HTTPS port %d is already in use", port)
			}
		}
	}
	args := []string{"serve", "--yes", fmt.Sprintf("--https=%d", port)}
	if enabled {
		args = append(args, "--bg", fmt.Sprintf("http://localhost:%d", port))
	} else {
		args = append(args, "off")
	}
	output, err := exec.CommandContext(ctx, tailscalePath, args...).CombinedOutput()
	if err != nil {
		message := strings.TrimSpace(string(output))
		if strings.Contains(message, "Access denied: serve config denied") {
			return fmt.Errorf("Tailscale Serve needs operator access. Run: sudo tailscale set --operator=$USER")
		}
		if message == "" {
			message = err.Error()
		}
		return fmt.Errorf("tailscale serve: %s", message)
	}
	return nil
}

func tailscalePortInUse(exposures map[int]tailscaleExposure, externalPort, localPort int) bool {
	for targetPort, exposure := range exposures {
		if exposure.Port == externalPort && targetPort != localPort {
			return true
		}
	}
	return false
}
