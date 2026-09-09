package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"endonend/protocol/manifest"
)

// fakePublishRunner is a ghpublish.Runner test double recording every call
// it receives, so tests can verify publishGithub's orchestration without
// actually invoking git or gh. "gh repo view" always fails, simulating a
// repo that doesn't exist yet, so EnsureRepo's create path gets exercised.
type fakePublishRunner struct {
	calls []string
}

func (f *fakePublishRunner) Run(dir, name string, args ...string) (string, error) {
	f.calls = append(f.calls, strings.TrimSpace(name+" "+strings.Join(args, " ")))
	if name == "gh" && len(args) >= 2 && args[0] == "repo" && args[1] == "view" {
		return "", fmt.Errorf("fake: repo not found")
	}
	return "", nil
}

func (f *fakePublishRunner) called(substr string) bool {
	for _, c := range f.calls {
		if strings.Contains(c, substr) {
			return true
		}
	}
	return false
}

func writePublishFixture(t *testing.T, dir string) (manifestPath, historyPath string) {
	t.Helper()
	m := manifest.Manifest{
		ManifestVersion: "1.0",
		Identity:        manifest.Identity{Type: "artist", Name: "Test", URL: "https://someartist.github.io/bandname", ContactEmail: "band@example.com"},
		Refresh:         manifest.Refresh{TTLSeconds: 21600},
		Catalog: []manifest.Album{{
			AlbumID: "demo", AlbumVersion: 1, AlbumName: "Demo", ReleaseDate: "2024-01-01",
			Images: manifest.Images{Front: "https://someartist.github.io/bandname/albums/demo/cover.jpg", Insert: []string{}},
			Tracks: []manifest.Track{{TrackID: "t1", Number: "A1", Name: "Song", File: "https://someartist.github.io/bandname/albums/demo/01-song.mp3"}},
		}},
	}
	manifestPath = filepath.Join(dir, "manifest.json")
	historyPath = filepath.Join(dir, "history.json")
	raw, err := json.Marshal(m)
	if err != nil {
		t.Fatalf("marshal manifest: %v", err)
	}
	if err := os.WriteFile(manifestPath, raw, 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}
	if err := os.WriteFile(historyPath, []byte(`[]`), 0o644); err != nil {
		t.Fatalf("write history: %v", err)
	}

	assetsDir := filepath.Join(dir, "bandcamp-import", "demo")
	if err := os.MkdirAll(assetsDir, 0o755); err != nil {
		t.Fatalf("mkdir assets: %v", err)
	}
	if err := os.WriteFile(filepath.Join(assetsDir, "cover.jpg"), []byte("cover"), 0o644); err != nil {
		t.Fatalf("write cover: %v", err)
	}
	if err := os.WriteFile(filepath.Join(assetsDir, "01-song.mp3"), []byte("audio"), 0o644); err != nil {
		t.Fatalf("write track: %v", err)
	}
	return manifestPath, historyPath
}

func TestPublishGithub_StagesAndPushesSuccessfully(t *testing.T) {
	dir := t.TempDir()
	manifestPath, historyPath := writePublishFixture(t, dir)
	runner := &fakePublishRunner{}

	parsed := publishGithubArgs{
		manifestPath: manifestPath,
		historyPath:  historyPath,
		assetsDir:    filepath.Join(dir, "bandcamp-import"),
		outDir:       filepath.Join(dir, "publish"),
		branch:       "main",
	}

	if err := publishGithub(parsed, runner); err != nil {
		t.Fatalf("publishGithub: %v", err)
	}

	if _, err := os.Stat(filepath.Join(parsed.outDir, ".well-known", "endonend", "manifest.json")); err != nil {
		t.Errorf("staged manifest.json missing: %v", err)
	}
	if _, err := os.Stat(filepath.Join(parsed.outDir, "albums", "demo", "cover.jpg")); err != nil {
		t.Errorf("staged cover.jpg missing: %v", err)
	}

	if !runner.called("gh repo create someartist/bandname") {
		t.Errorf("calls = %v, want a gh repo create for someartist/bandname", runner.calls)
	}
	if !runner.called("git push -u origin main") {
		t.Errorf("calls = %v, want a git push", runner.calls)
	}
	if !runner.called("repos/someartist/bandname/pages") {
		t.Errorf("calls = %v, want a Pages-enable call", runner.calls)
	}
}

func TestPublishGithub_NonGitHubPagesURLErrors(t *testing.T) {
	dir := t.TempDir()
	manifestPath, historyPath := writePublishFixture(t, dir)

	// Overwrite identity.url with a non-github.io URL.
	raw, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatalf("read manifest: %v", err)
	}
	var m manifest.Manifest
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	m.Identity.URL = "https://ligatures.example"
	raw, err = json.Marshal(m)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if err := os.WriteFile(manifestPath, raw, 0o644); err != nil {
		t.Fatalf("write manifest: %v", err)
	}

	parsed := publishGithubArgs{manifestPath: manifestPath, historyPath: historyPath, outDir: filepath.Join(dir, "publish")}
	if err := publishGithub(parsed, &fakePublishRunner{}); err == nil {
		t.Fatal("publishGithub with a non-github.io identity.url: want error, got nil")
	}
}

func TestPublishGithub_MissingManifestErrors(t *testing.T) {
	dir := t.TempDir()
	parsed := publishGithubArgs{manifestPath: filepath.Join(dir, "does-not-exist.json"), outDir: filepath.Join(dir, "publish")}
	if err := publishGithub(parsed, &fakePublishRunner{}); err == nil {
		t.Fatal("publishGithub with a missing manifest: want error, got nil")
	}
}

func TestParsePublishGithubArgs_Defaults(t *testing.T) {
	a, err := parsePublishGithubArgs(nil)
	if err != nil {
		t.Fatalf("parsePublishGithubArgs: %v", err)
	}
	if a.manifestPath != manifestOutPath || a.historyPath != historyOutPath {
		t.Errorf("defaults = %+v, want manifest/history to match the package-wide defaults", a)
	}
	if a.assetsDir != "bandcamp-import" || a.outDir != "publish" || a.branch != "main" || a.private {
		t.Errorf("defaults = %+v, unexpected", a)
	}
}

func TestParsePublishGithubArgs_OverridesDefaults(t *testing.T) {
	a, err := parsePublishGithubArgs([]string{"--assets-dir", "assets", "--out-dir", "site", "--branch", "gh-pages", "--private"})
	if err != nil {
		t.Fatalf("parsePublishGithubArgs: %v", err)
	}
	if a.assetsDir != "assets" || a.outDir != "site" || a.branch != "gh-pages" || !a.private {
		t.Errorf("parsed = %+v, overrides did not apply", a)
	}
}
