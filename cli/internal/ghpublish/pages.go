// Package ghpublish implements "publish github": assembling a signed
// manifest and its assets into a GitHub Pages-ready site tree and pushing
// it to a new or existing GitHub repo, per KB/0004-endonend-artist-cli.md.
package ghpublish

import (
	"fmt"
	"net/url"
	"strings"
)

// PagesTarget is the GitHub repo an identity.url implies, per GitHub
// Pages' own naming rules: a user/org page must live in a repo named
// "<owner>.github.io" and serves from its root, while a project page
// lives in a repo named after the URL's first path segment and serves at
// "/<repo>/".
type PagesTarget struct {
	Owner      string
	Repo       string
	IsUserPage bool
}

// ParsePagesURL parses identityURL into the GitHub repo that must serve it.
// It returns an error if identityURL isn't a github.io URL: this package
// only ever publishes to GitHub Pages, per its own name.
func ParsePagesURL(identityURL string) (*PagesTarget, error) {
	u, err := url.Parse(identityURL)
	if err != nil {
		return nil, fmt.Errorf("parse identity.url %q: %w", identityURL, err)
	}
	host := strings.ToLower(u.Hostname())
	owner, ok := strings.CutSuffix(host, ".github.io")
	if !ok || owner == "" {
		return nil, fmt.Errorf("identity.url %q is not a github.io URL; \"publish github\" only publishes to GitHub Pages", identityURL)
	}

	path := strings.Trim(u.Path, "/")
	if path == "" {
		return &PagesTarget{Owner: owner, Repo: owner + ".github.io", IsUserPage: true}, nil
	}
	// A GitHub Pages project URL's first path segment is always the repo
	// name; identity.url may have more path beyond that (KB/0003-manifest.md's
	// own path-based identity examples), but that's namespacing within the
	// one repo, not a second repo.
	repo := strings.SplitN(path, "/", 2)[0]
	return &PagesTarget{Owner: owner, Repo: repo, IsUserPage: false}, nil
}

// URL is the GitHub Pages URL this target ultimately serves at.
func (t *PagesTarget) URL() string {
	if t.IsUserPage {
		return fmt.Sprintf("https://%s.github.io", t.Owner)
	}
	return fmt.Sprintf("https://%s.github.io/%s", t.Owner, t.Repo)
}

// FullName is the "owner/repo" form gh and git both expect.
func (t *PagesTarget) FullName() string {
	return t.Owner + "/" + t.Repo
}
