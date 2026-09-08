package main

import (
	"encoding/json"
	"fmt"
	"html"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"endonend/cli/internal/manifest"
)

// bandcampFixtureServer serves a minimal Bandcamp-shaped album page whose
// embedded track file / og:image URLs point back at the same server, so
// the whole import (fetch page, then download assets) stays local.
func bandcampFixtureServer(t *testing.T, artist, title string, trackTitles []string) *httptest.Server {
	t.Helper()
	var mux http.ServeMux
	server := httptest.NewServer(&mux)
	t.Cleanup(server.Close)

	tracks := make([]map[string]any, len(trackTitles))
	for i, title := range trackTitles {
		path := fmt.Sprintf("/track%d.mp3", i+1)
		mux.HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte("fake mp3 bytes"))
		})
		tracks[i] = map[string]any{
			"title": title, "track_num": i + 1, "duration": 120.0 + float64(i),
			"file": map[string]string{"mp3-128": server.URL + path},
		}
	}
	mux.HandleFunc("/art.jpg", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("fake jpeg bytes"))
	})

	tralbum := map[string]any{
		"artist": artist,
		"current": map[string]any{
			"title": title, "release_date": "27 Sep 2017 08:39:16 GMT",
		},
		"trackinfo": tracks,
	}
	raw, err := json.Marshal(tralbum)
	if err != nil {
		t.Fatalf("marshal tralbum: %v", err)
	}
	page := fmt.Sprintf(`<!doctype html><html><head>
<meta property="og:image" content="%s/art.jpg">
</head><body><div data-tralbum="%s"></div></body></html>`, server.URL, html.EscapeString(string(raw)))

	mux.HandleFunc("/album/", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(page))
	})
	return server
}

func TestCmdImportBandcamp_FullImportWithDownload(t *testing.T) {
	server := bandcampFixtureServer(t, "fatal flaw", "Demo", []string{"PARACIDIC", "LOST AND FOUND"})
	dir := t.TempDir()
	sourceOut := filepath.Join(dir, "endonend.source.json")
	downloadDir := filepath.Join(dir, "downloads")

	code := captureExitCode(t, func() int {
		return cmdImport([]string{"bandcamp",
			"--url", "https://ligatures.example",
			"--contact-email", "band@ligatures.example",
			"--source", sourceOut,
			"--download-dir", downloadDir,
			server.URL + "/album/demo",
		})
	})
	if code != 0 {
		t.Fatalf("cmdImport exit code = %d, want 0", code)
	}

	src := loadSourceFile(sourceOut)
	if src == nil {
		t.Fatal("source file was not written")
	}
	if src.Identity.Name != "fatal flaw" || src.Identity.URL != "https://ligatures.example" {
		t.Errorf("identity = %+v", src.Identity)
	}
	if len(src.Catalog) != 1 {
		t.Fatalf("got %d albums, want 1", len(src.Catalog))
	}
	album := src.Catalog[0]
	if album.AlbumID != "demo" || album.AlbumName != "Demo" || album.ReleaseDate != "2017-09-27" {
		t.Errorf("album = %+v", album)
	}
	wantBase := "https://ligatures.example/albums/demo"
	if album.Images.Front != wantBase+"/cover.jpg" || album.Images.Back != wantBase+"/cover.jpg" {
		t.Errorf("images = %+v, want both pointing at %s/cover.jpg", album.Images, wantBase)
	}
	if len(album.Tracks) != 2 {
		t.Fatalf("got %d tracks, want 2", len(album.Tracks))
	}
	if album.Tracks[0].Name != "PARACIDIC" || album.Tracks[0].File != wantBase+"/01-paracidic.mp3" {
		t.Errorf("track 0 = %+v", album.Tracks[0])
	}

	for _, name := range []string{"cover.jpg", "01-paracidic.mp3", "02-lost-and-found.mp3"} {
		p := filepath.Join(downloadDir, "demo", name)
		if _, err := os.Stat(p); err != nil {
			t.Errorf("expected downloaded file %s: %v", p, err)
		}
	}
}

func TestCmdImportBandcamp_SkipDownload(t *testing.T) {
	server := bandcampFixtureServer(t, "fatal flaw", "Demo", []string{"PARACIDIC"})
	dir := t.TempDir()
	sourceOut := filepath.Join(dir, "endonend.source.json")
	downloadDir := filepath.Join(dir, "downloads")

	code := captureExitCode(t, func() int {
		return cmdImport([]string{"bandcamp",
			"--url", "https://ligatures.example",
			"--contact-email", "band@ligatures.example",
			"--source", sourceOut,
			"--download-dir", downloadDir,
			"--skip-download",
			server.URL + "/album/demo",
		})
	})
	if code != 0 {
		t.Fatalf("cmdImport exit code = %d, want 0", code)
	}
	if _, err := os.Stat(downloadDir); err == nil {
		t.Error("--skip-download still created a download directory")
	}
}

func TestCmdImportBandcamp_RequiresURLAndEmailForNewSource(t *testing.T) {
	server := bandcampFixtureServer(t, "fatal flaw", "Demo", []string{"PARACIDIC"})
	dir := t.TempDir()
	sourceOut := filepath.Join(dir, "endonend.source.json")

	code := captureExitCode(t, func() int {
		return cmdImport([]string{"bandcamp", "--source", sourceOut, server.URL + "/album/demo"})
	})
	if code != 1 {
		t.Errorf("cmdImport exit code = %d, want 1 without --url/--contact-email for a new source", code)
	}
}

func TestCmdImportBandcamp_UpsertsIntoExistingSource(t *testing.T) {
	server1 := bandcampFixtureServer(t, "fatal flaw", "Demo", []string{"PARACIDIC"})
	server2 := bandcampFixtureServer(t, "fatal flaw", "Second Album", []string{"NEW SONG"})
	dir := t.TempDir()
	sourceOut := filepath.Join(dir, "endonend.source.json")
	downloadDir := filepath.Join(dir, "downloads")

	run := func(server *httptest.Server) int {
		return captureExitCode(t, func() int {
			return cmdImport([]string{"bandcamp",
				"--url", "https://ligatures.example",
				"--contact-email", "band@ligatures.example",
				"--source", sourceOut,
				"--download-dir", downloadDir,
				server.URL + "/album/x",
			})
		})
	}
	if code := run(server1); code != 0 {
		t.Fatalf("first import exit code = %d", code)
	}
	if code := run(server2); code != 0 {
		t.Fatalf("second import exit code = %d", code)
	}

	src := loadSourceFile(sourceOut)
	if src == nil || len(src.Catalog) != 2 {
		t.Fatalf("catalog = %+v, want 2 albums after importing two different releases", src)
	}

	// Re-importing the same album again should replace, not duplicate.
	if code := run(server1); code != 0 {
		t.Fatalf("re-import exit code = %d", code)
	}
	src = loadSourceFile(sourceOut)
	if len(src.Catalog) != 2 {
		t.Errorf("catalog has %d albums after re-importing an existing one, want still 2", len(src.Catalog))
	}
}

func TestCmdImportBandcamp_ExistingSourceKeepsIdentityWithoutFlags(t *testing.T) {
	server := bandcampFixtureServer(t, "fatal flaw", "Demo", []string{"PARACIDIC"})
	dir := t.TempDir()
	sourceOut := filepath.Join(dir, "endonend.source.json")

	if err := writeSourceFile(sourceOut, &manifest.Source{
		ManifestVersion: "1.0",
		Identity: manifest.SourceIdentity{
			Type: "artist", Name: "Existing Name", URL: "https://existing.example", ContactEmail: "old@existing.example",
		},
		Refresh: manifest.Refresh{TTLSeconds: 21600},
	}); err != nil {
		t.Fatalf("seed source: %v", err)
	}

	code := captureExitCode(t, func() int {
		return cmdImport([]string{"bandcamp", "--source", sourceOut, server.URL + "/album/demo"})
	})
	if code != 0 {
		t.Fatalf("cmdImport exit code = %d, want 0 reusing an existing identity", code)
	}
	src := loadSourceFile(sourceOut)
	if src.Identity.Name != "Existing Name" || src.Identity.URL != "https://existing.example" {
		t.Errorf("identity = %+v, want it left untouched", src.Identity)
	}
}

func TestCmdImport_UnknownSourceErrors(t *testing.T) {
	code := captureExitCode(t, func() int { return cmdImport([]string{"spotify", "http://example.com"}) })
	if code != 1 {
		t.Errorf("cmdImport([\"spotify\", ...]) = %d, want 1", code)
	}
	code = captureExitCode(t, func() int { return cmdImport(nil) })
	if code != 1 {
		t.Errorf("cmdImport(nil) = %d, want 1", code)
	}
}

func TestParseImportBandcampArgs_Defaults(t *testing.T) {
	got, err := parseImportBandcampArgs([]string{"https://x.bandcamp.com/album/y"})
	if err != nil {
		t.Fatalf("parseImportBandcampArgs returned error: %v", err)
	}
	if got.albumURL != "https://x.bandcamp.com/album/y" || got.identityType != "artist" || got.downloadDir != "bandcamp-import" {
		t.Errorf("parseImportBandcampArgs() = %+v", got)
	}
}

func TestParseImportBandcampArgs_NoURLErrors(t *testing.T) {
	if _, err := parseImportBandcampArgs([]string{"--url", "https://x.example"}); err == nil {
		t.Error("parseImportBandcampArgs with no album URL: want error, got nil")
	}
}
