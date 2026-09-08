package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"endonend/cli/internal/manifest"
)

func TestIsURL(t *testing.T) {
	cases := map[string]bool{
		"https://example.com": true,
		"http://example.com":  true,
		"./manifest.json":     false,
		"manifest.json":       false,
		"ftp://example.com":   false,
		"":                    false,
	}
	for in, want := range cases {
		if got := isURL(in); got != want {
			t.Errorf("isURL(%q) = %v, want %v", in, got, want)
		}
	}
}

func chdir(t *testing.T, dir string) {
	t.Helper()
	orig, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("Chdir: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(orig) })
}

func TestLoadExistingSource_MissingFileReturnsNil(t *testing.T) {
	chdir(t, t.TempDir())
	if src := loadExistingSource(); src != nil {
		t.Errorf("loadExistingSource() = %+v, want nil when the file is missing", src)
	}
}

func TestLoadExistingSource_ReadsValidFile(t *testing.T) {
	chdir(t, t.TempDir())
	if err := os.WriteFile(sourcePath, []byte(minimalSourceJSON), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}
	src := loadExistingSource()
	if src == nil || src.Identity.Name != "Ligatures" {
		t.Errorf("loadExistingSource() = %+v, want identity.name \"Ligatures\"", src)
	}
}

func TestLoadExistingSource_MalformedFileReturnsNil(t *testing.T) {
	chdir(t, t.TempDir())
	if err := os.WriteFile(sourcePath, []byte("{not json"), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}
	if src := loadExistingSource(); src != nil {
		t.Errorf("loadExistingSource() = %+v, want nil for malformed JSON", src)
	}
}

func TestMenuAddAlbum_BuildsAlbumFromScriptedInput(t *testing.T) {
	lines := []string{
		"agency-2024", "Agency", "2024-05-01",
		"https://x/front.png", "https://x/back.png",
		"n", // no insert image
		"y", // add a track
		"a1", "", "Opening", "", "https://x/opening.mp3",
		"", // don't add another track (default false)
	}
	p := newTestPrompter(strings.Join(lines, "\n") + "\n")

	a := menuAddAlbum(p)

	if a.AlbumID != "agency-2024" || a.AlbumName != "Agency" || a.ReleaseDate != "2024-05-01" {
		t.Errorf("album identity/metadata = %+v", a)
	}
	if a.Images.Front != "https://x/front.png" || a.Images.Back != "https://x/back.png" {
		t.Errorf("album images = %+v", a.Images)
	}
	if len(a.Images.Insert) != 0 {
		t.Errorf("Images.Insert = %v, want empty", a.Images.Insert)
	}
	if len(a.Tracks) != 1 {
		t.Fatalf("got %d tracks, want 1", len(a.Tracks))
	}
	tr := a.Tracks[0]
	if tr.TrackID != "a1" || tr.Number != "1" || tr.Name != "Opening" || tr.File != "https://x/opening.mp3" {
		t.Errorf("track = %+v", tr)
	}
}

func TestMenuImportBandcamp_CreatesSourceWhenNoneExists(t *testing.T) {
	chdir(t, t.TempDir())
	server := bandcampFixtureServer(t, "fatal flaw", "Demo", []string{"PARACIDIC"})

	lines := []string{
		server.URL + "/album/demo",
		"",                          // default artist type
		"https://ligatures.example", // identity url
		"band@ligatures.example",    // contact email
		"n",                         // skip download
	}
	p := newTestPrompter(strings.Join(lines, "\n") + "\n")

	out := captureStdout(t, func() { menuImportBandcamp(p) })
	if strings.Contains(out, "Error") {
		t.Errorf("menuImportBandcamp printed an error:\n%s", out)
	}

	src := loadExistingSource()
	if src == nil {
		t.Fatal("endonend.source.json was not created")
	}
	if src.Identity.URL != "https://ligatures.example" || src.Identity.ContactEmail != "band@ligatures.example" {
		t.Errorf("identity = %+v", src.Identity)
	}
	if len(src.Catalog) != 1 || src.Catalog[0].AlbumName != "Demo" {
		t.Errorf("catalog = %+v", src.Catalog)
	}
	if _, err := os.Stat("bandcamp-import"); err == nil {
		t.Error("answering 'n' to download still created a download directory")
	}
}

func TestMenuImportBandcamp_ReusesExistingSourceAndDownloads(t *testing.T) {
	chdir(t, t.TempDir())
	if err := writeSourceFile(sourcePath, &manifest.Source{
		ManifestVersion: "1.0",
		Identity: manifest.SourceIdentity{
			Type: "artist", Name: "Existing", URL: "https://existing.example", ContactEmail: "old@existing.example",
		},
		Refresh: manifest.Refresh{TTLSeconds: 21600},
	}); err != nil {
		t.Fatalf("seed source: %v", err)
	}
	server := bandcampFixtureServer(t, "fatal flaw", "Demo", []string{"PARACIDIC"})

	lines := []string{
		server.URL + "/album/demo",
		"y", // download this time
	}
	p := newTestPrompter(strings.Join(lines, "\n") + "\n")
	out := captureStdout(t, func() { menuImportBandcamp(p) })
	if strings.Contains(out, "Error") {
		t.Errorf("menuImportBandcamp printed an error:\n%s", out)
	}

	src := loadExistingSource()
	if src.Identity.URL != "https://existing.example" {
		t.Errorf("identity.URL = %q, want the existing value left untouched", src.Identity.URL)
	}
	if len(src.Catalog) != 1 {
		t.Fatalf("catalog = %+v, want 1 album", src.Catalog)
	}
	if _, err := os.Stat(filepath.Join("bandcamp-import", "demo", "cover.jpg")); err != nil {
		t.Errorf("expected downloaded cover art: %v", err)
	}
}
