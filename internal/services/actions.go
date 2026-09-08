package services

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"syscall"
	"time"
)

var (
	ErrServiceNotFound = errors.New("service not found")
	ErrInvalidAction   = errors.New("invalid service action")
)

type ActionRequest struct {
	ID     string `json:"id"`
	Action string `json:"action"`
	Port   int    `json:"port,omitempty"`
}

func Act(ctx context.Context, projectsDir string, request ActionRequest) error {
	serviceList, err := List(ctx, projectsDir)
	if err != nil {
		return err
	}
	service, ok := findService(serviceList, request.ID)
	if !ok {
		return ErrServiceNotFound
	}

	switch request.Action {
	case "expose":
		port, ok := findHTTPPort(service, request.Port)
		if !ok {
			return fmt.Errorf("%w: HTTP port not found", ErrInvalidAction)
		}
		if port.ExposedURL != "" {
			return nil
		}
		return setTailscaleServe(ctx, request.Port, true)
	case "unexpose":
		port, ok := findHTTPPort(service, request.Port)
		if !ok || port.ExposedPort == 0 {
			return fmt.Errorf("%w: exposed HTTP port not found", ErrInvalidAction)
		}
		return setTailscaleServe(ctx, port.ExposedPort, false)
	case "end":
		for _, port := range service.Ports {
			if port.ExposedURL != "" {
				if err := setTailscaleServe(ctx, port.ExposedPort, false); err != nil {
					return err
				}
			}
		}
		return endService(ctx, service)
	default:
		return ErrInvalidAction
	}
}

func findService(services []Service, id string) (Service, bool) {
	for _, service := range services {
		if service.ID == id {
			return service, true
		}
	}
	return Service{}, false
}

func findHTTPPort(service Service, port int) (Port, bool) {
	for _, candidate := range service.Ports {
		if candidate.Host == port && candidate.Scheme == "http" {
			return candidate, true
		}
	}
	return Port{}, false
}

func endService(ctx context.Context, service Service) error {
	switch service.Runtime {
	case runtimeDocker:
		dockerPath, err := exec.LookPath("docker")
		if err != nil {
			return fmt.Errorf("docker is not installed")
		}
		containerID := strings.TrimPrefix(service.ID, "docker:")
		if containerID == "" || containerID == service.ID {
			return ErrInvalidAction
		}
		ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
		defer cancel()
		output, err := exec.CommandContext(ctx, dockerPath, "stop", containerID).CombinedOutput()
		if err != nil {
			return commandError("docker stop", output, err)
		}
		return nil
	case runtimeProcess:
		pidText := strings.TrimPrefix(service.ID, "process:")
		pid, err := strconv.Atoi(pidText)
		if err != nil || pid <= 0 {
			return ErrInvalidAction
		}
		process, err := os.FindProcess(pid)
		if err != nil {
			return fmt.Errorf("find process: %w", err)
		}
		if err := process.Signal(syscall.SIGTERM); err != nil {
			return fmt.Errorf("end process: %w", err)
		}
		return nil
	default:
		return ErrInvalidAction
	}
}

func commandError(name string, output []byte, err error) error {
	message := strings.TrimSpace(string(output))
	if message == "" {
		message = err.Error()
	}
	return fmt.Errorf("%s: %s", name, message)
}
