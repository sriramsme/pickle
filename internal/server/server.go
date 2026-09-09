package server

import (
	"encoding/json"
	"errors"
	"io/fs"
	"log"
	"net/http"

	"github.com/sriramsme/pickle/internal/projects"
	"github.com/sriramsme/pickle/internal/services"
	"github.com/sriramsme/pickle/internal/terminal"
	tmuxctl "github.com/sriramsme/pickle/internal/tmux"
	webassets "github.com/sriramsme/pickle/web"
)

type openProjectRequest struct {
	Name string `json:"name"`
}

func New(projectsDir string) http.Handler {
	terminalHandler := terminal.New("pickle")

	mux := http.NewServeMux()
	mux.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		if err := terminalHandler.Serve(w, r); err != nil {
			log.Printf("terminal connection: %v", err)
		}
	})
	mux.HandleFunc("/api/tmux/sessions", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		sessions, err := tmuxctl.ListSessions(r.Context())
		if err != nil {
			log.Printf("list tmux sessions: %v", err)
			http.Error(w, "failed to list tmux sessions", http.StatusInternalServerError)
			return
		}
		writeJSON(w, sessions)
	})
	mux.HandleFunc("/api/projects", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			projectList, err := projects.List(projectsDir)
			if err != nil {
				log.Printf("list projects: %v", err)
				http.Error(w, "failed to list projects", http.StatusInternalServerError)
				return
			}
			writeJSON(w, projectList)
		case http.MethodPost:
			var request openProjectRequest
			r.Body = http.MaxBytesReader(w, r.Body, 4<<10)
			if err := json.NewDecoder(r.Body).Decode(&request); err != nil || request.Name == "" {
				http.Error(w, "invalid project", http.StatusBadRequest)
				return
			}

			project, directory, err := projects.Find(projectsDir, request.Name)
			if errors.Is(err, projects.ErrNotFound) {
				http.Error(w, "project not found", http.StatusNotFound)
				return
			}
			if err != nil {
				log.Printf("find project: %v", err)
				http.Error(w, "failed to find project", http.StatusInternalServerError)
				return
			}
			if err := tmuxctl.EnsureSession(r.Context(), project.Session, directory); err != nil {
				log.Printf("open project: %v", err)
				http.Error(w, "failed to open project", http.StatusInternalServerError)
				return
			}
			writeJSON(w, project)
		default:
			w.Header().Set("Allow", "GET, POST")
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})
	mux.Handle("/api/services", serviceHandler(projectsDir))

	dist, err := fs.Sub(webassets.Dist, "dist")
	if err != nil {
		panic(err)
	}
	index, err := fs.ReadFile(dist, "index.html")
	if err != nil {
		panic(err)
	}
	serveIndex := func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write(index)
	}
	mux.HandleFunc("/tmux/", serveIndex)
	mux.HandleFunc("/projects", serveIndex)
	mux.HandleFunc("/services", serveIndex)
	mux.HandleFunc("/sessions", serveIndex)
	mux.Handle("/", http.FileServer(http.FS(dist)))

	return mux
}

func serviceHandler(projectsDir string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			serviceList, err := services.List(r.Context(), projectsDir)
			if err != nil {
				log.Printf("list services: %v", err)
				http.Error(w, "failed to list services", http.StatusInternalServerError)
				return
			}
			writeJSON(w, serviceList)
		case http.MethodPost:
			var request services.ActionRequest
			r.Body = http.MaxBytesReader(w, r.Body, 4<<10)
			if err := json.NewDecoder(r.Body).Decode(&request); err != nil || request.ID == "" {
				http.Error(w, "invalid service action", http.StatusBadRequest)
				return
			}
			if err := services.Act(r.Context(), projectsDir, request); err != nil {
				if errors.Is(err, services.ErrServiceNotFound) {
					http.Error(w, err.Error(), http.StatusNotFound)
					return
				}
				if errors.Is(err, services.ErrInvalidAction) {
					http.Error(w, err.Error(), http.StatusBadRequest)
					return
				}
				log.Printf("service action: %v", err)
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			writeJSON(w, map[string]bool{"ok": true})
		default:
			w.Header().Set("Allow", "GET, POST")
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})
}

func writeJSON(w http.ResponseWriter, value any) {
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(value); err != nil {
		log.Printf("encode JSON response: %v", err)
	}
}
