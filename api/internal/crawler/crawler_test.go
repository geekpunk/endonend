package crawler

import (
	"context"
	"crypto/ed25519"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"endonend/api/internal/db"
	"endonend/api/internal/store"
	"endonend/protocol/history"
	"endonend/protocol/manifest"
	"endonend/protocol/signing"
)

func testStore(t *testing.T) *store.Store {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set, skipping Postgres integration test")
	}
	ctx := context.Background()
	pool, err := db.Open(ctx, dsn)
	if err != nil {
		t.Fatalf("db.Open: %v", err)
	}
	t.Cleanup(pool.Close)
	_, err = pool.Exec(ctx, `TRUNCATE identities, label_rosters, albums, tracks, album_splits, contributions, merch_links, purchase_links, history_entries RESTART IDENTITY CASCADE`)
	if err != nil {
		t.Fatalf("truncate tables: %v", err)
	}
	return store.New(pool)
}

// buildSignedManifest returns a minimal, validly signed manifest.
// identity.url and history.url are filled in by the caller once the
// httptest.Server serving it exists, then re-signed.
func buildSignedManifest(t *testing.T) (manifest.Manifest, ed25519.PrivateKey) {
	t.Helper()
	pub, priv, err := signing.GenerateKeypair()
	if err != nil {
		t.Fatalf("GenerateKeypair: %v", err)
	}
	m := manifest.Manifest{
		ManifestVersion: "1.0",
		Identity:        manifest.Identity{Type: "artist", Name: "Ligatures", PublicKey: signing.EncodePublicKey(pub), ContactEmail: "band@ligatures.example"},
		Refresh:         manifest.Refresh{TTLSeconds: 21600},
		Catalog: []manifest.Album{
			{
				AlbumID: "agency-2024", AlbumVersion: 1, AlbumName: "Agency", ReleaseDate: "2024-05-01",
				Images: manifest.Images{Front: "https://x/front.png", Back: "https://x/back.png", Insert: []string{}},
				Tracks: []manifest.Track{
					{TrackID: "a1", Number: "A1", Name: "Opening", Duration: "3'30\"", File: "https://x/opening.mp3"},
				},
			},
		},
	}
	return m, priv
}

func resign(t *testing.T, m *manifest.Manifest, priv ed25519.PrivateKey) {
	t.Helper()
	canonicalBytes, err := signing.CanonicalWithoutField(m, "signature")
	if err != nil {
		t.Fatalf("CanonicalWithoutField: %v", err)
	}
	m.Signature = manifest.Signature{Algorithm: "ed25519", Value: signing.SignCanonical(priv, canonicalBytes)}
}

func serveManifest(t *testing.T, m *manifest.Manifest, entries *[]manifest.HistoryEntry) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/history.json" {
			_ = json.NewEncoder(w).Encode(*entries)
			return
		}
		_ = json.NewEncoder(w).Encode(*m)
	}))
	m.Identity.URL = server.URL
	m.History.URL = server.URL + "/history.json"
	return server
}

// signedHistoryEntry builds a single release_added entry signed by priv,
// matching the one album buildSignedManifest's manifest declares, so the
// manifest's history chain verifies.
func signedHistoryEntry(t *testing.T, priv ed25519.PrivateKey) manifest.HistoryEntry {
	t.Helper()
	entry := manifest.HistoryEntry{
		Timestamp: "2024-01-01T00:00:00Z",
		Type:      manifest.HistoryTypeReleaseAdded,
		Data:      map[string]any{"albumId": "agency-2024"},
	}
	canonicalEntry, err := signing.CanonicalWithoutField(entry, "signature")
	if err != nil {
		t.Fatalf("CanonicalWithoutField: %v", err)
	}
	entry.Signature = manifest.Signature{Algorithm: "ed25519", Value: signing.SignCanonical(priv, canonicalEntry)}
	return entry
}

func TestPollOne_ValidManifestIndexes(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	m, priv := buildSignedManifest(t)
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
		t.Fatalf("PollOne: %v", err)
	}

	got, err := s.GetIdentity(ctx, server.URL)
	if err != nil {
		t.Fatalf("GetIdentity: %v", err)
	}
	if got == nil {
		t.Fatal("GetIdentity returned nil after a successful PollOne")
	}
	if got.Name != "Ligatures" {
		t.Errorf("indexed identity name = %q, want Ligatures", got.Name)
	}
}

func TestPollOne_InvalidManifestDoesNotIndex(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	m, _ := buildSignedManifest(t)
	m.Identity.ContactEmail = "" // makes this manifest fail required-field validation
	entries := []manifest.HistoryEntry{}
	server := serveManifest(t, &m, &entries)
	defer server.Close()

	c := New(s)
	url := server.URL + "/.well-known/endonend/manifest.json"
	if err := c.PollOne(ctx, url); err == nil {
		t.Fatal("PollOne for an invalid manifest: want error, got nil")
	}

	got, err := s.GetIdentity(ctx, server.URL)
	if err != nil {
		t.Fatalf("GetIdentity: %v", err)
	}
	if got != nil {
		t.Error("GetIdentity after a failed first-time PollOne: want nil, got a result")
	}
}

func TestPollOne_FailedRecrawlKeepsOldDataAndRecordsError(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	m, priv := buildSignedManifest(t)
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
		t.Fatalf("first PollOne: %v", err)
	}

	// Break the served manifest (bad signature) and re-poll the same URL.
	m.Signature.Value = "not-a-real-signature"
	if err := c.PollOne(ctx, url); err == nil {
		t.Fatal("second PollOne with a broken manifest: want error, got nil")
	}

	got, err := s.GetIdentity(ctx, server.URL)
	if err != nil {
		t.Fatalf("GetIdentity: %v", err)
	}
	if got == nil {
		t.Fatal("GetIdentity after a failed re-crawl: want the old data to remain, got nil")
	}
	if got.Name != "Ligatures" {
		t.Errorf("identity name after failed re-crawl = %q, want the old value Ligatures to remain", got.Name)
	}
	if got.LastError == "" {
		t.Error("identity.LastError after a failed re-crawl: want a recorded reason, got empty")
	}
}

func TestPollOne_UnreachableURLRecordsError(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	c := New(s)
	if err := c.PollOne(ctx, "http://127.0.0.1:1/.well-known/endonend/manifest.json"); err == nil {
		t.Fatal("PollOne against an unreachable URL: want error, got nil")
	}
}
