package generate

import (
	"os"
	"testing"

	"endonend/protocol/history"
	"endonend/protocol/manifest"
	"endonend/protocol/signing"
)

func TestRotate_SwitchesKeyAndRecordsEntry(t *testing.T) {
	dir := testEnv(t)
	sourcePath := writeSource(t, dir, minimalSource())
	o := opts(dir, sourcePath)
	first, err := Run(o)
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	result, err := Rotate(RotateOptions{ManifestPath: o.ManifestOutPath, HistoryPath: o.HistoryOutPath, Now: fixedNow})
	if err != nil {
		t.Fatalf("Rotate returned error: %v", err)
	}
	if result.OldPublicKey != first.PublicKey {
		t.Errorf("OldPublicKey = %q, want %q", result.OldPublicKey, first.PublicKey)
	}
	if result.NewPublicKey == result.OldPublicKey {
		t.Error("NewPublicKey equals OldPublicKey, rotation did not change the key")
	}
	if result.Manifest.Identity.PublicKey != result.NewPublicKey {
		t.Error("manifest identity.publicKey was not updated to the new key")
	}

	// The rotation entry must verify against the OLD key.
	oldPub, err := signing.DecodePublicKey(result.OldPublicKey)
	if err != nil {
		t.Fatalf("DecodePublicKey: %v", err)
	}
	_, entries, err := loadPrevious(o.ManifestOutPath, o.HistoryOutPath)
	if err != nil {
		t.Fatalf("loadPrevious: %v", err)
	}
	last := entries[len(entries)-1]
	if last.Type != manifest.HistoryTypeKeyRotated {
		t.Fatalf("last history entry type = %q, want key_rotated", last.Type)
	}
	canonicalBytes, err := signing.CanonicalWithoutField(last, "signature")
	if err != nil {
		t.Fatalf("CanonicalWithoutField: %v", err)
	}
	ok, err := signing.VerifyCanonical(oldPub, canonicalBytes, last.Signature.Value)
	if err != nil {
		t.Fatalf("VerifyCanonical: %v", err)
	}
	if !ok {
		t.Error("key_rotated entry does not verify against the retired (old) key")
	}

	// The manifest itself must now verify against the NEW key.
	newPub, err := signing.DecodePublicKey(result.NewPublicKey)
	if err != nil {
		t.Fatalf("DecodePublicKey: %v", err)
	}
	manifestCanonical, err := signing.CanonicalWithoutField(result.Manifest, "signature")
	if err != nil {
		t.Fatalf("CanonicalWithoutField: %v", err)
	}
	ok, err = signing.VerifyCanonical(newPub, manifestCanonical, result.Manifest.Signature.Value)
	if err != nil {
		t.Fatalf("VerifyCanonical: %v", err)
	}
	if !ok {
		t.Error("manifest signature does not verify against the new key after rotation")
	}

	head, err := history.Hash(last)
	if err != nil {
		t.Fatalf("history.Hash: %v", err)
	}
	if result.Manifest.History.HeadHash != head {
		t.Errorf("manifest.history.headHash = %q, want %q", result.Manifest.History.HeadHash, head)
	}
}

func TestRotate_ErrorsWithoutExistingManifest(t *testing.T) {
	dir := testEnv(t)
	_, err := Rotate(RotateOptions{
		ManifestPath: dir + "/manifest.json",
		HistoryPath:  dir + "/history.json",
	})
	if err == nil {
		t.Error("Rotate without an existing manifest: want error, got nil")
	}
}

func TestRotate_ErrorsWithoutLocalKey(t *testing.T) {
	dir := testEnv(t)
	sourcePath := writeSource(t, dir, minimalSource())
	o := opts(dir, sourcePath)
	if _, err := Run(o); err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	keyPath, err := signing.KeyPath(minimalSource().Identity.URL)
	if err != nil {
		t.Fatalf("KeyPath: %v", err)
	}
	if err := os.Remove(keyPath); err != nil {
		t.Fatalf("remove key: %v", err)
	}

	if _, err := Rotate(RotateOptions{ManifestPath: o.ManifestOutPath, HistoryPath: o.HistoryOutPath}); err == nil {
		t.Error("Rotate with the local key already deleted: want error, got nil")
	}
}
