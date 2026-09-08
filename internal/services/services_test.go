package services

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseListeningSockets(t *testing.T) {
	sockets, err := parseListeningSockets([]byte(
		"sl local_address rem_address st tx_queue rx_queue tr tm->when retrnsmt uid timeout inode\n" +
			"0: 0100007F:1435 00000000:0000 0A 00000000:00000000 00:00000000 00000000 1000 0 42\n" +
			"1: 0100007F:0BB8 0100007F:1234 01 00000000:00000000 00:00000000 00000000 1000 0 43\n",
	))
	if err != nil {
		t.Fatalf("parse sockets: %v", err)
	}
	if len(sockets) != 1 || sockets[0].Port != 5173 || sockets[0].Inode != "42" {
		t.Fatalf("unexpected sockets: %+v", sockets)
	}
}

func TestListAssociatesProjectProcess(t *testing.T) {
	root := t.TempDir()
	projectsDir := filepath.Join(root, "projects")
	projectDir := filepath.Join(projectsDir, "alpha.app", "web")
	procRoot := filepath.Join(root, "proc")
	processDir := filepath.Join(procRoot, "123")

	for _, directory := range []string{
		projectDir,
		filepath.Join(procRoot, "net"),
		filepath.Join(processDir, "fd"),
	} {
		if err := os.MkdirAll(directory, 0o755); err != nil {
			t.Fatalf("create directory: %v", err)
		}
	}
	if err := os.WriteFile(filepath.Join(procRoot, "net", "tcp"), []byte(
		"0: 0100007F:1435 00000000:0000 0A 00000000:00000000 00:00000000 00000000 1000 0 42\n",
	), 0o644); err != nil {
		t.Fatalf("write sockets: %v", err)
	}
	if err := os.WriteFile(filepath.Join(processDir, "comm"), []byte("node\n"), 0o644); err != nil {
		t.Fatalf("write process name: %v", err)
	}
	if err := os.Symlink(projectDir, filepath.Join(processDir, "cwd")); err != nil {
		t.Fatalf("link cwd: %v", err)
	}
	if err := os.Symlink("socket:[42]", filepath.Join(processDir, "fd", "8")); err != nil {
		t.Fatalf("link socket: %v", err)
	}
	if err := os.Symlink("socket:[42]", filepath.Join(processDir, "fd", "9")); err != nil {
		t.Fatalf("link duplicate socket: %v", err)
	}

	serviceList, err := list(procRoot, projectsDir, 999)
	if err != nil {
		t.Fatalf("list services: %v", err)
	}
	if len(serviceList) != 1 {
		t.Fatalf("unexpected service count: %d", len(serviceList))
	}
	service := serviceList[0]
	if service.Project != "alpha.app" || service.Process != "node" || service.Port != 5173 {
		t.Fatalf("unexpected service: %+v", service)
	}
}

func TestProjectForPathRejectsSibling(t *testing.T) {
	projects := map[string]struct{}{"app": {}}
	if _, ok := projectForPath("/home/user/projects", "/home/user/projects-old/app", projects); ok {
		t.Fatal("matched sibling directory")
	}
}
