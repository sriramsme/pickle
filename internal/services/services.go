package services

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/sriramsme/pickle/internal/projects"
)

type Service struct {
	Project string `json:"project"`
	Process string `json:"process"`
	Port    int    `json:"port"`
}

type listeningSocket struct {
	Inode string
	Port  int
}

func List(projectsDir string) ([]Service, error) {
	return list("/proc", projectsDir, os.Getpid())
}

func list(procRoot, projectsDir string, excludedPID int) ([]Service, error) {
	projectList, err := projects.List(projectsDir)
	if err != nil {
		return nil, err
	}
	if len(projectList) == 0 {
		return []Service{}, nil
	}

	projectsRoot, err := filepath.Abs(projectsDir)
	if err != nil {
		return nil, fmt.Errorf("resolve projects directory: %w", err)
	}
	if resolvedRoot, err := filepath.EvalSymlinks(projectsRoot); err == nil {
		projectsRoot = resolvedRoot
	}

	projectNames := make(map[string]struct{}, len(projectList))
	for _, project := range projectList {
		projectNames[project.Name] = struct{}{}
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

	seen := make(map[string]struct{})
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

		processName, err := os.ReadFile(filepath.Join(processDir, "comm"))
		if err != nil {
			continue
		}
		process := strings.TrimSpace(string(processName))

		fds, err := os.ReadDir(filepath.Join(processDir, "fd"))
		if err != nil {
			continue
		}
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
				key := fmt.Sprintf("%s\x00%s\x00%d", project, process, socket.Port)
				if _, ok := seen[key]; ok {
					continue
				}
				seen[key] = struct{}{}
				services = append(services, Service{
					Project: project,
					Process: process,
					Port:    socket.Port,
				})
			}
		}
	}

	sort.Slice(services, func(i, j int) bool {
		if services[i].Project != services[j].Project {
			return services[i].Project < services[j].Project
		}
		if services[i].Port != services[j].Port {
			return services[i].Port < services[j].Port
		}
		return services[i].Process < services[j].Process
	})
	return services, nil
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
