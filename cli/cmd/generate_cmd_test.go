package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

const minimalSourceJSON = `{
  "manifestVersion": "1.0",
  "identity": {"type":"artist","name":"Ligatures","url":"https://ligatures.example","contactEmail":"band@ligatures.example"},
  "refresh": {"ttlSeconds": 21600},
  "catalog": [
    {"albumId":"agency-2024","albumVersion":1,"albumName":"Agency","releaseDate":"2024-05-01",
     "images":{"front":"https://x/front.png","back":"https://x/back.png","insert":[]},
     "tracks":[{"trackId":"a1","number":"A1","name":"Opening","duration":"3'30\"","file":"https://x/opening.mp3"}]}
  ]
}`

func TestParseGenerateArgs_Defaults(t *testing.T) {
	opts, err := parseGenerateArgs(nil)
	if err != nil {
		t.Fatalf("parseGenerateArgs returned error: %v", err)
	}
	if opts.SourcePath != sourcePath || opts.ManifestOutPath != manifestOutPath || opts.HistoryOutPath != historyOutPath {
		t.Errorf("parseGenerateArgs(nil) = %+v, want the package defaults", opts)
	}
}

func TestParseGenerateArgs_CustomFlags(t *testing.T) {
	opts, err := parseGenerateArgs([]string{"--source", "s.json", "--manifest-out", "m.json", "--history-out", "h.json"})
	if err != nil {
		t.Fatalf("parseGenerateArgs returned error: %v", err)
	}
	if opts.SourcePath != "s.json" || opts.ManifestOutPath != "m.json" || opts.HistoryOutPath != "h.json" {
		t.Errorf("parseGenerateArgs() = %+v, want custom paths", opts)
	}
}

func TestParseGenerateArgs_UnknownFlagErrors(t *testing.T) {
	if _, err := parseGenerateArgs([]string{"--nope"}); err == nil {
		t.Error("parseGenerateArgs(--nope): want error, got nil")
	}
}

func TestCmdGenerate_Success(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	dir := t.TempDir()
	src := filepath.Join(dir, "union.source.json")
	if err := os.WriteFile(src, []byte(minimalSourceJSON), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}
	manifestOut := filepath.Join(dir, "manifest.json")
	historyOut := filepath.Join(dir, "history.json")

	code := captureExitCode(t, func() int {
		return cmdGenerate([]string{"--source", src, "--manifest-out", manifestOut, "--history-out", historyOut})
	})
	if code != 0 {
		t.Fatalf("cmdGenerate exit code = %d, want 0", code)
	}
	if _, err := os.Stat(manifestOut); err != nil {
		t.Errorf("manifest not written: %v", err)
	}

	var m map[string]any
	raw, err := os.ReadFile(manifestOut)
	if err != nil {
		t.Fatalf("read manifest: %v", err)
	}
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatalf("manifest is not valid JSON: %v", err)
	}
}

func TestCmdGenerate_MissingSourceFails(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	code := captureExitCode(t, func() int {
		return cmdGenerate([]string{"--source", filepath.Join(t.TempDir(), "nope.json")})
	})
	if code != 1 {
		t.Errorf("cmdGenerate exit code = %d, want 1 for a missing source file", code)
	}
}

// captureExitCode runs f while discarding whatever it prints to stdout, so
// test output stays focused on assertions.
func captureExitCode(t *testing.T, f func() int) int {
	t.Helper()
	var code int
	captureStdout(t, func() { code = f() })
	return code
}
