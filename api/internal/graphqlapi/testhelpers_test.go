package graphqlapi

import (
	"crypto/ed25519"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"endonend/protocol/history"
	"endonend/protocol/manifest"
	"endonend/protocol/signing"
)

// buildTestManifest returns a minimal, unsigned manifest and the key that
// will sign it, structurally valid except for identity.url/history.url,
// which the caller fills in once it knows the httptest.Server's address.
func buildTestManifest(t *testing.T) (manifest.Manifest, ed25519.PrivateKey) {
	t.Helper()
	pub, priv, err := signing.GenerateKeypair()
	if err != nil {
		t.Fatalf("GenerateKeypair: %v", err)
	}
	m := manifest.Manifest{
		ManifestVersion: "1.0",
		Identity:        manifest.Identity{Type: "artist", Name: "Ligatures", PublicKey: signing.EncodePublicKey(pub), ContactEmail: "band@ligatures.example"},
		Refresh:         manifest.Refresh{TTLSeconds: 21600},
		Catalog: []manifest.Album{{
			AlbumID: "agency-2024", AlbumVersion: 1, AlbumName: "Agency", ReleaseDate: "2024-05-01",
			Images: manifest.Images{Front: "https://x/front.png", Back: "https://x/back.png", Insert: []string{}},
			Tracks: []manifest.Track{{TrackID: "a1", Number: "A1", Name: "Opening", Duration: "3'30\"", File: "https://x/opening.mp3"}},
		}},
	}
	return m, priv
}

// serveTestManifest starts a server for m and *entries, and sets
// m.Identity.URL/m.History.URL to the server's own address, matching the
// "fetch location must equal identity.url" validation rule.
func serveTestManifest(t *testing.T, m *manifest.Manifest, entries *[]manifest.HistoryEntry) *httptest.Server {
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

// signTestManifest signs a release_added history entry matching m's one
// album, sets m.History.HeadHash, and signs m itself, all with priv, so the
// served manifest passes the full validator including its history chain.
func signTestManifest(t *testing.T, m *manifest.Manifest, priv ed25519.PrivateKey) manifest.HistoryEntry {
	t.Helper()
	entry := manifest.HistoryEntry{
		Timestamp: "2024-01-01T00:00:00Z",
		Type:      manifest.HistoryTypeReleaseAdded,
		Data:      map[string]any{"albumId": m.Catalog[0].AlbumID},
	}
	canonicalEntry, err := signing.CanonicalWithoutField(entry, "signature")
	if err != nil {
		t.Fatalf("CanonicalWithoutField(entry): %v", err)
	}
	entry.Signature = manifest.Signature{Algorithm: "ed25519", Value: signing.SignCanonical(priv, canonicalEntry)}

	head, err := history.Hash(entry)
	if err != nil {
		t.Fatalf("history.Hash: %v", err)
	}
	m.History.HeadHash = head

	canonicalManifest, err := signing.CanonicalWithoutField(m, "signature")
	if err != nil {
		t.Fatalf("CanonicalWithoutField(manifest): %v", err)
	}
	m.Signature = manifest.Signature{Algorithm: "ed25519", Value: signing.SignCanonical(priv, canonicalManifest)}
	return entry
}
