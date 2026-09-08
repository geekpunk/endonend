// Command server runs the endonend platform's crawler and versioned
// GraphQL catalog API in one process, per KB/0010-mvp-scope.md's MVP
// scope: a single combined backend service, matching
// KB/0006-container-infrastructure.md's compose design.
package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"endonend/api/internal/crawler"
	"endonend/api/internal/db"
	"endonend/api/internal/graphqlapi"
	"endonend/api/internal/store"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	if err := run(ctx); err != nil {
		log.Fatal(err)
	}
}

func run(ctx context.Context) error {
	dsn, err := requireEnv("DATABASE_URL")
	if err != nil {
		return err
	}
	adminToken, err := requireEnv("ADMIN_TOKEN")
	if err != nil {
		return err
	}
	port := envOr("PORT", "8080")
	checkInterval := envOr("CRAWL_CHECK_INTERVAL", "60s")
	interval, err := time.ParseDuration(checkInterval)
	if err != nil {
		return err
	}

	pool, err := db.Open(ctx, dsn)
	if err != nil {
		return err
	}
	defer pool.Close()

	s := store.New(pool)
	c := crawler.New(s)

	go c.RunLoop(ctx, interval, func(url string, err error) {
		log.Printf("crawl error for %s: %v", url, err)
	})

	mux := http.NewServeMux()
	mux.Handle("/v1/graphql", graphqlapi.NewPublicHandler(s))
	mux.Handle("/v1/admin/graphql", graphqlapi.NewAdminHandler(s, c, adminToken))
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) })

	srv := &http.Server{Addr: ":" + port, Handler: mux}
	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		// Best-effort: a shutdown error here just means some in-flight
		// requests were cut off by the timeout, nothing left to act on.
		_ = srv.Shutdown(shutdownCtx)
	}()

	log.Printf("listening on :%s (public /v1/graphql, admin /v1/admin/graphql)", port)
	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}
