package projects

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

var ErrNotFound = errors.New("project not found")

type Project struct {
	Name    string `json:"name"`
	Session string `json:"session"`
}

func List(root string) ([]Project, error) {
	entries, err := os.ReadDir(root)
	if errors.Is(err, os.ErrNotExist) {
		return []Project{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read projects directory: %w", err)
	}

	projects := make([]Project, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		projects = append(projects, Project{
			Name:    entry.Name(),
			Session: SessionName(entry.Name()),
		})
	}

	sort.Slice(projects, func(i, j int) bool {
		left := strings.ToLower(projects[i].Name)
		right := strings.ToLower(projects[j].Name)
		if left == right {
			return projects[i].Name < projects[j].Name
		}
		return left < right
	})
	return projects, nil
}

func Find(root, name string) (Project, string, error) {
	projects, err := List(root)
	if err != nil {
		return Project{}, "", err
	}
	for _, project := range projects {
		if project.Name == name {
			return project, filepath.Join(root, project.Name), nil
		}
	}
	return Project{}, "", ErrNotFound
}

func SessionName(projectName string) string {
	return strings.ReplaceAll(projectName, ".", "_")
}
