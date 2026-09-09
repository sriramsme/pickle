package config

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestStoreStartsUnconfigured(t *testing.T) {
	projectsDirectory := filepath.Join(t.TempDir(), "projects")
	store, err := Load(filepath.Join(t.TempDir(), "config.json"), projectsDirectory, "")
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	settings := store.Current()
	if settings.Configured || settings.ProjectsDirectory != projectsDirectory {
		t.Fatalf("unexpected settings: %+v", settings)
	}
}

func TestStoreSavesAndLoadsProjectsDirectory(t *testing.T) {
	root := t.TempDir()
	projectsDirectory := filepath.Join(root, "code")
	if err := os.Mkdir(projectsDirectory, 0o755); err != nil {
		t.Fatalf("create projects directory: %v", err)
	}
	path := filepath.Join(root, "config", "config.json")

	store, err := Load(path, filepath.Join(root, "projects"), "")
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	settings, err := store.Save(projectsDirectory)
	if err != nil {
		t.Fatalf("save config: %v", err)
	}
	if !settings.Configured || settings.ProjectsDirectory != projectsDirectory {
		t.Fatalf("unexpected saved settings: %+v", settings)
	}

	reloaded, err := Load(path, filepath.Join(root, "projects"), "")
	if err != nil {
		t.Fatalf("reload config: %v", err)
	}
	if reloaded.Current() != settings {
		t.Fatalf("unexpected reloaded settings: %+v", reloaded.Current())
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("inspect config: %v", err)
	}
	if permissions := info.Mode().Perm(); permissions != 0o600 {
		t.Fatalf("unexpected config permissions: %o", permissions)
	}
}

func TestStoreRejectsMissingProjectsDirectory(t *testing.T) {
	store, err := Load(filepath.Join(t.TempDir(), "config.json"), t.TempDir(), "")
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	if _, err := store.Save(filepath.Join(t.TempDir(), "missing")); !errors.Is(err, ErrInvalidDirectory) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestProjectsDirectoryOverrideLocksSetting(t *testing.T) {
	projectsDirectory := t.TempDir()
	store, err := Load(filepath.Join(t.TempDir(), "config.json"), t.TempDir(), projectsDirectory)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	settings := store.Current()
	if !settings.Configured || !settings.ProjectsDirectoryLocked {
		t.Fatalf("unexpected settings: %+v", settings)
	}
	if _, err := store.Save(t.TempDir()); !errors.Is(err, ErrLocked) {
		t.Fatalf("unexpected error: %v", err)
	}
}
