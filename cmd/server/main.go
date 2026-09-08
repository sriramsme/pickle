package main

import (
	"flag"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/sriramsme/pickle/internal/server"
)

func main() {
	home, err := os.UserHomeDir()
	if err != nil {
		log.Fatal(err)
	}

	addr := flag.String("addr", "127.0.0.1:8080", "HTTP listen address")
	projectsDir := flag.String("projects-dir", filepath.Join(home, "projects"), "projects directory")
	flag.Parse()

	httpServer := &http.Server{
		Addr:              *addr,
		Handler:           server.New(*projectsDir),
		ReadHeaderTimeout: 5 * time.Second,
	}

	log.Printf("pickle listening on http://%s", *addr)
	if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}
}
