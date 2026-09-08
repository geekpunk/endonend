// Package signing implements ed25519 key generation, storage, and the
// sign/verify operations KB/0003-manifest.md's signature section and
// KB/0004-endonend-artist-cli.md's key management section describe.
package signing

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"endonend/cli/internal/canonical"
)

const KeyPrefix = "ed25519:"

// GenerateKeypair produces a fresh ed25519 keypair.
func GenerateKeypair() (ed25519.PublicKey, ed25519.PrivateKey, error) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, nil, fmt.Errorf("generate keypair: %w", err)
	}
	return pub, priv, nil
}

// EncodePublicKey renders a public key as manifest's identity.publicKey
// shape, e.g. "ed25519:AbCdEf...".
func EncodePublicKey(pub ed25519.PublicKey) string {
	return KeyPrefix + base64.StdEncoding.EncodeToString(pub)
}

// DecodePublicKey parses identity.publicKey back into raw key bytes.
func DecodePublicKey(s string) (ed25519.PublicKey, error) {
	if !strings.HasPrefix(s, KeyPrefix) {
		return nil, fmt.Errorf("unsupported public key format: %q", s)
	}
	raw, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(s, KeyPrefix))
	if err != nil {
		return nil, fmt.Errorf("decode public key: %w", err)
	}
	if len(raw) != ed25519.PublicKeySize {
		return nil, fmt.Errorf("public key has wrong length: got %d bytes, want %d", len(raw), ed25519.PublicKeySize)
	}
	return ed25519.PublicKey(raw), nil
}

var slugUnsafe = regexp.MustCompile(`[^a-zA-Z0-9.-]+`)

// Slug turns an identity.url into a filesystem-safe name for its key file,
// e.g. "https://someartist.github.io/bandname" -> "someartist.github.io_bandname".
func Slug(identityURL string) string {
	s := identityURL
	s = strings.TrimPrefix(s, "https://")
	s = strings.TrimPrefix(s, "http://")
	s = slugUnsafe.ReplaceAllString(s, "_")
	s = strings.Trim(s, "_")
	if s == "" {
		s = "identity"
	}
	return s
}

// KeyDir returns ~/.endonend/keys, creating it (mode 0700) if needed.
func KeyDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("locate home directory: %w", err)
	}
	dir := filepath.Join(home, ".endonend", "keys")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", fmt.Errorf("create key directory: %w", err)
	}
	gitignore := filepath.Join(filepath.Join(home, ".endonend"), ".gitignore")
	if _, err := os.Stat(gitignore); os.IsNotExist(err) {
		_ = os.WriteFile(gitignore, []byte("keys/\n"), 0o644)
	}
	return dir, nil
}

// KeyPath returns the local private key path for a given identity.url.
func KeyPath(identityURL string) (string, error) {
	dir, err := KeyDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, Slug(identityURL)+".key"), nil
}

// SavePrivateKey writes priv to path with mode 0600.
func SavePrivateKey(path string, priv ed25519.PrivateKey) error {
	enc := base64.StdEncoding.EncodeToString(priv)
	if err := os.WriteFile(path, []byte(enc+"\n"), 0o600); err != nil {
		return fmt.Errorf("write private key: %w", err)
	}
	return nil
}

// LoadPrivateKey reads a private key previously written by SavePrivateKey.
func LoadPrivateKey(path string) (ed25519.PrivateKey, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	decoded, err := base64.StdEncoding.DecodeString(strings.TrimSpace(string(raw)))
	if err != nil {
		return nil, fmt.Errorf("decode private key at %s: %w", path, err)
	}
	if len(decoded) != ed25519.PrivateKeySize {
		return nil, fmt.Errorf("private key at %s has wrong length: got %d bytes, want %d", path, len(decoded), ed25519.PrivateKeySize)
	}
	return ed25519.PrivateKey(decoded), nil
}

// KeyExists reports whether a private key file already exists for path.
func KeyExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// SignCanonical signs already-canonicalized bytes and returns a base64
// signature value.
func SignCanonical(priv ed25519.PrivateKey, canonicalBytes []byte) string {
	sig := ed25519.Sign(priv, canonicalBytes)
	return base64.StdEncoding.EncodeToString(sig)
}

// VerifyCanonical verifies a base64 signature value against already-
// canonicalized bytes.
func VerifyCanonical(pub ed25519.PublicKey, canonicalBytes []byte, sigValue string) (bool, error) {
	sig, err := base64.StdEncoding.DecodeString(sigValue)
	if err != nil {
		return false, fmt.Errorf("decode signature value: %w", err)
	}
	return ed25519.Verify(pub, canonicalBytes, sig), nil
}

// CanonicalWithoutField marshals v to JSON, removes the named top-level
// field (typically "signature"), and returns the RFC 8785 canonical bytes
// of what remains, per 0003: "over the canonicalized manifest with this
// signature object excluded."
func CanonicalWithoutField(v any, field string) ([]byte, error) {
	generic, err := canonical.ToMap(v)
	if err != nil {
		return nil, err
	}
	delete(generic, field)
	return canonical.Marshal(generic)
}
