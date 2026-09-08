// Package generate implements the "Create or update your manifest" logic
// from KB/0004-endonend-artist-cli.md: read endonend.source.json, diff it
// against the previously generated manifest.json (if any) to compute
// history entries automatically, then write a freshly signed manifest.json
// and updated history.json.
package generate

import (
	"crypto/ed25519"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"endonend/protocol/canonical"
	"endonend/protocol/history"
	"endonend/protocol/manifest"
	"endonend/protocol/signing"
)

type Options struct {
	SourcePath      string
	ManifestOutPath string
	HistoryOutPath  string
	// Now is overridable for tests; defaults to time.Now when zero.
	Now func() time.Time
}

type Result struct {
	Manifest       *manifest.Manifest
	History        []manifest.HistoryEntry
	NewEntries     []manifest.HistoryEntry
	KeyGenerated   bool
	PrivateKeyPath string
	PublicKey      string
}

func (o *Options) now() time.Time {
	if o.Now != nil {
		return o.Now()
	}
	return time.Now().UTC()
}

// Run executes the full generate flow and writes the resulting manifest
// and history files to disk.
func Run(opts Options) (*Result, error) {
	raw, err := os.ReadFile(opts.SourcePath)
	if err != nil {
		return nil, fmt.Errorf("read source file %s: %w", opts.SourcePath, err)
	}
	var src manifest.Source
	if err := json.Unmarshal(raw, &src); err != nil {
		return nil, fmt.Errorf("parse source file %s: %w", opts.SourcePath, err)
	}
	if err := validateSource(&src); err != nil {
		return nil, err
	}

	keyPath, err := signing.KeyPath(src.Identity.URL)
	if err != nil {
		return nil, err
	}
	var priv ed25519.PrivateKey
	keyGenerated := false
	if signing.KeyExists(keyPath) {
		priv, err = signing.LoadPrivateKey(keyPath)
		if err != nil {
			return nil, err
		}
	} else {
		var pub ed25519.PublicKey
		pub, priv, err = signing.GenerateKeypair()
		if err != nil {
			return nil, err
		}
		_ = pub
		if err := signing.SavePrivateKey(keyPath, priv); err != nil {
			return nil, err
		}
		keyGenerated = true
	}
	pub := priv.Public().(ed25519.PublicKey)
	pubStr := signing.EncodePublicKey(pub)

	prevManifest, prevHistory, err := loadPrevious(opts.ManifestOutPath, opts.HistoryOutPath)
	if err != nil {
		return nil, err
	}

	newEntries, catalog, err := diff(prevManifest, &src, opts.now())
	if err != nil {
		return nil, err
	}

	allEntries := append(append([]manifest.HistoryEntry{}, prevHistory...), newEntries...)
	for i := range allEntries {
		if newlySigned := i >= len(prevHistory); newlySigned {
			prev := ""
			if i > 0 {
				prev, err = history.Hash(allEntries[i-1])
				if err != nil {
					return nil, err
				}
			}
			allEntries[i].PreviousHash = prev
			canonicalBytes, err := signing.CanonicalWithoutField(allEntries[i], "signature")
			if err != nil {
				return nil, err
			}
			allEntries[i].Signature = manifest.Signature{
				Algorithm: manifest.SignatureAlgorithmEd25519,
				Value:     signing.SignCanonical(priv, canonicalBytes),
			}
		}
	}

	headHash := ""
	if len(allEntries) > 0 {
		headHash, err = history.Hash(allEntries[len(allEntries)-1])
		if err != nil {
			return nil, err
		}
	}

	historyURL := src.Identity.URL + "/.well-known/endonend/history.json"
	if src.History != nil && src.History.URL != "" {
		historyURL = src.History.URL
	}

	m := &manifest.Manifest{
		ManifestVersion: src.ManifestVersion,
		Identity: manifest.Identity{
			Type:         src.Identity.Type,
			Name:         src.Identity.Name,
			URL:          src.Identity.URL,
			PublicKey:    pubStr,
			ContactEmail: src.Identity.ContactEmail,
		},
		Refresh:       src.Refresh,
		Label:         src.Label,
		Contributions: src.Contributions,
		Beacon:        src.Beacon,
		Merch:         src.Merch,
		History:       manifest.HistoryRef{URL: historyURL, HeadHash: headHash},
		Presentation:  src.Presentation,
		Catalog:       catalog,
	}

	canonicalBytes, err := signing.CanonicalWithoutField(m, "signature")
	if err != nil {
		return nil, err
	}
	m.Signature = manifest.Signature{
		Algorithm: manifest.SignatureAlgorithmEd25519,
		Value:     signing.SignCanonical(priv, canonicalBytes),
	}

	if err := writeJSON(opts.ManifestOutPath, m); err != nil {
		return nil, err
	}
	if err := writeJSON(opts.HistoryOutPath, allEntries); err != nil {
		return nil, err
	}

	return &Result{
		Manifest:       m,
		History:        allEntries,
		NewEntries:     newEntries,
		KeyGenerated:   keyGenerated,
		PrivateKeyPath: keyPath,
		PublicKey:      pubStr,
	}, nil
}

func validateSource(src *manifest.Source) error {
	switch {
	case src.ManifestVersion == "":
		return fmt.Errorf("source is missing manifestVersion")
	case src.Identity.Type != "artist" && src.Identity.Type != "label":
		return fmt.Errorf("identity.type must be \"artist\" or \"label\", got %q", src.Identity.Type)
	case src.Identity.Name == "":
		return fmt.Errorf("identity.name is required")
	case src.Identity.URL == "":
		return fmt.Errorf("identity.url is required")
	case src.Identity.ContactEmail == "":
		return fmt.Errorf("identity.contactEmail is required")
	case src.Refresh.TTLSeconds <= 0:
		return fmt.Errorf("refresh.ttlSeconds must be a positive integer")
	}
	if src.Identity.Type == "artist" && len(src.Catalog) == 0 {
		return fmt.Errorf("artist manifests require a non-empty catalog")
	}
	if src.Identity.Type == "label" && len(src.Catalog) > 0 {
		return fmt.Errorf("label manifests must not declare a catalog")
	}
	seenAlbum := map[string]bool{}
	seenTrack := map[string]bool{}
	for _, a := range src.Catalog {
		if a.AlbumID == "" {
			return fmt.Errorf("every album needs an albumId")
		}
		if seenAlbum[a.AlbumID] {
			return fmt.Errorf("duplicate albumId %q", a.AlbumID)
		}
		seenAlbum[a.AlbumID] = true
		if a.Images.Front == "" || a.Images.Back == "" {
			return fmt.Errorf("album %q: images.front and images.back are required", a.AlbumID)
		}
		for _, t := range a.Tracks {
			if t.TrackID == "" {
				return fmt.Errorf("album %q: every track needs a trackId", a.AlbumID)
			}
			if seenTrack[t.TrackID] {
				return fmt.Errorf("duplicate trackId %q", t.TrackID)
			}
			seenTrack[t.TrackID] = true
		}
	}
	return nil
}

func loadPrevious(manifestPath, historyPath string) (*manifest.Manifest, []manifest.HistoryEntry, error) {
	raw, err := os.ReadFile(manifestPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil, nil
		}
		return nil, nil, fmt.Errorf("read existing manifest %s: %w", manifestPath, err)
	}
	var prev manifest.Manifest
	if err := json.Unmarshal(raw, &prev); err != nil {
		return nil, nil, fmt.Errorf("parse existing manifest %s: %w", manifestPath, err)
	}
	histRaw, err := os.ReadFile(historyPath)
	if err != nil {
		return nil, nil, fmt.Errorf("a manifest already exists at %s but its history log %s could not be read (%w); cannot safely compute a diff", manifestPath, historyPath, err)
	}
	var entries []manifest.HistoryEntry
	if err := json.Unmarshal(histRaw, &entries); err != nil {
		return nil, nil, fmt.Errorf("parse existing history %s: %w", historyPath, err)
	}
	return &prev, entries, nil
}

func writeJSON(path string, v any) error {
	out, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Errorf("encode %s: %w", path, err)
	}
	out = append(out, '\n')
	if err := os.WriteFile(path, out, 0o644); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	return nil
}

func albumFingerprint(a manifest.Album) (string, error) {
	a.AlbumVersion = 0
	generic, err := canonical.ToMap(a)
	if err != nil {
		return "", err
	}
	delete(generic, "albumVersion")
	b, err := canonical.Marshal(generic)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func newEntry(t string, data map[string]any, now time.Time) manifest.HistoryEntry {
	return manifest.HistoryEntry{
		Timestamp: now.Format(time.RFC3339),
		Type:      t,
		Data:      data,
	}
}

// diff compares the previous manifest (nil on first generate) against the
// new source, returning the history entries the change implies and the
// catalog with albumVersion values the tool itself computes.
func diff(prev *manifest.Manifest, src *manifest.Source, now time.Time) ([]manifest.HistoryEntry, []manifest.Album, error) {
	var entries []manifest.HistoryEntry

	if prev != nil {
		identityChanges := map[string]any{}
		if prev.Identity.Name != src.Identity.Name {
			identityChanges["name"] = map[string]any{"old": prev.Identity.Name, "new": src.Identity.Name}
		}
		if prev.Identity.ContactEmail != src.Identity.ContactEmail {
			identityChanges["contactEmail"] = map[string]any{"old": prev.Identity.ContactEmail, "new": src.Identity.ContactEmail}
		}
		if len(identityChanges) > 0 {
			entries = append(entries, newEntry(manifest.HistoryTypeIdentityUpdated, identityChanges, now))
		}

		oldLabel, err := canonical.ToMap(prev.Label)
		if err != nil {
			return nil, nil, err
		}
		newLabel, err := canonical.ToMap(src.Label)
		if err != nil {
			return nil, nil, err
		}
		oldLabelJSON, _ := json.Marshal(oldLabel)
		newLabelJSON, _ := json.Marshal(newLabel)
		if string(oldLabelJSON) != string(newLabelJSON) {
			entries = append(entries, newEntry(manifest.HistoryTypeSplitChanged, map[string]any{
				"old": oldLabel,
				"new": newLabel,
			}, now))
		}
	}

	prevAlbums := map[string]manifest.Album{}
	if prev != nil {
		for _, a := range prev.Catalog {
			prevAlbums[a.AlbumID] = a
		}
	}

	catalog := make([]manifest.Album, len(src.Catalog))
	seen := map[string]bool{}
	for i, a := range src.Catalog {
		seen[a.AlbumID] = true
		old, existed := prevAlbums[a.AlbumID]
		if !existed {
			a.AlbumVersion = 1
			entries = append(entries, newEntry(manifest.HistoryTypeReleaseAdded, map[string]any{
				"albumId": a.AlbumID,
			}, now))
			catalog[i] = a
			continue
		}
		oldFp, err := albumFingerprint(old)
		if err != nil {
			return nil, nil, err
		}
		newFp, err := albumFingerprint(a)
		if err != nil {
			return nil, nil, err
		}
		if oldFp == newFp {
			a.AlbumVersion = old.AlbumVersion
		} else {
			a.AlbumVersion = old.AlbumVersion + 1
			entries = append(entries, newEntry(manifest.HistoryTypeReleaseUpdated, map[string]any{
				"albumId":         a.AlbumID,
				"oldAlbumVersion": old.AlbumVersion,
				"newAlbumVersion": a.AlbumVersion,
			}, now))
		}
		catalog[i] = a
	}
	for id, old := range prevAlbums {
		if !seen[id] {
			entries = append(entries, newEntry(manifest.HistoryTypeReleaseRemoved, map[string]any{
				"albumId":      id,
				"albumVersion": old.AlbumVersion,
			}, now))
		}
	}

	return entries, catalog, nil
}
