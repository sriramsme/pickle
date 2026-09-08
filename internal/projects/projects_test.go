package projects

import (
	"os"
	"path/filepath"
	"testing"
)

func TestList(t *testing.T) {
	root := t.TempDir()
	for _, name := range []string{"zebra", "alpha.app"} {
		if err := os.Mkdir(filepath.Join(root, name), 0o755); err != nil {
			t.Fatalf("create project: %v", err)
		}
	}
	if err := os.WriteFile(filepath.Join(root, "notes.txt"), []byte("ignored"), 0o644); err != nil {
		t.Fatalf("create file: %v", err)
	}

	projects, err := List(root)
	if err != nil {
		t.Fatalf("list projects: %v", err)
	}
	if len(projects) != 2 {
		t.Fatalf("unexpected project count: %d", len(projects))
	}
	if projects[0].Name != "alpha.app" || projects[0].Session != "alpha_app" {
		t.Fatalf("unexpected first project: %+v", projects[0])
	}
	if projects[1].Name != "zebra" || projects[1].Session != "zebra" {
		t.Fatalf("unexpected second project: %+v", projects[1])
	}
}

func TestListMissingDirectory(t *testing.T) {
	projects, err := List(filepath.Join(t.TempDir(), "missing"))
	if err != nil {
		t.Fatalf("list missing directory: %v", err)
	}
	if len(projects) != 0 {
		t.Fatalf("unexpected projects: %+v", projects)
	}
}

func TestFindRejectsNestedPath(t *testing.T) {
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "project"), 0o755); err != nil {
		t.Fatalf("create project: %v", err)
	}

	if _, _, err := Find(root, "project/../other"); err != ErrNotFound {
		t.Fatalf("unexpected error: %v", err)
	}
}
