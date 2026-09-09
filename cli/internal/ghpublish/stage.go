package ghpublish

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"endonend/protocol/manifest"
)

// StageOptions configures where Stage looks for already-downloaded local
// assets and where it assembles the publishable site tree.
type StageOptions struct {
	ManifestPath string // signed manifest.json to publish
	HistoryPath  string // paired history.json to publish
	AssetsDir    string // local directory holding downloaded album assets, per <AssetsDir>/<albumId>/<filename>
	OutDir       string // where the assembled, publish-ready tree is written
}

// Stage assembles a directory tree matching exactly what identity.url must
// serve: manifest.json and history.json under .well-known/endonend/, and
// every asset the manifest actually references under albums/<albumId>/,
// copied from AssetsDir. It only ever copies files the manifest itself
// declares as living under identity.url (front/back/insert images, track
// files, the ZIP download), never anything else in the working directory,
// so unrelated local files can't end up published by accident. Assets that
// live elsewhere (merch links, purchase links) are left as the external
// URLs they already are.
//
// This assumes the "<identity.url>/albums/<albumId>/<filename>" layout
// KB/0005-bandcamp-import.md's importer already writes, mirrored locally
// at "<AssetsDir>/<albumId>/<filename>". A hand-built manifest that uses a
// different URL layout needs its assets already staged some other way.
func Stage(m *manifest.Manifest, opts StageOptions) error {
	wellKnown := filepath.Join(opts.OutDir, ".well-known", "endonend")
	if err := os.MkdirAll(wellKnown, 0o755); err != nil {
		return fmt.Errorf("create .well-known/endonend: %w", err)
	}
	if err := copyFile(opts.ManifestPath, filepath.Join(wellKnown, "manifest.json")); err != nil {
		return fmt.Errorf("stage manifest.json: %w", err)
	}
	if err := copyFile(opts.HistoryPath, filepath.Join(wellKnown, "history.json")); err != nil {
		return fmt.Errorf("stage history.json: %w", err)
	}

	s := &stager{base: strings.TrimRight(m.Identity.URL, "/") + "/", opts: opts, seen: map[string]bool{}}
	for _, album := range m.Catalog {
		if err := s.stageAsset(album.Images.Front); err != nil {
			return err
		}
		if err := s.stageAsset(album.Images.Back); err != nil {
			return err
		}
		for _, insert := range album.Images.Insert {
			if err := s.stageAsset(insert); err != nil {
				return err
			}
		}
		if err := s.stageAsset(album.DownloadZip); err != nil {
			return err
		}
		for _, t := range album.Tracks {
			if err := s.stageAsset(t.File); err != nil {
				return err
			}
		}
	}
	return nil
}

type stager struct {
	base string
	opts StageOptions
	seen map[string]bool
}

func (s *stager) stageAsset(assetURL string) error {
	if assetURL == "" || s.seen[assetURL] {
		return nil
	}
	rel, ok := strings.CutPrefix(assetURL, s.base)
	if !ok {
		// Not one of this identity's own hosted URLs (for example, a URL
		// on some other host); nothing of this identity's own to stage.
		return nil
	}
	s.seen[assetURL] = true

	localRel := strings.TrimPrefix(rel, "albums/")
	src := filepath.Join(s.opts.AssetsDir, filepath.FromSlash(localRel))
	dst := filepath.Join(s.opts.OutDir, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return fmt.Errorf("create directory for %s: %w", rel, err)
	}
	if err := copyFile(src, dst); err != nil {
		return fmt.Errorf("stage %s (expected local file at %s): %w", rel, src, err)
	}
	return nil
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer func() { _ = in.Close() }()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer func() { _ = out.Close() }()

	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return out.Close()
}
