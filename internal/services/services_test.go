package services

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
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
	if err := os.Symlink("/usr/bin/node", filepath.Join(processDir, "exe")); err != nil {
		t.Fatalf("link process executable: %v", err)
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

	serviceList, err := listProcesses(procRoot, projectsDir, 999)
	if err != nil {
		t.Fatalf("list services: %v", err)
	}
	if len(serviceList) != 1 {
		t.Fatalf("unexpected service count: %d", len(serviceList))
	}
	service := serviceList[0]
	if service.ID != "process:123" || service.Project != "alpha.app" || service.Name != "node" || service.Runtime != "process" {
		t.Fatalf("unexpected service: %+v", service)
	}
	if len(service.Ports) != 1 || service.Ports[0].Host != 5173 {
		t.Fatalf("unexpected service ports: %+v", service.Ports)
	}
}

func TestParseDockerContainers(t *testing.T) {
	projectsDir := t.TempDir()
	projectDir := filepath.Join(projectsDir, "alpha.app")
	if err := os.Mkdir(projectDir, 0o755); err != nil {
		t.Fatalf("create project: %v", err)
	}

	data := []byte(fmt.Sprintf(`[
		{
			"Id": "abc123",
			"Name": "/alpha-web-1",
			"Config": {
				"Image": "node:22",
				"Labels": {
					"com.docker.compose.project.working_dir": %q,
					"com.docker.compose.service": "web"
				}
			},
			"State": {"Status": "running", "Health": {"Status": "healthy"}},
			"NetworkSettings": {
				"Ports": {"3000/tcp": [{"HostPort": "3000"}]}
			}
		},
		{
			"Id": "def456",
			"Name": "/alpha-worker-1",
			"Config": {
				"Image": "node:22",
				"Labels": {
					"com.docker.compose.project.working_dir": %q,
					"com.docker.compose.service": "worker"
				}
			},
			"State": {"Status": "running", "Health": null},
			"NetworkSettings": {"Ports": {}}
		}
	]`, projectDir, projectDir))

	serviceList, err := parseDockerContainers(data, projectsDir)
	if err != nil {
		t.Fatalf("parse docker containers: %v", err)
	}
	if len(serviceList) != 2 {
		t.Fatalf("unexpected service count: %d", len(serviceList))
	}

	web := serviceList[0]
	if web.ID != "docker:abc123" || web.Project != "alpha.app" || web.Name != "web" || web.Health != "healthy" {
		t.Fatalf("unexpected web service: %+v", web)
	}
	if len(web.Ports) != 1 || web.Ports[0].Host != 3000 || web.Ports[0].Container != 3000 {
		t.Fatalf("unexpected web ports: %+v", web.Ports)
	}

	worker := serviceList[1]
	if worker.Name != "worker" || len(worker.Ports) != 0 {
		t.Fatalf("unexpected worker service: %+v", worker)
	}
}

func TestProbeHTTP(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	address, err := url.Parse(server.URL)
	if err != nil {
		t.Fatalf("parse server URL: %v", err)
	}
	port, err := strconv.Atoi(address.Port())
	if err != nil {
		t.Fatalf("parse server port: %v", err)
	}

	serviceList := []Service{{Ports: []Port{{Host: port, Protocol: "tcp"}}}}
	probeHTTP(serviceList)
	if serviceList[0].Ports[0].Scheme != "http" {
		t.Fatalf("unexpected scheme: %q", serviceList[0].Ports[0].Scheme)
	}
}

func TestParseTailscaleServeStatus(t *testing.T) {
	data := []byte(`{
		"Web": {
			"machine.example.ts.net:443": {
				"Handlers": {"/": {"Proxy": "http://127.0.0.1:8080"}}
			},
			"machine.example.ts.net:4321": {
				"Handlers": {"/": {"Proxy": "http://localhost:4321"}}
			}
		}
	}`)

	exposures, err := parseTailscaleServeStatus(data)
	if err != nil {
		t.Fatalf("parse serve status: %v", err)
	}
	if exposures[8080].URL != "https://machine.example.ts.net/" || exposures[8080].Port != 443 {
		t.Fatalf("unexpected default exposure: %+v", exposures[8080])
	}
	if exposures[4321].URL != "https://machine.example.ts.net:4321/" || exposures[4321].Port != 4321 {
		t.Fatalf("unexpected port exposure: %+v", exposures[4321])
	}
	if !tailscalePortInUse(exposures, 443, 443) {
		t.Fatal("expected occupied HTTPS port")
	}
	if tailscalePortInUse(exposures, 4321, 4321) {
		t.Fatal("matched exposure for the same local port")
	}
}

func TestHasHTTPPort(t *testing.T) {
	service := Service{Ports: []Port{
		{Host: 3000, Protocol: "tcp", Scheme: "http"},
		{Host: 5432, Protocol: "tcp"},
	}}
	if _, ok := findHTTPPort(service, 3000); !ok {
		t.Fatal("expected HTTP port")
	}
	if _, ok := findHTTPPort(service, 5432); ok {
		t.Fatal("unexpected non-HTTP port")
	}
}

func TestProjectForPathRejectsSibling(t *testing.T) {
	projects := map[string]struct{}{"app": {}}
	if _, ok := projectForPath("/home/user/projects", "/home/user/projects-old/app", projects); ok {
		t.Fatal("matched sibling directory")
	}
}
