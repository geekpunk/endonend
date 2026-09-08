package history

import (
	"testing"

	"endonend/protocol/manifest"
)

func entry(previousHash, sigValue string) manifest.HistoryEntry {
	return manifest.HistoryEntry{
		Timestamp:    "2024-01-01T00:00:00Z",
		Type:         manifest.HistoryTypeReleaseAdded,
		Data:         map[string]any{"albumId": "a"},
		PreviousHash: previousHash,
		Signature:    manifest.Signature{Algorithm: "ed25519", Value: sigValue},
	}
}

func TestHash_IsDeterministic(t *testing.T) {
	e := entry("", "sig1")
	h1, err := Hash(e)
	if err != nil {
		t.Fatalf("Hash returned error: %v", err)
	}
	h2, err := Hash(e)
	if err != nil {
		t.Fatalf("Hash returned error: %v", err)
	}
	if h1 != h2 {
		t.Errorf("Hash() not deterministic: %q != %q", h1, h2)
	}
	if len(h1) != len(HashPrefix)+64 {
		t.Errorf("Hash() = %q, want %s + 64 hex chars", h1, HashPrefix)
	}
}

func TestHash_DiffersOnContentChange(t *testing.T) {
	h1, err := Hash(entry("", "sig1"))
	if err != nil {
		t.Fatalf("Hash returned error: %v", err)
	}
	h2, err := Hash(entry("", "sig2"))
	if err != nil {
		t.Fatalf("Hash returned error: %v", err)
	}
	if h1 == h2 {
		t.Error("Hash() produced the same hash for entries with different signatures")
	}
}

func TestVerifyChain_EmptyIsValid(t *testing.T) {
	head, err := VerifyChain(nil)
	if err != nil {
		t.Fatalf("VerifyChain(nil) returned error: %v", err)
	}
	if head != "" {
		t.Errorf("VerifyChain(nil) head = %q, want empty", head)
	}
}

func TestVerifyChain_ValidChain(t *testing.T) {
	e0 := entry("", "sig0")
	h0, err := Hash(e0)
	if err != nil {
		t.Fatalf("Hash returned error: %v", err)
	}
	e1 := entry(h0, "sig1")
	h1, err := Hash(e1)
	if err != nil {
		t.Fatalf("Hash returned error: %v", err)
	}

	head, err := VerifyChain([]manifest.HistoryEntry{e0, e1})
	if err != nil {
		t.Fatalf("VerifyChain returned error: %v", err)
	}
	if head != h1 {
		t.Errorf("VerifyChain() head = %q, want %q", head, h1)
	}
}

func TestVerifyChain_RejectsNonEmptyGenesisPreviousHash(t *testing.T) {
	_, err := VerifyChain([]manifest.HistoryEntry{entry("sha256:deadbeef", "sig0")})
	if err == nil {
		t.Error("VerifyChain with a non-empty genesis previousHash: want error, got nil")
	}
}

func TestVerifyChain_RejectsBrokenLink(t *testing.T) {
	e0 := entry("", "sig0")
	e1 := entry("sha256:wronghash", "sig1")
	_, err := VerifyChain([]manifest.HistoryEntry{e0, e1})
	if err == nil {
		t.Error("VerifyChain with a mismatched previousHash: want error, got nil")
	}
}
