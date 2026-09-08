// Package history implements the append-only hash chain described in
// KB/0003-manifest.md's "History log format" section: computing an entry's
// hash, chaining new entries onto a prior log, and verifying the chain is
// unbroken.
package history

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"

	"endonend/protocol/canonical"
	"endonend/protocol/manifest"
)

const HashPrefix = "sha256:"

// Hash computes the chain hash of a single entry: sha256 over its RFC 8785
// canonical JSON (including its own signature), hex-encoded and prefixed.
func Hash(entry manifest.HistoryEntry) (string, error) {
	canonicalBytes, err := canonical.Marshal(entry)
	if err != nil {
		return "", fmt.Errorf("canonicalize history entry: %w", err)
	}
	sum := sha256.Sum256(canonicalBytes)
	return HashPrefix + hex.EncodeToString(sum[:]), nil
}

// VerifyChain walks entries in order, checking that each one's
// previousHash matches the hash of the entry before it (empty/absent on
// the genesis entry), and returns the head hash of the last entry.
// Structural only: it does not check signatures, see internal/validate.
func VerifyChain(entries []manifest.HistoryEntry) (headHash string, err error) {
	prev := ""
	for i, e := range entries {
		wantPrev := prev
		if i == 0 {
			if e.PreviousHash != "" {
				return "", fmt.Errorf("entry %d: genesis entry must have an empty previousHash, got %q", i, e.PreviousHash)
			}
		} else if e.PreviousHash != wantPrev {
			return "", fmt.Errorf("entry %d: previousHash %q does not match hash of entry %d (%q)", i, e.PreviousHash, i-1, wantPrev)
		}
		h, err := Hash(e)
		if err != nil {
			return "", fmt.Errorf("entry %d: %w", i, err)
		}
		prev = h
	}
	return prev, nil
}
