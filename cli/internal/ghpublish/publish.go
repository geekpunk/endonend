package ghpublish

import (
	"fmt"
	"strings"
)

// EnsureRepo creates the GitHub repo for target if it doesn't already
// exist, so re-running "publish github" is safe.
func EnsureRepo(r Runner, target *PagesTarget, private bool) (created bool, err error) {
	if _, err := r.Run("", "gh", "repo", "view", target.FullName()); err == nil {
		return false, nil
	}

	visibility := "--public"
	if private {
		visibility = "--private"
	}
	if _, err := r.Run("", "gh", "repo", "create", target.FullName(), visibility); err != nil {
		return false, fmt.Errorf("create GitHub repo %s: %w", target.FullName(), err)
	}
	return true, nil
}

// PushDir initializes dir as a git repo if it isn't already one, points it
// at target's remote, commits anything staged there, and pushes branch.
// dir is expected to persist between runs (Stage always writes into the
// same OutDir), so after the first run this is an ordinary fast-forward
// push, never a forced one: a real, unexpected divergence is surfaced as
// an error for the artist to resolve, not silently overwritten.
func PushDir(r Runner, dir string, target *PagesTarget, branch string) error {
	if _, err := r.Run(dir, "git", "rev-parse", "--is-inside-work-tree"); err != nil {
		if _, err := r.Run(dir, "git", "init", "-b", branch); err != nil {
			return fmt.Errorf("git init: %w", err)
		}
	}

	remote := fmt.Sprintf("https://github.com/%s.git", target.FullName())
	// Removing before adding is how this stays idempotent across repeated
	// publish runs regardless of whether a remote already existed (from an
	// earlier run) or not; "no such remote" on a first run is expected and
	// ignored, since the following "remote add" is what actually matters.
	_, _ = r.Run(dir, "git", "remote", "remove", "origin")
	if _, err := r.Run(dir, "git", "remote", "add", "origin", remote); err != nil {
		return fmt.Errorf("git remote add: %w", err)
	}

	if _, err := r.Run(dir, "git", "add", "-A"); err != nil {
		return fmt.Errorf("git add: %w", err)
	}
	status, err := r.Run(dir, "git", "status", "--porcelain")
	if err != nil {
		return fmt.Errorf("git status: %w", err)
	}
	if strings.TrimSpace(status) != "" {
		if _, err := r.Run(dir, "git", "commit", "-m", "Publish catalog update"); err != nil {
			return fmt.Errorf("git commit: %w", err)
		}
	}

	if _, err := r.Run(dir, "git", "push", "-u", "origin", branch); err != nil {
		return fmt.Errorf("git push: %w", err)
	}
	return nil
}

// EnablePages turns on GitHub Pages for target's repo, serving branch from
// the repository root. Tolerates Pages already being enabled.
func EnablePages(r Runner, target *PagesTarget, branch string) error {
	_, err := r.Run("", "gh", "api",
		fmt.Sprintf("repos/%s/pages", target.FullName()),
		"-X", "POST",
		"-f", "build_type=legacy",
		"-f", "source[branch]="+branch,
		"-f", "source[path]=/",
	)
	if err != nil && !strings.Contains(err.Error(), "already exists") {
		return fmt.Errorf("enable GitHub Pages: %w", err)
	}
	return nil
}
