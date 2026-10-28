// Command bookmarks serves a small JSON API for saving links.
package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"bookmarks/handlers"
	"bookmarks/store"
)

func main() {
	addr := envOr("BOOKMARKS_ADDR", ":8080")
	dataFile := os.Getenv("BOOKMARKS_DATA_FILE")

	s := store.New()
	if dataFile != "" {
		var err error
		if s, err = store.Open(dataFile); err != nil {
			log.Fatalf("open store: %v", err)
		}
	}

	srv := &http.Server{
		Addr:              addr,
		Handler:           handlers.New(s).Routes(),
		ReadHeaderTimeout: 5 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go func() {
		log.Printf("listening on %s", addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("serve: %v", err)
		}
	}()

	<-ctx.Done()
	shutdown, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdown); err != nil {
		log.Printf("shutdown: %v", err)
	}
}

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
