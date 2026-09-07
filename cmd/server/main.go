package main

import (
	"flag"
	"log"
	"net/http"
	"time"

	"github.com/sriramsme/pickle/internal/server"
)

func main() {
	addr := flag.String("addr", "127.0.0.1:8080", "HTTP listen address")
	flag.Parse()

	httpServer := &http.Server{
		Addr:              *addr,
		Handler:           server.New(),
		ReadHeaderTimeout: 5 * time.Second,
	}

	log.Printf("pickle listening on http://%s", *addr)
	if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}
}
