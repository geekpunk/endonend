package main

import (
	"path/filepath"
	"testing"
)

// run's zero-argument branch launches the interactive menu, which blocks
// reading os.Stdin; it's exercised through the menu helpers directly
// (see menu_test.go) rather than through run() itself.

func TestRun_HelpVariants(t *testing.T) {
	for _, args := range [][]string{{"help"}, {"--help"}, {"-h"}} {
		t.Run(args[0], func(t *testing.T) {
			code := captureExitCode(t, func() int { return run(args) })
			if code != 0 {
				t.Errorf("run(%v) = %d, want 0", args, code)
			}
		})
	}
}

func TestRun_UnknownCommandExitsOne(t *testing.T) {
	code := captureExitCode(t, func() int { return run([]string{"bogus"}) })
	if code != 1 {
		t.Errorf("run([\"bogus\"]) = %d, want 1", code)
	}
}

func TestRun_DispatchesToGenerate(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	code := captureExitCode(t, func() int {
		return run([]string{"generate", "--source", filepath.Join(t.TempDir(), "nope.json")})
	})
	if code != 1 {
		t.Errorf("run([\"generate\", ...missing source]) = %d, want 1", code)
	}
}

func TestRun_DispatchesToValidate(t *testing.T) {
	code := captureExitCode(t, func() int {
		return run([]string{"validate", filepath.Join(t.TempDir(), "nope.json")})
	})
	if code != 1 {
		t.Errorf("run([\"validate\", ...missing file]) = %d, want 1", code)
	}
}

func TestRun_DispatchesToKeygen(t *testing.T) {
	code := captureExitCode(t, func() int { return run([]string{"keygen"}) })
	if code != 1 {
		t.Errorf("run([\"keygen\"]) = %d, want 1 without --url or --rotate", code)
	}
}
