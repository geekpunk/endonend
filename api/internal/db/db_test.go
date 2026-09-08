package db

import (
	"context"
	"os"
	"testing"
)

// testDSN returns the Postgres connection string integration tests use,
// skipping the test when TEST_DATABASE_URL isn't set (for example, in CI,
// which doesn't run a Postgres service for this module). Point it at a
// throwaway database: tests apply the schema and write rows to it.
func testDSN(t *testing.T) string {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set, skipping Postgres integration test")
	}
	return dsn
}

func TestOpen_AppliesSchemaIdempotently(t *testing.T) {
	dsn := testDSN(t)
	ctx := context.Background()

	pool, err := Open(ctx, dsn)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer pool.Close()

	// A second Open against the same database must not fail even though
	// every table already exists.
	pool2, err := Open(ctx, dsn)
	if err != nil {
		t.Fatalf("second Open: %v", err)
	}
	defer pool2.Close()

	var tableCount int
	err = pool.QueryRow(ctx, `
		SELECT count(*) FROM information_schema.tables
		WHERE table_schema = 'public' AND table_name = 'identities'
	`).Scan(&tableCount)
	if err != nil {
		t.Fatalf("query information_schema: %v", err)
	}
	if tableCount != 1 {
		t.Errorf("identities table count = %d, want 1", tableCount)
	}
}

func TestOpen_RecordsMigrationExactlyOnce(t *testing.T) {
	dsn := testDSN(t)
	ctx := context.Background()

	pool, err := Open(ctx, dsn)
	if err != nil {
		t.Fatalf("first Open: %v", err)
	}
	defer pool.Close()

	pool2, err := Open(ctx, dsn)
	if err != nil {
		t.Fatalf("second Open: %v", err)
	}
	defer pool2.Close()

	var count int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM schema_migrations WHERE version = 1`).Scan(&count); err != nil {
		t.Fatalf("query schema_migrations: %v", err)
	}
	if count != 1 {
		t.Errorf("schema_migrations rows for version 1 = %d, want exactly 1 (migration must not reapply)", count)
	}
}

func TestOpen_RejectsBadDSN(t *testing.T) {
	if _, err := Open(context.Background(), "postgres://nope:nope@127.0.0.1:1/nope"); err == nil {
		t.Error("Open with an unreachable DSN: want error, got nil")
	}
}
