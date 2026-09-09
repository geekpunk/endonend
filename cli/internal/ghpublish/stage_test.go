package ghpublish

import (
	"os"
	"path/filepath"
	"testing"

	"endonend/protocol/manifest"
)

func writeTestFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir for %s: %v", path, err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func TestStage_AssemblesSiteTreeFromDeclaredAssets(t *testing.T) {
	dir := t.TempDir()
	manifestPath := filepath.Join(dir, "manifest.json")
	historyPath := filepath.Join(dir, "history.json")
	assetsDir := filepath.Join(dir, "bandcamp-import")
	outDir := filepath.Join(dir, "publish")

	writeTestFile(t, manifestPath, `{"signed":true}`)
	writeTestFile(t, historyPath, `[]`)
	writeTestFile(t, filepath.Join(assetsDir, "demo-album", "cover.jpg"), "cover-bytes")
	writeTestFile(t, filepath.Join(assetsDir, "demo-album", "01-opening.mp3"), "audio-bytes")
	writeTestFile(t, filepath.Join(assetsDir, "demo-album", "liner.png"), "insert-bytes")
	writeTestFile(t, filepath.Join(assetsDir, "demo-album", "demo-album.zip"), "zip-bytes")

	m := &manifest.Manifest{
		Identity: manifest.Identity{URL: "https://someartist.github.io/bandname"},
		Catalog: []manifest.Album{
			{
				AlbumID: "demo-album",
				Images: manifest.Images{
					Front:  "https://someartist.github.io/bandname/albums/demo-album/cover.jpg",
					Back:   "https://someartist.github.io/bandname/albums/demo-album/cover.jpg",
					Insert: []string{"https://someartist.github.io/bandname/albums/demo-album/liner.png"},
				},
				DownloadZip: "https://someartist.github.io/bandname/albums/demo-album/demo-album.zip",
				Tracks: []manifest.Track{
					{File: "https://someartist.github.io/bandname/albums/demo-album/01-opening.mp3"},
				},
			},
		},
	}

	err := Stage(m, StageOptions{
		ManifestPath: manifestPath,
		HistoryPath:  historyPath,
		AssetsDir:    assetsDir,
		OutDir:       outDir,
	})
	if err != nil {
		t.Fatalf("Stage: %v", err)
	}

	assertFileContent(t, filepath.Join(outDir, ".well-known", "endonend", "manifest.json"), `{"signed":true}`)
	assertFileContent(t, filepath.Join(outDir, ".well-known", "endonend", "history.json"), `[]`)
	assertFileContent(t, filepath.Join(outDir, "albums", "demo-album", "cover.jpg"), "cover-bytes")
	assertFileContent(t, filepath.Join(outDir, "albums", "demo-album", "01-opening.mp3"), "audio-bytes")
	assertFileContent(t, filepath.Join(outDir, "albums", "demo-album", "liner.png"), "insert-bytes")
	assertFileContent(t, filepath.Join(outDir, "albums", "demo-album", "demo-album.zip"), "zip-bytes")
}

func TestStage_SkipsAssetsHostedElsewhere(t *testing.T) {
	dir := t.TempDir()
	manifestPath := filepath.Join(dir, "manifest.json")
	historyPath := filepath.Join(dir, "history.json")
	writeTestFile(t, manifestPath, `{}`)
	writeTestFile(t, historyPath, `[]`)
	outDir := filepath.Join(dir, "publish")

	m := &manifest.Manifest{
		Identity: manifest.Identity{URL: "https://someartist.github.io/bandname"},
		Catalog: []manifest.Album{
			{
				AlbumID: "demo-album",
				Images: manifest.Images{
					Front: "https://cdn.somehost.example/cover.jpg", // hosted elsewhere, not this identity's own URL
					Back:  "https://cdn.somehost.example/cover.jpg",
				},
			},
		},
	}

	if err := Stage(m, StageOptions{ManifestPath: manifestPath, HistoryPath: historyPath, AssetsDir: dir, OutDir: outDir}); err != nil {
		t.Fatalf("Stage: %v", err)
	}

	if _, err := os.Stat(filepath.Join(outDir, "albums")); !os.IsNotExist(err) {
		t.Errorf("expected no albums/ directory to be created for externally-hosted assets, stat err = %v", err)
	}
}

func TestStage_MissingLocalAssetReturnsError(t *testing.T) {
	dir := t.TempDir()
	manifestPath := filepath.Join(dir, "manifest.json")
	historyPath := filepath.Join(dir, "history.json")
	writeTestFile(t, manifestPath, `{}`)
	writeTestFile(t, historyPath, `[]`)

	m := &manifest.Manifest{
		Identity: manifest.Identity{URL: "https://someartist.github.io/bandname"},
		Catalog: []manifest.Album{
			{
				AlbumID: "demo-album",
				Images:  manifest.Images{Front: "https://someartist.github.io/bandname/albums/demo-album/cover.jpg"},
			},
		},
	}

	err := Stage(m, StageOptions{
		ManifestPath: manifestPath,
		HistoryPath:  historyPath,
		AssetsDir:    filepath.Join(dir, "does-not-exist"),
		OutDir:       filepath.Join(dir, "publish"),
	})
	if err == nil {
		t.Fatal("Stage with a missing local asset: want error, got nil")
	}
}

func TestStage_DeduplicatesRepeatedAssetURLs(t *testing.T) {
	// Two catalog fields pointing at the same URL (an artist could set
	// images.front and images.back to the same file deliberately) must
	// not fail on double-copying the same file.
	dir := t.TempDir()
	manifestPath := filepath.Join(dir, "manifest.json")
	historyPath := filepath.Join(dir, "history.json")
	assetsDir := filepath.Join(dir, "bandcamp-import")
	writeTestFile(t, manifestPath, `{}`)
	writeTestFile(t, historyPath, `[]`)
	writeTestFile(t, filepath.Join(assetsDir, "demo-album", "cover.jpg"), "cover-bytes")

	m := &manifest.Manifest{
		Identity: manifest.Identity{URL: "https://someartist.github.io/bandname"},
		Catalog: []manifest.Album{
			{
				AlbumID: "demo-album",
				Images: manifest.Images{
					Front: "https://someartist.github.io/bandname/albums/demo-album/cover.jpg",
					Back:  "https://someartist.github.io/bandname/albums/demo-album/cover.jpg",
				},
			},
		},
	}

	outDir := filepath.Join(dir, "publish")
	if err := Stage(m, StageOptions{ManifestPath: manifestPath, HistoryPath: historyPath, AssetsDir: assetsDir, OutDir: outDir}); err != nil {
		t.Fatalf("Stage: %v", err)
	}
	assertFileContent(t, filepath.Join(outDir, "albums", "demo-album", "cover.jpg"), "cover-bytes")
}

func assertFileContent(t *testing.T, path, want string) {
	t.Helper()
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	if string(got) != want {
		t.Errorf("content of %s = %q, want %q", path, got, want)
	}
}
