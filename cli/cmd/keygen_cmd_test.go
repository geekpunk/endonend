package main

import (
	"os"
	"path/filepath"
	"testing"

	"endonend/cli/internal/signing"
)

func TestParseKeygenArgs_RequiresURLUnlessRotating(t *testing.T) {
	if _, err := parseKeygenArgs(nil); err == nil {
		t.Error("parseKeygenArgs(nil): want error, got nil")
	}
	got, err := parseKeygenArgs([]string{"--url", "https://ligatures.example"})
	if err != nil {
		t.Fatalf("parseKeygenArgs returned error: %v", err)
	}
	if got.url != "https://ligatures.example" || got.rotate {
		t.Errorf("parseKeygenArgs() = %+v, want url set and rotate false", got)
	}
}

func TestParseKeygenArgs_RotateNeedsNoURL(t *testing.T) {
	got, err := parseKeygenArgs([]string{"--rotate"})
	if err != nil {
		t.Fatalf("parseKeygenArgs returned error: %v", err)
	}
	if !got.rotate || got.manifestOut != manifestOutPath || got.historyOut != historyOutPath {
		t.Errorf("parseKeygenArgs(--rotate) = %+v, want rotate true with default paths", got)
	}
}

func TestCmdKeygen_FirstTimeGeneratesKey(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	code := captureExitCode(t, func() int {
		return cmdKeygen([]string{"--url", "https://ligatures.example"})
	})
	if code != 0 {
		t.Fatalf("cmdKeygen exit code = %d, want 0", code)
	}
	keyPath, err := signing.KeyPath("https://ligatures.example")
	if err != nil {
		t.Fatalf("KeyPath: %v", err)
	}
	if !signing.KeyExists(keyPath) {
		t.Error("cmdKeygen did not write a key file")
	}
}

func TestCmdKeygen_RefusesToOverwriteWithoutRotate(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	if code := captureExitCode(t, func() int { return cmdKeygen([]string{"--url", "https://ligatures.example"}) }); code != 0 {
		t.Fatalf("first cmdKeygen failed with exit code %d", code)
	}
	code := captureExitCode(t, func() int { return cmdKeygen([]string{"--url", "https://ligatures.example"}) })
	if code != 1 {
		t.Errorf("cmdKeygen exit code = %d, want 1 when a key already exists", code)
	}
}

func TestCmdKeygen_Rotate(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	dir := t.TempDir()
	src := filepath.Join(dir, "endonend.source.json")
	if err := os.WriteFile(src, []byte(minimalSourceJSON), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}
	manifestOut := filepath.Join(dir, "manifest.json")
	historyOut := filepath.Join(dir, "history.json")
	if code := captureExitCode(t, func() int {
		return cmdGenerate([]string{"--source", src, "--manifest-out", manifestOut, "--history-out", historyOut})
	}); code != 0 {
		t.Fatalf("setup cmdGenerate failed")
	}

	code := captureExitCode(t, func() int {
		return cmdKeygen([]string{"--rotate", "--manifest-out", manifestOut, "--history-out", historyOut})
	})
	if code != 0 {
		t.Fatalf("cmdKeygen --rotate exit code = %d, want 0", code)
	}

	if code := captureExitCode(t, func() int { return cmdValidate([]string{manifestOut}) }); code != 0 {
		t.Errorf("manifest invalid after rotation, cmdValidate exit code = %d", code)
	}
}

func TestCmdKeygen_RotateWithoutManifestFails(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	dir := t.TempDir()
	code := captureExitCode(t, func() int {
		return cmdKeygen([]string{"--rotate", "--manifest-out", filepath.Join(dir, "manifest.json"), "--history-out", filepath.Join(dir, "history.json")})
	})
	if code != 1 {
		t.Errorf("cmdKeygen --rotate exit code = %d, want 1 with no manifest to rotate", code)
	}
}
