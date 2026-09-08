// Package canonical wraps RFC 8785 (JSON Canonicalization Scheme) so every
// signer and verifier in this tool produces the exact same bytes, per
// KB/0003-manifest.md's signature section.
package canonical

import (
	"encoding/json"
	"fmt"

	"github.com/gowebpki/jcs"
)

// Marshal encodes v as JSON, then canonicalizes it per RFC 8785.
func Marshal(v any) ([]byte, error) {
	raw, err := json.Marshal(v)
	if err != nil {
		return nil, fmt.Errorf("marshal before canonicalization: %w", err)
	}
	out, err := jcs.Transform(raw)
	if err != nil {
		return nil, fmt.Errorf("canonicalize: %w", err)
	}
	return out, nil
}

// ToMap round-trips v through JSON into a generic map, so callers can drop
// or inspect individual top-level fields before canonicalizing.
func ToMap(v any) (map[string]any, error) {
	raw, err := json.Marshal(v)
	if err != nil {
		return nil, fmt.Errorf("marshal to map: %w", err)
	}
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		return nil, fmt.Errorf("unmarshal to map: %w", err)
	}
	return m, nil
}
