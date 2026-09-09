package config

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

var (
	ErrInvalidDirectory = errors.New("invalid projects directory")
	ErrLocked           = errors.New("projects directory is set by a command-line flag")
)

type Settings struct {
	Configured              bool   `json:"configured"`
	ProjectsDirectory       string `json:"projectsDirectory"`
	ProjectsDirectoryLocked bool   `json:"projectsDirectoryLocked"`
}

type fileConfig struct {
	ProjectsDirectory string `json:"projectsDirectory"`
}

type Store struct {
	mu       sync.RWMutex
	path     string
	settings Settings
}

func DefaultPath(home string) string {
	if directory := os.Getenv("XDG_CONFIG_HOME"); filepath.IsAbs(directory) {
		return filepath.Join(directory, "pickle", "config.json")
	}
	return filepath.Join(home, ".config", "pickle", "config.json")
}

func Load(path, defaultProjectsDirectory, projectsDirectoryOverride string) (*Store, error) {
	defaultDirectory, err := cleanDirectory(defaultProjectsDirectory)
	if err != nil {
		return nil, err
	}
	store := &Store{
		path: path,
		settings: Settings{
			ProjectsDirectory: defaultDirectory,
		},
	}

	if projectsDirectoryOverride != "" {
		directory, err := validateDirectory(projectsDirectoryOverride)
		if err != nil {
			return nil, err
		}
		store.settings = Settings{
			Configured:              true,
			ProjectsDirectory:       directory,
			ProjectsDirectoryLocked: true,
		}
		return store, nil
	}

	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return store, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}

	var saved fileConfig
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&saved); err != nil {
		return nil, fmt.Errorf("decode config: %w", err)
	}
	if err := ensureJSONEnd(decoder); err != nil {
		return nil, fmt.Errorf("decode config: %w", err)
	}
	directory, err := cleanDirectory(saved.ProjectsDirectory)
	if err != nil {
		return nil, err
	}
	store.settings.Configured = true
	store.settings.ProjectsDirectory = directory
	return store, nil
}

func (s *Store) Current() Settings {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.settings
}

func (s *Store) ProjectsDirectory() string {
	return s.Current().ProjectsDirectory
}

func (s *Store) Save(projectsDirectory string) (Settings, error) {
	directory, err := validateDirectory(projectsDirectory)
	if err != nil {
		return Settings{}, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if s.settings.ProjectsDirectoryLocked {
		return Settings{}, ErrLocked
	}

	data, err := json.MarshalIndent(fileConfig{ProjectsDirectory: directory}, "", "  ")
	if err != nil {
		return Settings{}, fmt.Errorf("encode config: %w", err)
	}
	if err := writeFile(s.path, append(data, '\n')); err != nil {
		return Settings{}, err
	}

	s.settings.Configured = true
	s.settings.ProjectsDirectory = directory
	return s.settings, nil
}

func cleanDirectory(value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", fmt.Errorf("%w: choose a directory", ErrInvalidDirectory)
	}
	if value == "~" || strings.HasPrefix(value, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("resolve home directory: %w", err)
		}
		if value == "~" {
			value = home
		} else {
			value = filepath.Join(home, strings.TrimPrefix(value, "~/"))
		}
	}
	directory, err := filepath.Abs(value)
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrInvalidDirectory, err)
	}
	return filepath.Clean(directory), nil
}

func validateDirectory(value string) (string, error) {
	directory, err := cleanDirectory(value)
	if err != nil {
		return "", err
	}
	info, err := os.Stat(directory)
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrInvalidDirectory, err)
	}
	if !info.IsDir() {
		return "", fmt.Errorf("%w: path is not a directory", ErrInvalidDirectory)
	}
	return directory, nil
}

func ensureJSONEnd(decoder *json.Decoder) error {
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		if err == nil {
			return errors.New("unexpected data after config")
		}
		return err
	}
	return nil
}

func writeFile(path string, data []byte) error {
	directory := filepath.Dir(path)
	if err := os.MkdirAll(directory, 0o700); err != nil {
		return fmt.Errorf("create config directory: %w", err)
	}
	temporary, err := os.CreateTemp(directory, ".config-*.tmp")
	if err != nil {
		return fmt.Errorf("create config: %w", err)
	}
	temporaryName := temporary.Name()
	defer os.Remove(temporaryName)

	if err := temporary.Chmod(0o600); err != nil {
		_ = temporary.Close()
		return fmt.Errorf("set config permissions: %w", err)
	}
	if _, err := temporary.Write(data); err != nil {
		_ = temporary.Close()
		return fmt.Errorf("write config: %w", err)
	}
	if err := temporary.Close(); err != nil {
		return fmt.Errorf("close config: %w", err)
	}
	if err := os.Rename(temporaryName, path); err != nil {
		return fmt.Errorf("save config: %w", err)
	}
	return nil
}
