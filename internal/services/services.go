package services

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/sriramsme/pickle/internal/projects"
)

const (
	runtimeProcess = "process"
	runtimeDocker  = "docker"
)

type Service struct {
	ID        string `json:"id"`
	Project   string `json:"project"`
	Name      string `json:"name"`
	Runtime   string `json:"runtime"`
	State     string `json:"state,omitempty"`
	Health    string `json:"health,omitempty"`
	Image     string `json:"image,omitempty"`
	Container string `json:"container,omitempty"`
	Ports     []Port `json:"ports"`
}

type Port struct {
	Host        int    `json:"host,omitempty"`
	Container   int    `json:"container,omitempty"`
	Protocol    string `json:"protocol"`
	Scheme      string `json:"scheme,omitempty"`
	ExposedURL  string `json:"exposedUrl,omitempty"`
	ExposedPort int    `json:"exposedPort,omitempty"`
}

type listeningSocket struct {
	Inode string
	Port  int
}

func List(ctx context.Context, projectsDir string) ([]Service, error) {
	processServices, err := listProcesses("/proc", projectsDir, os.Getpid())
	if err != nil {
		return nil, err
	}

	dockerServices, _ := listDocker(ctx, projectsDir)
	serviceList := append(processServices, dockerServices...)
	probeHTTP(serviceList)
	markTailscaleExposures(ctx, serviceList)
	sortServices(serviceList)
	return serviceList, nil
}

func listProcesses(procRoot, projectsDir string, excludedPID int) ([]Service, error) {
	projectList, projectsRoot, projectNames, err := projectIndex(projectsDir)
	if err != nil {
		return nil, err
	}
	if len(projectList) == 0 {
		return []Service{}, nil
	}

	sockets, err := readListeningSockets(procRoot)
	if err != nil {
		return nil, err
	}
	socketsByInode := make(map[string][]listeningSocket, len(sockets))
	for _, socket := range sockets {
		socketsByInode[socket.Inode] = append(socketsByInode[socket.Inode], socket)
	}

	entries, err := os.ReadDir(procRoot)
	if err != nil {
		return nil, fmt.Errorf("read proc: %w", err)
	}

	services := make([]Service, 0)
	for _, entry := range entries {
		pid, err := strconv.Atoi(entry.Name())
		if err != nil || !entry.IsDir() || pid == excludedPID {
			continue
		}

		processDir := filepath.Join(procRoot, entry.Name())
		cwd, err := os.Readlink(filepath.Join(processDir, "cwd"))
		if err != nil {
			continue
		}
		project, ok := projectForPath(projectsRoot, cwd, projectNames)
		if !ok {
			continue
		}

		processName, err := readProcessName(processDir)
		if err != nil {
			continue
		}
		ports := processPorts(processDir, socketsByInode)
		if len(ports) == 0 {
			continue
		}

		services = append(services, Service{
			ID:      fmt.Sprintf("process:%d", pid),
			Project: project,
			Name:    processName,
			Runtime: runtimeProcess,
			State:   "running",
			Ports:   ports,
		})
	}

	return services, nil
}

func readProcessName(processDir string) (string, error) {
	executable, err := os.Readlink(filepath.Join(processDir, "exe"))
	if err == nil {
		return filepath.Base(strings.TrimSuffix(executable, " (deleted)")), nil
	}
	name, err := os.ReadFile(filepath.Join(processDir, "comm"))
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(name)), nil
}

func processPorts(processDir string, socketsByInode map[string][]listeningSocket) []Port {
	fds, err := os.ReadDir(filepath.Join(processDir, "fd"))
	if err != nil {
		return nil
	}

	seen := make(map[int]struct{})
	ports := make([]Port, 0)
	for _, fd := range fds {
		target, err := os.Readlink(filepath.Join(processDir, "fd", fd.Name()))
		if err != nil {
			continue
		}
		inode, ok := socketInode(target)
		if !ok {
			continue
		}
		for _, socket := range socketsByInode[inode] {
			if _, ok := seen[socket.Port]; ok {
				continue
			}
			seen[socket.Port] = struct{}{}
			ports = append(ports, Port{Host: socket.Port, Protocol: "tcp"})
		}
	}
	sort.Slice(ports, func(i, j int) bool { return ports[i].Host < ports[j].Host })
	return ports
}

func projectIndex(projectsDir string) ([]projects.Project, string, map[string]struct{}, error) {
	projectList, err := projects.List(projectsDir)
	if err != nil {
		return nil, "", nil, err
	}
	projectsRoot, err := filepath.Abs(projectsDir)
	if err != nil {
		return nil, "", nil, fmt.Errorf("resolve projects directory: %w", err)
	}
	if resolvedRoot, err := filepath.EvalSymlinks(projectsRoot); err == nil {
		projectsRoot = resolvedRoot
	}
	projectNames := make(map[string]struct{}, len(projectList))
	for _, project := range projectList {
		projectNames[project.Name] = struct{}{}
	}
	return projectList, projectsRoot, projectNames, nil
}

func sortServices(services []Service) {
	sort.Slice(services, func(i, j int) bool {
		if services[i].Project != services[j].Project {
			return strings.ToLower(services[i].Project) < strings.ToLower(services[j].Project)
		}
		if services[i].Runtime != services[j].Runtime {
			return services[i].Runtime < services[j].Runtime
		}
		return strings.ToLower(services[i].Name) < strings.ToLower(services[j].Name)
	})
}

func readListeningSockets(procRoot string) ([]listeningSocket, error) {
	var sockets []listeningSocket
	for _, name := range []string{"tcp", "tcp6"} {
		data, err := os.ReadFile(filepath.Join(procRoot, "net", name))
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return nil, fmt.Errorf("read proc net %s: %w", name, err)
		}
		parsed, err := parseListeningSockets(data)
		if err != nil {
			return nil, fmt.Errorf("parse proc net %s: %w", name, err)
		}
		sockets = append(sockets, parsed...)
	}
	return sockets, nil
}

func parseListeningSockets(data []byte) ([]listeningSocket, error) {
	var sockets []listeningSocket
	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) == 0 || fields[0] == "sl" {
			continue
		}
		if len(fields) < 10 {
			return nil, fmt.Errorf("invalid socket row %q", scanner.Text())
		}
		if fields[3] != "0A" {
			continue
		}

		_, portHex, ok := strings.Cut(fields[1], ":")
		if !ok {
			return nil, fmt.Errorf("invalid local address %q", fields[1])
		}
		port, err := strconv.ParseUint(portHex, 16, 16)
		if err != nil {
			return nil, fmt.Errorf("parse port %q: %w", portHex, err)
		}
		sockets = append(sockets, listeningSocket{Inode: fields[9], Port: int(port)})
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scan sockets: %w", err)
	}
	return sockets, nil
}

func projectForPath(root, cwd string, projectNames map[string]struct{}) (string, bool) {
	relative, err := filepath.Rel(root, cwd)
	if err != nil || relative == "." || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return "", false
	}
	project := strings.Split(relative, string(filepath.Separator))[0]
	_, ok := projectNames[project]
	return project, ok
}

func socketInode(target string) (string, bool) {
	const prefix = "socket:["
	if !strings.HasPrefix(target, prefix) || !strings.HasSuffix(target, "]") {
		return "", false
	}
	return target[len(prefix) : len(target)-1], true
}
