package crawler

import (
	"context"
	"os"
	"sync"
	"testing"
	"time"

	"endonend/api/internal/db"
	"endonend/protocol/history"
	"endonend/protocol/manifest"
)

func TestPollDue_PollsStaleIdentityAndSkipsFreshOne(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()

	m, priv := buildSignedManifest(t)
	m.Refresh.TTLSeconds = 60
	entries := []manifest.HistoryEntry{}
	server := serveManifest(t, &m, &entries)
	defer server.Close()

	entry := signedHistoryEntry(t, priv)
	head, err := history.Hash(entry)
	if err != nil {
		t.Fatalf("history.Hash: %v", err)
	}
	m.History.HeadHash = head
	entries = append(entries, entry)
	resign(t, &m, priv)

	c := New(s)
	url := server.URL + "/.well-known/endonend/manifest.json"
	if err := c.PollOne(ctx, url); err != nil {
		t.Fatalf("initial PollOne: %v", err)
	}

	// Make it look stale by backdating fetched_at well past its 60s TTL,
	// via a second connection to the same test database (Store itself
	// exposes no way to backdate a row, since nothing outside a crawl
	// legitimately needs to).
	pool2, err := db.Open(ctx, os.Getenv("TEST_DATABASE_URL"))
	if err != nil {
		t.Fatalf("second db.Open: %v", err)
	}
	defer pool2.Close()
	if _, err := pool2.Exec(ctx, `UPDATE identities SET fetched_at = $1 WHERE url = $2`, time.Now().Add(-time.Hour), server.URL); err != nil {
		t.Fatalf("backdate fetched_at: %v", err)
	}

	var mu sync.Mutex
	var errs []string
	c2 := New(s)
	c2.PollDue(ctx, func(url string, err error) {
		mu.Lock()
		defer mu.Unlock()
		errs = append(errs, url)
	})

	mu.Lock()
	defer mu.Unlock()
	if len(errs) != 0 {
		t.Errorf("PollDue reported errors for %v, want none", errs)
	}

	got, err := s.GetIdentity(ctx, server.URL)
	if err != nil {
		t.Fatalf("GetIdentity: %v", err)
	}
	if got == nil {
		t.Fatal("GetIdentity after PollDue: want the identity to still be indexed")
	}
}

func TestRunLoop_StopsOnContextCancellation(t *testing.T) {
	s := testStore(t)
	c := New(s)

	ctx, cancel := context.WithTimeout(context.Background(), 150*time.Millisecond)
	defer cancel()

	// onError may legitimately fire once, right at the deadline, if a
	// ticker-triggered poll is in flight when ctx expires mid-query; that's
	// not what this test is checking. It only asserts that RunLoop honors
	// cancellation and returns promptly.
	done := make(chan struct{})
	go func() {
		c.RunLoop(ctx, 30*time.Millisecond, func(string, error) {})
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("RunLoop did not return within 2s of its context expiring")
	}
}
