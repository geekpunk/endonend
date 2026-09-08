package signing

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGenerateKeypair_ProducesUsableKeys(t *testing.T) {
	pub, priv, err := GenerateKeypair()
	if err != nil {
		t.Fatalf("GenerateKeypair returned error: %v", err)
	}
	sig := SignCanonical(priv, []byte("hello"))
	ok, err := VerifyCanonical(pub, []byte("hello"), sig)
	if err != nil {
		t.Fatalf("VerifyCanonical returned error: %v", err)
	}
	if !ok {
		t.Error("VerifyCanonical() = false, want true for a freshly generated keypair")
	}
}

func TestEncodeDecodePublicKey_RoundTrip(t *testing.T) {
	pub, _, err := GenerateKeypair()
	if err != nil {
		t.Fatalf("GenerateKeypair returned error: %v", err)
	}
	encoded := EncodePublicKey(pub)
	if encoded[:len(KeyPrefix)] != KeyPrefix {
		t.Errorf("EncodePublicKey() = %q, want prefix %q", encoded, KeyPrefix)
	}
	decoded, err := DecodePublicKey(encoded)
	if err != nil {
		t.Fatalf("DecodePublicKey returned error: %v", err)
	}
	if string(decoded) != string(pub) {
		t.Error("DecodePublicKey(EncodePublicKey(pub)) != pub")
	}
}

func TestDecodePublicKey_RejectsBadInput(t *testing.T) {
	cases := []string{
		"",
		"rsa:AbCd",
		"ed25519:not-base64!!!",
		"ed25519:" + "AA==", // valid base64, wrong length
	}
	for _, c := range cases {
		if _, err := DecodePublicKey(c); err == nil {
			t.Errorf("DecodePublicKey(%q): want error, got nil", c)
		}
	}
}

func TestVerifyCanonical_RejectsTamperedData(t *testing.T) {
	pub, priv, err := GenerateKeypair()
	if err != nil {
		t.Fatalf("GenerateKeypair returned error: %v", err)
	}
	sig := SignCanonical(priv, []byte("original"))
	ok, err := VerifyCanonical(pub, []byte("tampered"), sig)
	if err != nil {
		t.Fatalf("VerifyCanonical returned error: %v", err)
	}
	if ok {
		t.Error("VerifyCanonical() = true for tampered data, want false")
	}
}

func TestVerifyCanonical_RejectsBadSignatureEncoding(t *testing.T) {
	pub, _, err := GenerateKeypair()
	if err != nil {
		t.Fatalf("GenerateKeypair returned error: %v", err)
	}
	if _, err := VerifyCanonical(pub, []byte("x"), "not-base64!!!"); err == nil {
		t.Error("VerifyCanonical with malformed signature value: want error, got nil")
	}
}

func TestSlug_ProducesFilesystemSafeNames(t *testing.T) {
	cases := map[string]string{
		"https://ligatures.example":             "ligatures.example",
		"http://ligatures.example":              "ligatures.example",
		"https://someartist.github.io/bandname": "someartist.github.io_bandname",
		"https://x.example/a/b?c=d":             "x.example_a_b_c_d",
	}
	for url, want := range cases {
		if got := Slug(url); got != want {
			t.Errorf("Slug(%q) = %q, want %q", url, got, want)
		}
	}
}

func TestSaveAndLoadPrivateKey_RoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.key")

	_, priv, err := GenerateKeypair()
	if err != nil {
		t.Fatalf("GenerateKeypair returned error: %v", err)
	}
	if err := SavePrivateKey(path, priv); err != nil {
		t.Fatalf("SavePrivateKey returned error: %v", err)
	}
	if !KeyExists(path) {
		t.Fatal("KeyExists() = false right after SavePrivateKey")
	}
	loaded, err := LoadPrivateKey(path)
	if err != nil {
		t.Fatalf("LoadPrivateKey returned error: %v", err)
	}
	if string(loaded) != string(priv) {
		t.Error("LoadPrivateKey(SavePrivateKey(priv)) != priv")
	}
}

func TestKeyExists_FalseForMissingFile(t *testing.T) {
	if KeyExists(filepath.Join(t.TempDir(), "nope.key")) {
		t.Error("KeyExists() = true for a file that was never written")
	}
}

func TestLoadPrivateKey_RejectsWrongLength(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.key")
	// "AA==" decodes to a single zero byte, far short of an ed25519 key.
	if err := os.WriteFile(path, []byte("AA==\n"), 0o600); err != nil {
		t.Fatalf("os.WriteFile: %v", err)
	}
	if _, err := LoadPrivateKey(path); err == nil {
		t.Error("LoadPrivateKey with wrong-length key: want error, got nil")
	}
}

func TestKeyPath_UsesSlugOfIdentityURL(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	path, err := KeyPath("https://ligatures.example")
	if err != nil {
		t.Fatalf("KeyPath returned error: %v", err)
	}
	if filepath.Base(path) != "ligatures.example.key" {
		t.Errorf("KeyPath() basename = %q, want %q", filepath.Base(path), "ligatures.example.key")
	}
}

func TestCanonicalWithoutField_DropsNamedField(t *testing.T) {
	type withSig struct {
		Name      string `json:"name"`
		Signature string `json:"signature"`
	}
	got, err := CanonicalWithoutField(withSig{Name: "x", Signature: "should-be-dropped"}, "signature")
	if err != nil {
		t.Fatalf("CanonicalWithoutField returned error: %v", err)
	}
	want := `{"name":"x"}`
	if string(got) != want {
		t.Errorf("CanonicalWithoutField() = %s, want %s", got, want)
	}
}
