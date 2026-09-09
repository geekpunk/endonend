package ghpublish

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestExecRunner_CapturesOutput(t *testing.T) {
	out, err := DefaultRunner.Run("", "echo", "hello")
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if strings.TrimSpace(out) != "hello" {
		t.Errorf("output = %q, want %q", out, "hello")
	}
}

func TestExecRunner_ReturnsErrorWithOutputOnFailure(t *testing.T) {
	_, err := DefaultRunner.Run("", "false")
	if err == nil {
		t.Fatal("Run of a command that exits non-zero: want error, got nil")
	}
}

func TestExecRunner_RunsInGivenDirectory(t *testing.T) {
	dir := t.TempDir()
	wantDir, err := filepath.EvalSymlinks(dir)
	if err != nil {
		t.Fatalf("EvalSymlinks(%q): %v", dir, err)
	}

	out, err := DefaultRunner.Run(dir, "pwd")
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	gotDir, err := filepath.EvalSymlinks(strings.TrimSpace(out))
	if err != nil {
		t.Fatalf("EvalSymlinks(%q): %v", out, err)
	}
	if gotDir != wantDir {
		t.Errorf("pwd output resolved to %q, want %q", gotDir, wantDir)
	}
}
