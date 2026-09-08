package generate

import (
	"crypto/ed25519"
	"fmt"
	"time"

	"endonend/cli/internal/history"
	"endonend/cli/internal/manifest"
	"endonend/cli/internal/signing"
)

type RotateOptions struct {
	ManifestPath string
	HistoryPath  string
	Now          func() time.Time
}

type RotateResult struct {
	OldPublicKey string
	NewPublicKey string
	Manifest     *manifest.Manifest
}

// Rotate signs a key_rotated history entry with the currently active
// private key, generates a new keypair, switches the manifest and local
// key store over to it, and re-signs the manifest with the new key. Per
// KB/0003's Key rotation section, the old entry is signed by the *old* key,
// proof the previous key-holder authorized this specific handoff.
func Rotate(opts RotateOptions) (*RotateResult, error) {
	now := time.Now().UTC
	if opts.Now != nil {
		now = opts.Now
	}

	prev, prevHistory, err := loadPrevious(opts.ManifestPath, opts.HistoryPath)
	if err != nil {
		return nil, err
	}
	if prev == nil {
		return nil, fmt.Errorf("no manifest found at %s; run generate before rotating a key", opts.ManifestPath)
	}

	keyPath, err := signing.KeyPath(prev.Identity.URL)
	if err != nil {
		return nil, err
	}
	if !signing.KeyExists(keyPath) {
		return nil, fmt.Errorf("no local private key found for %s; rotation requires the key currently in the manifest", prev.Identity.URL)
	}
	oldPriv, err := signing.LoadPrivateKey(keyPath)
	if err != nil {
		return nil, err
	}
	oldPub := oldPriv.Public().(ed25519.PublicKey)
	oldPubStr := signing.EncodePublicKey(oldPub)
	if oldPubStr != prev.Identity.PublicKey {
		return nil, fmt.Errorf("local private key at %s does not match identity.publicKey in %s; cannot safely rotate", keyPath, opts.ManifestPath)
	}

	newPub, newPriv, err := signing.GenerateKeypair()
	if err != nil {
		return nil, err
	}
	newPubStr := signing.EncodePublicKey(newPub)

	prevHash := ""
	if len(prevHistory) > 0 {
		prevHash, err = history.Hash(prevHistory[len(prevHistory)-1])
		if err != nil {
			return nil, err
		}
	}
	entry := manifest.HistoryEntry{
		Timestamp: now().Format(time.RFC3339),
		Type:      manifest.HistoryTypeKeyRotated,
		Data: map[string]any{
			"oldPublicKey": oldPubStr,
			"newPublicKey": newPubStr,
		},
		PreviousHash: prevHash,
	}
	canonicalBytes, err := signing.CanonicalWithoutField(entry, "signature")
	if err != nil {
		return nil, err
	}
	entry.Signature = manifest.Signature{
		Algorithm: manifest.SignatureAlgorithmEd25519,
		Value:     signing.SignCanonical(oldPriv, canonicalBytes),
	}

	allEntries := append(append([]manifest.HistoryEntry{}, prevHistory...), entry)
	headHash, err := history.Hash(entry)
	if err != nil {
		return nil, err
	}

	updated := *prev
	updated.Identity.PublicKey = newPubStr
	updated.History.HeadHash = headHash
	manifestCanonical, err := signing.CanonicalWithoutField(&updated, "signature")
	if err != nil {
		return nil, err
	}
	updated.Signature = manifest.Signature{
		Algorithm: manifest.SignatureAlgorithmEd25519,
		Value:     signing.SignCanonical(newPriv, manifestCanonical),
	}

	if err := signing.SavePrivateKey(keyPath, newPriv); err != nil {
		return nil, err
	}
	if err := writeJSON(opts.ManifestPath, &updated); err != nil {
		return nil, err
	}
	if err := writeJSON(opts.HistoryPath, allEntries); err != nil {
		return nil, err
	}

	return &RotateResult{OldPublicKey: oldPubStr, NewPublicKey: newPubStr, Manifest: &updated}, nil
}
