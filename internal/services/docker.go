package services

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"sort"
	"strconv"
	"strings"
	"time"
)

type dockerPortBinding struct {
	HostPort string `json:"HostPort"`
}

type dockerContainer struct {
	ID     string `json:"Id"`
	Name   string `json:"Name"`
	Config struct {
		Image  string            `json:"Image"`
		Labels map[string]string `json:"Labels"`
	} `json:"Config"`
	State struct {
		Status string `json:"Status"`
		Health *struct {
			Status string `json:"Status"`
		} `json:"Health"`
	} `json:"State"`
	NetworkSettings struct {
		Ports map[string][]dockerPortBinding `json:"Ports"`
	} `json:"NetworkSettings"`
}

func listDocker(ctx context.Context, projectsDir string) ([]Service, error) {
	dockerPath, err := exec.LookPath("docker")
	if err != nil {
		return []Service{}, nil
	}

	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	output, err := exec.CommandContext(ctx, dockerPath, "ps", "--quiet", "--no-trunc").Output()
	if err != nil {
		return []Service{}, err
	}
	ids := strings.Fields(string(output))
	if len(ids) == 0 {
		return []Service{}, nil
	}

	args := append([]string{"inspect"}, ids...)
	output, err = exec.CommandContext(ctx, dockerPath, args...).Output()
	if err != nil {
		return []Service{}, err
	}
	return parseDockerContainers(output, projectsDir)
}

func parseDockerContainers(data []byte, projectsDir string) ([]Service, error) {
	_, projectsRoot, projectNames, err := projectIndex(projectsDir)
	if err != nil {
		return nil, err
	}

	var containers []dockerContainer
	if err := json.Unmarshal(data, &containers); err != nil {
		return nil, fmt.Errorf("parse docker containers: %w", err)
	}

	services := make([]Service, 0, len(containers))
	for _, container := range containers {
		workingDir := container.Config.Labels["com.docker.compose.project.working_dir"]
		project, ok := projectForPath(projectsRoot, workingDir, projectNames)
		if !ok {
			continue
		}

		name := container.Config.Labels["com.docker.compose.service"]
		if name == "" {
			name = strings.TrimPrefix(container.Name, "/")
		}
		health := ""
		if container.State.Health != nil {
			health = container.State.Health.Status
		}

		services = append(services, Service{
			ID:        "docker:" + container.ID,
			Project:   project,
			Name:      name,
			Runtime:   runtimeDocker,
			State:     container.State.Status,
			Health:    health,
			Image:     container.Config.Image,
			Container: strings.TrimPrefix(container.Name, "/"),
			Ports:     dockerPorts(container.NetworkSettings.Ports),
		})
	}
	return services, nil
}

func dockerPorts(bindings map[string][]dockerPortBinding) []Port {
	ports := make([]Port, 0, len(bindings))
	seen := make(map[string]struct{})
	for containerPort, hosts := range bindings {
		targetText, protocol, ok := strings.Cut(containerPort, "/")
		if !ok {
			continue
		}
		target, err := strconv.Atoi(targetText)
		if err != nil {
			continue
		}
		if len(hosts) == 0 {
			key := fmt.Sprintf("0:%d:%s", target, protocol)
			seen[key] = struct{}{}
			ports = append(ports, Port{Container: target, Protocol: protocol})
			continue
		}
		for _, binding := range hosts {
			host, err := strconv.Atoi(binding.HostPort)
			if err != nil {
				continue
			}
			key := fmt.Sprintf("%d:%d:%s", host, target, protocol)
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
			ports = append(ports, Port{Host: host, Container: target, Protocol: protocol})
		}
	}
	sort.Slice(ports, func(i, j int) bool {
		if ports[i].Host != ports[j].Host {
			return ports[i].Host < ports[j].Host
		}
		return ports[i].Container < ports[j].Container
	})
	return ports
}
