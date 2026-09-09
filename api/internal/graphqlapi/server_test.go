package graphqlapi

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"endonend/api/internal/crawler"
	"endonend/api/internal/db"
	"endonend/api/internal/store"
	"endonend/protocol/manifest"
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

func doGraphQL(t *testing.T, h http.Handler, query string, headers map[string]string) map[string]any {
	t.Helper()
	body, err := json.Marshal(map[string]string{"query": query})
	if err != nil {
		t.Fatalf("marshal request body: %v", err)
	}
	req := httptest.NewRequest(http.MethodPost, "/graphql", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		return map[string]any{"httpStatus": float64(rec.Code)}
	}
	var out map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &out); err != nil {
		t.Fatalf("unmarshal response %s: %v", rec.Body.String(), err)
	}
	return out
}

func TestPublicHandler_QueryArtistAndAlbum(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	m := &manifest.Manifest{
		ManifestVersion: "1.0",
		Identity:        manifest.Identity{Type: "artist", Name: "Ligatures", URL: "https://ligatures.example", ContactEmail: "band@ligatures.example"},
		Refresh:         manifest.Refresh{TTLSeconds: 21600},
		History:         manifest.HistoryRef{URL: "https://ligatures.example/.well-known/endonend/history.json"},
		Catalog: []manifest.Album{{
			AlbumID: "agency-2024", AlbumVersion: 1, AlbumName: "Agency", ReleaseDate: "2024-05-01",
			Images: manifest.Images{Front: "https://x/front.png", Back: "https://x/back.png", Insert: []string{}},
			Tracks: []manifest.Track{{TrackID: "a1", Number: "A1", Name: "Opening", Duration: "3'30\"", File: "https://x/opening.mp3"}},
		}},
	}
	if err := s.UpsertManifest(ctx, m, []byte("{}"), nil, time.Now()); err != nil {
		t.Fatalf("seed UpsertManifest: %v", err)
	}

	h := NewPublicHandler(s)

	resp := doGraphQL(t, h, `{ artists { url name } }`, nil)
	data, ok := resp["data"].(map[string]any)
	if !ok {
		t.Fatalf("response has no data: %+v", resp)
	}
	artists, ok := data["artists"].([]any)
	if !ok || len(artists) != 1 {
		t.Fatalf("artists = %+v, want one artist", data["artists"])
	}
	artist := artists[0].(map[string]any)
	if artist["name"] != "Ligatures" {
		t.Errorf("artists[0].name = %v, want Ligatures", artist["name"])
	}

	resp = doGraphQL(t, h, `{ album(identityUrl: "https://ligatures.example", albumId: "agency-2024") { albumName tracks { name } } }`, nil)
	data = resp["data"].(map[string]any)
	album, ok := data["album"].(map[string]any)
	if !ok {
		t.Fatalf("album = %+v, want a result", data["album"])
	}
	if album["albumName"] != "Agency" {
		t.Errorf("album.albumName = %v, want Agency", album["albumName"])
	}
	tracks := album["tracks"].([]any)
	if len(tracks) != 1 || tracks[0].(map[string]any)["name"] != "Opening" {
		t.Errorf("album.tracks = %+v, want one track named Opening", tracks)
	}
}

func TestPublicHandler_SendsCORSHeaders(t *testing.T) {
	s := testStore(t)
	h := NewPublicHandler(s)

	req := httptest.NewRequest(http.MethodPost, "/graphql", bytes.NewReader([]byte(`{"query":"{ artists { url } }"}`)))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "*" {
		t.Errorf("Access-Control-Allow-Origin = %q, want \"*\"", got)
	}
}

func TestAdminHandler_PreflightSucceedsWithoutAToken(t *testing.T) {
	s := testStore(t)
	c := crawler.New(s)
	h := NewAdminHandler(s, c, "correct-secret")

	// A browser's CORS preflight never sends X-Admin-Token; the auth check
	// must not run for it, or the browser never gets to send the real
	// request at all.
	req := httptest.NewRequest(http.MethodOptions, "/admin/graphql", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	req.Header.Set("Access-Control-Request-Method", "POST")
	req.Header.Set("Access-Control-Request-Headers", "content-type,x-admin-token")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Errorf("OPTIONS preflight status = %d, want 204", rec.Code)
	}
	if got := rec.Header().Get("Access-Control-Allow-Headers"); got == "" {
		t.Error("Access-Control-Allow-Headers is empty on the preflight response")
	}
}

func TestPublicHandler_ImagesBackResolvesNullWhenAbsent(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	m := &manifest.Manifest{
		ManifestVersion: "1.0",
		Identity:        manifest.Identity{Type: "artist", Name: "Ligatures", URL: "https://ligatures.example", ContactEmail: "band@ligatures.example"},
		Refresh:         manifest.Refresh{TTLSeconds: 21600},
		History:         manifest.HistoryRef{URL: "https://ligatures.example/.well-known/endonend/history.json"},
		Catalog: []manifest.Album{{
			AlbumID: "agency-2024", AlbumVersion: 1, AlbumName: "Agency", ReleaseDate: "2024-05-01",
			Images: manifest.Images{Front: "https://x/front.png", Insert: []string{}}, // no Back
			Tracks: []manifest.Track{{TrackID: "a1", Number: "A1", Name: "Opening", File: "https://x/opening.mp3"}},
		}},
	}
	if err := s.UpsertManifest(ctx, m, []byte("{}"), nil, time.Now()); err != nil {
		t.Fatalf("seed UpsertManifest: %v", err)
	}

	h := NewPublicHandler(s)
	resp := doGraphQL(t, h, `{ album(identityUrl: "https://ligatures.example", albumId: "agency-2024") { imagesFront imagesBack } }`, nil)
	data := resp["data"].(map[string]any)
	album := data["album"].(map[string]any)
	if album["imagesFront"] != "https://x/front.png" {
		t.Errorf("album.imagesFront = %v, want https://x/front.png", album["imagesFront"])
	}
	if album["imagesBack"] != nil {
		t.Errorf("album.imagesBack = %v, want null when no back cover is declared", album["imagesBack"])
	}
}

func TestAdminHandler_RequiresSecret(t *testing.T) {
	s := testStore(t)
	c := crawler.New(s)
	h := NewAdminHandler(s, c, "correct-secret")

	resp := doGraphQL(t, h, `{ artists { url } }`, nil)
	if resp["httpStatus"] != float64(http.StatusUnauthorized) {
		t.Errorf("request with no token: httpStatus = %v, want 401", resp["httpStatus"])
	}

	resp = doGraphQL(t, h, `{ artists { url } }`, map[string]string{"X-Admin-Token": "wrong"})
	if resp["httpStatus"] != float64(http.StatusUnauthorized) {
		t.Errorf("request with wrong token: httpStatus = %v, want 401", resp["httpStatus"])
	}

	resp = doGraphQL(t, h, `{ artists { url } }`, map[string]string{"X-Admin-Token": "correct-secret"})
	if _, ok := resp["data"]; !ok {
		t.Errorf("request with correct token: want a data field, got %+v", resp)
	}
}

func TestAdminHandler_EmptySecretAlwaysRejects(t *testing.T) {
	s := testStore(t)
	c := crawler.New(s)
	h := NewAdminHandler(s, c, "")

	resp := doGraphQL(t, h, `{ artists { url } }`, map[string]string{"X-Admin-Token": ""})
	if resp["httpStatus"] != float64(http.StatusUnauthorized) {
		t.Errorf("admin handler with empty configured secret: httpStatus = %v, want 401 (must fail closed)", resp["httpStatus"])
	}
}

func TestAdminHandler_SubmitManifestUrlMutation(t *testing.T) {
	s := testStore(t)
	c := crawler.New(s)
	h := NewAdminHandler(s, c, "secret")

	m, priv := buildTestManifest(t)
	entries := []manifest.HistoryEntry{}
	server := serveTestManifest(t, &m, &entries)
	defer server.Close()
	entry := signTestManifest(t, &m, priv)
	entries = append(entries, entry)

	query := `mutation { submitManifestUrl(url: "` + server.URL + `") { accepted errors identity { name } } }`
	resp := doGraphQL(t, h, query, map[string]string{"X-Admin-Token": "secret"})
	data, ok := resp["data"].(map[string]any)
	if !ok {
		t.Fatalf("response has no data: %+v", resp)
	}
	result := data["submitManifestUrl"].(map[string]any)
	if result["accepted"] != true {
		t.Errorf("submitManifestUrl accepted = %v, errors = %v, want true", result["accepted"], result["errors"])
	}
	identity := result["identity"].(map[string]any)
	if identity["name"] != "Ligatures" {
		t.Errorf("submitManifestUrl identity.name = %v, want Ligatures", identity["name"])
	}
}

func TestAdminHandler_ValidateManifestMutation(t *testing.T) {
	s := testStore(t)
	c := crawler.New(s)
	h := NewAdminHandler(s, c, "secret")

	query := `mutation { validateManifest(rawJson: "not json") { valid errors { field message } } }`
	resp := doGraphQL(t, h, query, map[string]string{"X-Admin-Token": "secret"})
	data := resp["data"].(map[string]any)
	result := data["validateManifest"].(map[string]any)
	if result["valid"] != false {
		t.Errorf("validateManifest of garbage JSON: valid = %v, want false", result["valid"])
	}
	errs := result["errors"].([]any)
	if len(errs) == 0 {
		t.Error("validateManifest of garbage JSON: want at least one error, got none")
	}
}
