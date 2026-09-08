package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseValidateArgs_FlagsBeforeTarget(t *testing.T) {
	got, err := parseValidateArgs([]string{"--deep", "--json", "./manifest.json"})
	if err != nil {
		t.Fatalf("parseValidateArgs returned error: %v", err)
	}
	want := validateArgs{target: "./manifest.json", deep: true, json: true}
	if got != want {
		t.Errorf("parseValidateArgs() = %+v, want %+v", got, want)
	}
}

func TestParseValidateArgs_FlagsAfterTarget(t *testing.T) {
	// 0004's own usage example puts flags after the positional target:
	// `validate <path-or-url> [--deep] [--json]`.
	got, err := parseValidateArgs([]string{"./manifest.json", "--deep"})
	if err != nil {
		t.Fatalf("parseValidateArgs returned error: %v", err)
	}
	want := validateArgs{target: "./manifest.json", deep: true, json: false}
	if got != want {
		t.Errorf("parseValidateArgs() = %+v, want %+v", got, want)
	}
}

func TestParseValidateArgs_NoTargetErrors(t *testing.T) {
	if _, err := parseValidateArgs([]string{"--deep"}); err == nil {
		t.Error("parseValidateArgs with no target: want error, got nil")
	}
}

func TestParseValidateArgs_TwoTargetsErrors(t *testing.T) {
	if _, err := parseValidateArgs([]string{"a.json", "b.json"}); err == nil {
		t.Error("parseValidateArgs with two targets: want error, got nil")
	}
}

func TestParseValidateArgs_UnknownFlagErrors(t *testing.T) {
	if _, err := parseValidateArgs([]string{"--bogus", "a.json"}); err == nil {
		t.Error("parseValidateArgs with an unknown flag: want error, got nil")
	}
}

func TestCmdValidate_ValidManifestExitsZero(t *testing.T) {
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
		t.Fatalf("setup cmdGenerate failed with exit code %d", code)
	}

	code := captureExitCode(t, func() int {
		return cmdValidate([]string{manifestOut})
	})
	if code != 0 {
		t.Errorf("cmdValidate exit code = %d, want 0 for a freshly generated manifest", code)
	}
}

func TestCmdValidate_MissingFileExitsOne(t *testing.T) {
	code := captureExitCode(t, func() int {
		return cmdValidate([]string{filepath.Join(t.TempDir(), "nope.json")})
	})
	if code != 1 {
		t.Errorf("cmdValidate exit code = %d, want 1 for a missing file", code)
	}
}

func TestCmdValidate_BadArgsExitsOne(t *testing.T) {
	code := captureExitCode(t, func() int {
		return cmdValidate(nil)
	})
	if code != 1 {
		t.Errorf("cmdValidate exit code = %d, want 1 with no target argument", code)
	}
}
