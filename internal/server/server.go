package server

import (
	"io/fs"
	"log"
	"net/http"

	"github.com/sriramsme/pickle/internal/terminal"
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

	dist, err := fs.Sub(webassets.Dist, "dist")
	if err != nil {
		panic(err)
	}
	mux.Handle("/", http.FileServer(http.FS(dist)))

	return mux
}
