package server

import (
	"encoding/json"
	"io/fs"
	"log"
	"net/http"

	"github.com/sriramsme/pickle/internal/terminal"
	tmuxctl "github.com/sriramsme/pickle/internal/tmux"
	webassets "github.com/sriramsme/pickle/web"
)

func New() http.Handler {
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
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(sessions); err != nil {
			log.Printf("encode tmux sessions: %v", err)
		}
	})

	dist, err := fs.Sub(webassets.Dist, "dist")
	if err != nil {
		panic(err)
	}
	index, err := fs.ReadFile(dist, "index.html")
	if err != nil {
		panic(err)
	}
	mux.HandleFunc("/tmux/", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write(index)
	})
	mux.Handle("/", http.FileServer(http.FS(dist)))

	return mux
}
