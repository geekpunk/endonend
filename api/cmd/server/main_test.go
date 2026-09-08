package main

import (
	"context"
	"net/http"
	"os"
	"testing"
	"time"
)

func TestRun_MissingDatabaseURLReturnsError(t *testing.T) {
	t.Setenv("DATABASE_URL", "")
	t.Setenv("ADMIN_TOKEN", "secret")
	if err := run(context.Background()); err == nil {
		t.Error("run with no DATABASE_URL: want error, got nil")
	}
}

func TestRun_MissingAdminTokenReturnsError(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://ignored")
	t.Setenv("ADMIN_TOKEN", "")
	if err := run(context.Background()); err == nil {
		t.Error("run with no ADMIN_TOKEN: want error, got nil")
	}
}

func TestRun_ServesHealthzUntilCancelled(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set, skipping Postgres integration test")
	}
	t.Setenv("DATABASE_URL", dsn)
	t.Setenv("ADMIN_TOKEN", "secret")
	t.Setenv("PORT", "18080")
	t.Setenv("CRAWL_CHECK_INTERVAL", "1h")

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- run(ctx) }()

	var resp *http.Response
	var err error
	for range 50 {
		resp, err = http.Get("http://localhost:18080/healthz")
		if err == nil {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if err != nil {
		t.Fatalf("GET /healthz never succeeded: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("/healthz status = %d, want 200", resp.StatusCode)
	}
	_ = resp.Body.Close()

	cancel()
	select {
	case err := <-done:
		if err != nil {
			t.Errorf("run returned %v after cancellation, want nil", err)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("run did not return within 3s of context cancellation")
	}
}
