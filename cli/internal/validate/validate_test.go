package validate

import (
	"crypto/ed25519"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"endonend/cli/internal/history"
	"endonend/cli/internal/manifest"
	"endonend/cli/internal/signing"
)

// buildSignedManifest returns a minimal, validly signed manifest and its
// paired history log, along with the private key that signed it.
func buildSignedManifest(t *testing.T) (manifest.Manifest, []manifest.HistoryEntry, ed25519.PrivateKey) {
	t.Helper()
	pub, priv, err := signing.GenerateKeypair()
	if err != nil {
		t.Fatalf("GenerateKeypair: %v", err)
	}

	entry := manifest.HistoryEntry{
		Timestamp: "2024-01-01T00:00:00Z",
		Type:      manifest.HistoryTypeReleaseAdded,
		Data:      map[string]any{"albumId": "agency-2024"},
	}
	canonicalEntry, err := signing.CanonicalWithoutField(entry, "signature")
	if err != nil {
		t.Fatalf("CanonicalWithoutField: %v", err)
	}
	entry.Signature = manifest.Signature{Algorithm: "ed25519", Value: signing.SignCanonical(priv, canonicalEntry)}
	head, err := history.Hash(entry)
	if err != nil {
		t.Fatalf("history.Hash: %v", err)
	}

	m := manifest.Manifest{
		ManifestVersion: "1.0",
		Identity: manifest.Identity{
			Type: "artist", Name: "Ligatures", URL: "https://ligatures.example",
			PublicKey: signing.EncodePublicKey(pub), ContactEmail: "band@ligatures.example",
		},
		Refresh: manifest.Refresh{TTLSeconds: 21600},
		History: manifest.HistoryRef{URL: "https://ligatures.example/.well-known/endonend/history.json", HeadHash: head},
		Catalog: []manifest.Album{
			{
				AlbumID: "agency-2024", AlbumVersion: 1, AlbumName: "Agency", ReleaseDate: "2024-05-01",
				Images: manifest.Images{Front: "https://x/front.png", Back: "https://x/back.png", Insert: []string{}},
				Tracks: []manifest.Track{
					{TrackID: "a1", Number: "A1", Name: "Opening", Duration: "3'30\"", File: "https://x/opening.mp3"},
				},
			},
		},
	}
	canonicalManifest, err := signing.CanonicalWithoutField(&m, "signature")
	if err != nil {
		t.Fatalf("CanonicalWithoutField: %v", err)
	}
	m.Signature = manifest.Signature{Algorithm: "ed25519", Value: signing.SignCanonical(priv, canonicalManifest)}

	return m, []manifest.HistoryEntry{entry}, priv
}

func writeManifestAndHistory(t *testing.T, m manifest.Manifest, entries []manifest.HistoryEntry) string {
	t.Helper()
	dir := t.TempDir()
	manifestPath := filepath.Join(dir, "manifest.json")
	writeJSON(t, manifestPath, m)
	writeJSON(t, filepath.Join(dir, "history.json"), entries)
	return manifestPath
}

func writeJSON(t *testing.T, path string, v any) {
	t.Helper()
	raw, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		t.Fatalf("marshal %s: %v", path, err)
	}
	if err := os.WriteFile(path, raw, 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func TestValidatePath_ValidManifest(t *testing.T) {
	m, entries, _ := buildSignedManifest(t)
	path := writeManifestAndHistory(t, m, entries)

	report, err := ValidatePath(path, Options{})
	if err != nil {
		t.Fatalf("ValidatePath returned error: %v", err)
	}
	if !report.Valid {
		t.Errorf("report.Valid = false, want true; failures: %+v", report.Failures)
	}
	if len(report.Failures) != 0 {
		t.Errorf("report.Failures = %+v, want none", report.Failures)
	}
}

func TestValidatePath_TamperedFieldFailsSignature(t *testing.T) {
	m, entries, _ := buildSignedManifest(t)
	m.Identity.Name = "Someone Else"
	path := writeManifestAndHistory(t, m, entries)

	report, err := ValidatePath(path, Options{})
	if err != nil {
		t.Fatalf("ValidatePath returned error: %v", err)
	}
	if report.Valid {
		t.Fatal("report.Valid = true for a tampered manifest, want false")
	}
	if !hasFailureField(report, "signature.value") {
		t.Errorf("failures = %+v, want one on signature.value", report.Failures)
	}
}

func TestValidatePath_RejectsDisallowedAlgorithm(t *testing.T) {
	m, entries, _ := buildSignedManifest(t)
	m.Signature.Algorithm = "none"
	path := writeManifestAndHistory(t, m, entries)

	report, err := ValidatePath(path, Options{})
	if err != nil {
		t.Fatalf("ValidatePath returned error: %v", err)
	}
	if report.Valid {
		t.Fatal("report.Valid = true with signature.algorithm \"none\", want false")
	}
	if !hasFailureField(report, "signature.algorithm") {
		t.Errorf("failures = %+v, want one on signature.algorithm", report.Failures)
	}
}

func TestValidatePath_RejectsDisallowedHistoryEntryAlgorithm(t *testing.T) {
	m, entries, _ := buildSignedManifest(t)
	entries[0].Signature.Algorithm = "rsa"
	path := writeManifestAndHistory(t, m, entries)

	report, err := ValidatePath(path, Options{})
	if err != nil {
		t.Fatalf("ValidatePath returned error: %v", err)
	}
	if report.Valid {
		t.Fatal("report.Valid = true with a history entry's algorithm != ed25519, want false")
	}
}

func TestValidatePath_MissingImagesField(t *testing.T) {
	m, entries, _ := buildSignedManifest(t)
	dir := t.TempDir()
	manifestPath := filepath.Join(dir, "manifest.json")

	// Bypass the struct (which always carries an images object) to
	// simulate a manifest whose JSON genuinely omits images.insert.
	raw, err := json.Marshal(m)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var generic map[string]any
	if err := json.Unmarshal(raw, &generic); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	catalog := generic["catalog"].([]any)
	album := catalog[0].(map[string]any)
	images := album["images"].(map[string]any)
	delete(images, "insert")
	out, err := json.Marshal(generic)
	if err != nil {
		t.Fatalf("marshal generic: %v", err)
	}
	if err := os.WriteFile(manifestPath, out, 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	writeJSON(t, filepath.Join(dir, "history.json"), entries)

	report, err := ValidatePath(manifestPath, Options{})
	if err != nil {
		t.Fatalf("ValidatePath returned error: %v", err)
	}
	if !hasFailureField(report, "catalog[0].images.insert") {
		t.Errorf("failures = %+v, want one on catalog[0].images.insert", report.Failures)
	}
}

func TestValidatePath_SplitsMustSumTo100(t *testing.T) {
	m, entries, priv := buildSignedManifest(t)
	m.Catalog[0].Splits = []manifest.AlbumSplitEntry{
		{ManifestURL: "https://ligatures.example/.well-known/endonend/manifest.json", Role: "primary", Percentage: 60},
	}
	// Re-sign so the failure we detect is the split rule, not the
	// signature (which would otherwise also legitimately fail).
	resign(t, &m, priv)
	path := writeManifestAndHistory(t, m, entries)

	report, err := ValidatePath(path, Options{})
	if err != nil {
		t.Fatalf("ValidatePath returned error: %v", err)
	}
	if report.Valid {
		t.Fatal("report.Valid = true with splits summing to 60, want false")
	}
	if !hasFailureField(report, "catalog[0].splits") {
		t.Errorf("failures = %+v, want one on catalog[0].splits", report.Failures)
	}
}

func TestValidatePath_DuplicateAlbumID(t *testing.T) {
	m, entries, priv := buildSignedManifest(t)
	m.Catalog = append(m.Catalog, m.Catalog[0])
	resign(t, &m, priv)
	path := writeManifestAndHistory(t, m, entries)

	report, err := ValidatePath(path, Options{})
	if err != nil {
		t.Fatalf("ValidatePath returned error: %v", err)
	}
	if report.Valid {
		t.Fatal("report.Valid = true with duplicate albumId, want false")
	}
}

func TestValidatePath_BrokenHistoryChain(t *testing.T) {
	m, entries, _ := buildSignedManifest(t)
	entries[0].PreviousHash = "sha256:notgenesis"
	path := writeManifestAndHistory(t, m, entries)

	report, err := ValidatePath(path, Options{})
	if err != nil {
		t.Fatalf("ValidatePath returned error: %v", err)
	}
	if report.Valid {
		t.Fatal("report.Valid = true with a broken history chain, want false")
	}
	if !hasFailureField(report, "history") {
		t.Errorf("failures = %+v, want one on history", report.Failures)
	}
}

func TestValidatePath_HeadHashMismatch(t *testing.T) {
	m, entries, priv := buildSignedManifest(t)
	m.History.HeadHash = "sha256:wrong"
	resign(t, &m, priv)
	path := writeManifestAndHistory(t, m, entries)

	report, err := ValidatePath(path, Options{})
	if err != nil {
		t.Fatalf("ValidatePath returned error: %v", err)
	}
	if report.Valid {
		t.Fatal("report.Valid = true with mismatched headHash, want false")
	}
	if !hasFailureField(report, "history.headHash") {
		t.Errorf("failures = %+v, want one on history.headHash", report.Failures)
	}
}

func TestValidatePath_MissingHistoryFile(t *testing.T) {
	m, _, _ := buildSignedManifest(t)
	dir := t.TempDir()
	manifestPath := filepath.Join(dir, "manifest.json")
	writeJSON(t, manifestPath, m)
	// Deliberately do not write history.json.

	report, err := ValidatePath(manifestPath, Options{})
	if err != nil {
		t.Fatalf("ValidatePath returned error: %v", err)
	}
	if report.Valid {
		t.Fatal("report.Valid = true with no history.json present, want false")
	}
}

func TestValidatePath_KeyRotationChainVerifiesAcrossRotation(t *testing.T) {
	m, entries, oldPriv := buildSignedManifest(t)
	newPub, newPriv, err := signing.GenerateKeypair()
	if err != nil {
		t.Fatalf("GenerateKeypair: %v", err)
	}
	oldPubStr := m.Identity.PublicKey
	newPubStr := signing.EncodePublicKey(newPub)

	rotation := manifest.HistoryEntry{
		Timestamp: "2024-02-01T00:00:00Z",
		Type:      manifest.HistoryTypeKeyRotated,
		Data:      map[string]any{"oldPublicKey": oldPubStr, "newPublicKey": newPubStr},
	}
	prevHash, err := history.Hash(entries[0])
	if err != nil {
		t.Fatalf("history.Hash: %v", err)
	}
	rotation.PreviousHash = prevHash
	canonicalRotation, err := signing.CanonicalWithoutField(rotation, "signature")
	if err != nil {
		t.Fatalf("CanonicalWithoutField: %v", err)
	}
	rotation.Signature = manifest.Signature{Algorithm: "ed25519", Value: signing.SignCanonical(oldPriv, canonicalRotation)}
	entries = append(entries, rotation)

	m.Identity.PublicKey = newPubStr
	head, err := history.Hash(rotation)
	if err != nil {
		t.Fatalf("history.Hash: %v", err)
	}
	m.History.HeadHash = head
	canonicalManifest, err := signing.CanonicalWithoutField(&m, "signature")
	if err != nil {
		t.Fatalf("CanonicalWithoutField: %v", err)
	}
	m.Signature = manifest.Signature{Algorithm: "ed25519", Value: signing.SignCanonical(newPriv, canonicalManifest)}

	path := writeManifestAndHistory(t, m, entries)
	report, err := ValidatePath(path, Options{})
	if err != nil {
		t.Fatalf("ValidatePath returned error: %v", err)
	}
	if !report.Valid {
		t.Errorf("report.Valid = false after a properly authorized rotation, want true; failures: %+v", report.Failures)
	}
}

func TestValidatePath_KeyRotationSignedByWrongKeyFails(t *testing.T) {
	m, entries, _ := buildSignedManifest(t)
	attackerPub, attackerPriv, err := signing.GenerateKeypair()
	if err != nil {
		t.Fatalf("GenerateKeypair: %v", err)
	}
	rotation := manifest.HistoryEntry{
		Timestamp: "2024-02-01T00:00:00Z",
		Type:      manifest.HistoryTypeKeyRotated,
		Data:      map[string]any{"oldPublicKey": m.Identity.PublicKey, "newPublicKey": signing.EncodePublicKey(attackerPub)},
	}
	prevHash, err := history.Hash(entries[0])
	if err != nil {
		t.Fatalf("history.Hash: %v", err)
	}
	rotation.PreviousHash = prevHash
	canonicalRotation, err := signing.CanonicalWithoutField(rotation, "signature")
	if err != nil {
		t.Fatalf("CanonicalWithoutField: %v", err)
	}
	// Signed by the attacker's own new key, not the retiring old key:
	// exactly the hijack scenario 0003's Key rotation section guards
	// against.
	rotation.Signature = manifest.Signature{Algorithm: "ed25519", Value: signing.SignCanonical(attackerPriv, canonicalRotation)}
	entries = append(entries, rotation)

	m.Identity.PublicKey = signing.EncodePublicKey(attackerPub)
	head, err := history.Hash(rotation)
	if err != nil {
		t.Fatalf("history.Hash: %v", err)
	}
	m.History.HeadHash = head
	canonicalManifest, err := signing.CanonicalWithoutField(&m, "signature")
	if err != nil {
		t.Fatalf("CanonicalWithoutField: %v", err)
	}
	m.Signature = manifest.Signature{Algorithm: "ed25519", Value: signing.SignCanonical(attackerPriv, canonicalManifest)}

	path := writeManifestAndHistory(t, m, entries)
	report, err := ValidatePath(path, Options{})
	if err != nil {
		t.Fatalf("ValidatePath returned error: %v", err)
	}
	if report.Valid {
		t.Fatal("report.Valid = true for a key rotation not signed by the retiring key, want false")
	}
}

func TestValidateURL_FetchLocationMustMatchIdentityURL(t *testing.T) {
	m, entries, _ := buildSignedManifest(t)
	// identity.url is https://ligatures.example, but we serve it from a
	// completely different host.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/.well-known/endonend/manifest.json":
			_ = json.NewEncoder(w).Encode(m)
		case "/history.json":
			_ = json.NewEncoder(w).Encode(entries)
		}
	}))
	defer server.Close()

	report, err := ValidateURL(server.URL+"/.well-known/endonend/manifest.json", Options{})
	if err != nil {
		t.Fatalf("ValidateURL returned error: %v", err)
	}
	if report.Valid {
		t.Fatal("report.Valid = true when fetched-from URL doesn't match identity.url, want false")
	}
	if !hasFailureField(report, "identity.url") {
		t.Errorf("failures = %+v, want one on identity.url", report.Failures)
	}
}

func TestDeep_LabelAffiliationVerifiedOnMutualAttestation(t *testing.T) {
	artist, entries, artistPriv := buildSignedManifest(t)
	artistServer := newManifestServer(t, &artist, &entries)
	defer artistServer.Close()
	selfURL := artistServer.URL + "/.well-known/endonend/manifest.json"
	artist.Identity.URL = artistServer.URL
	artist.History.URL = artistServer.URL + "/history.json"

	label := manifest.Manifest{
		ManifestVersion: "1.0",
		Identity:        manifest.Identity{Type: "label", Name: "Small Label", ContactEmail: "hello@smalllabel.example"},
		Refresh:         manifest.Refresh{TTLSeconds: 21600},
		Label: &manifest.Label{Roster: []manifest.RosterEntry{
			{ArtistManifestURL: selfURL, Split: manifest.Split{Artist: 85, Label: 15}},
		}},
	}
	labelServer := newManifestServer(t, &label, nil)
	defer labelServer.Close()
	label.Identity.URL = labelServer.URL
	labelPub, labelPriv, err := signing.GenerateKeypair()
	if err != nil {
		t.Fatalf("GenerateKeypair: %v", err)
	}
	label.Identity.PublicKey = signing.EncodePublicKey(labelPub)
	resign(t, &label, labelPriv)

	artist.Label = &manifest.Label{AffiliatedLabel: labelServer.URL, Split: &manifest.Split{Artist: 85, Label: 15}}
	resign(t, &artist, artistPriv)

	report, err := ValidateURL(selfURL, Options{Deep: true})
	if err != nil {
		t.Fatalf("ValidateURL returned error: %v", err)
	}
	if !report.Valid {
		t.Errorf("report.Valid = false with a mutually-attesting label, want true; failures: %+v", report.Failures)
	}
}

func TestDeep_LabelAffiliationUnverifiedWhenRosterMissingArtist(t *testing.T) {
	artist, entries, artistPriv := buildSignedManifest(t)
	artistServer := newManifestServer(t, &artist, &entries)
	defer artistServer.Close()
	selfURL := artistServer.URL + "/.well-known/endonend/manifest.json"
	artist.Identity.URL = artistServer.URL
	artist.History.URL = artistServer.URL + "/history.json"

	label := manifest.Manifest{
		ManifestVersion: "1.0",
		Identity:        manifest.Identity{Type: "label", Name: "Small Label", ContactEmail: "hello@smalllabel.example"},
		Refresh:         manifest.Refresh{TTLSeconds: 21600},
		Label:           &manifest.Label{Roster: nil},
	}
	labelServer := newManifestServer(t, &label, nil)
	defer labelServer.Close()
	label.Identity.URL = labelServer.URL
	labelPub, labelPriv, err := signing.GenerateKeypair()
	if err != nil {
		t.Fatalf("GenerateKeypair: %v", err)
	}
	label.Identity.PublicKey = signing.EncodePublicKey(labelPub)
	resign(t, &label, labelPriv)

	artist.Label = &manifest.Label{AffiliatedLabel: labelServer.URL, Split: &manifest.Split{Artist: 85, Label: 15}}
	resign(t, &artist, artistPriv)

	report, err := ValidateURL(selfURL, Options{Deep: true})
	if err != nil {
		t.Fatalf("ValidateURL returned error: %v", err)
	}
	if report.Valid {
		t.Fatal("report.Valid = true when the label's roster doesn't list the artist back, want false")
	}
	if !hasKind(report, KindUnverified) {
		t.Errorf("failures = %+v, want an unverified kind", report.Failures)
	}
}

func TestDeep_AlbumSplitDisputedOnMismatchedContribution(t *testing.T) {
	artist, entries, artistPriv := buildSignedManifest(t)
	artistServer := newManifestServer(t, &artist, &entries)
	defer artistServer.Close()
	selfURL := artistServer.URL + "/.well-known/endonend/manifest.json"
	artist.Identity.URL = artistServer.URL
	artist.History.URL = artistServer.URL + "/history.json"

	feature := manifest.Manifest{
		ManifestVersion: "1.0",
		Identity:        manifest.Identity{Type: "artist", Name: "Jane Doe", ContactEmail: "jane@example.com"},
		Refresh:         manifest.Refresh{TTLSeconds: 21600},
		Contributions: []manifest.Contribution{
			{
				ManifestURL: selfURL,
				AlbumID:     "agency-2024", AlbumVersion: 1, Role: "feature", Percentage: 10, // disputed: artist claims 20
			},
		},
	}
	featureServer := newManifestServer(t, &feature, nil)
	defer featureServer.Close()
	feature.Identity.URL = featureServer.URL
	featurePub, featurePriv, err := signing.GenerateKeypair()
	if err != nil {
		t.Fatalf("GenerateKeypair: %v", err)
	}
	feature.Identity.PublicKey = signing.EncodePublicKey(featurePub)
	resign(t, &feature, featurePriv)

	artist.Catalog[0].Splits = []manifest.AlbumSplitEntry{
		{ManifestURL: selfURL, Role: "primary", Percentage: 80},
		{ManifestURL: featureServer.URL, Role: "feature", Percentage: 20},
	}
	resign(t, &artist, artistPriv)

	report, err := ValidateURL(selfURL, Options{Deep: true})
	if err != nil {
		t.Fatalf("ValidateURL returned error: %v", err)
	}
	if report.Valid {
		t.Fatal("report.Valid = true with a disputed album split, want false")
	}
	if !hasKind(report, KindDisputed) {
		t.Errorf("failures = %+v, want a disputed kind", report.Failures)
	}
}

// newManifestServer serves the current value of *m (and, if entries is
// non-nil, *entries at /history.json) on every request, read at request
// time so a test can keep mutating m up until the moment it calls
// ValidateURL, without needing to know the server's URL up front.
func newManifestServer(t *testing.T, m *manifest.Manifest, entries *[]manifest.HistoryEntry) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if entries != nil && r.URL.Path == "/history.json" {
			_ = json.NewEncoder(w).Encode(*entries)
			return
		}
		_ = json.NewEncoder(w).Encode(*m)
	}))
}

// resign re-signs m with priv, the same key that already signed its
// paired history log. Using a fresh key here instead would make the
// history chain's key-rotation check fail for an unrelated reason,
// since identity.publicKey would no longer match whatever key signed
// the existing history entries.
func resign(t *testing.T, m *manifest.Manifest, priv ed25519.PrivateKey) {
	t.Helper()
	m.Identity.PublicKey = signing.EncodePublicKey(priv.Public().(ed25519.PublicKey))
	canonicalBytes, err := signing.CanonicalWithoutField(m, "signature")
	if err != nil {
		t.Fatalf("CanonicalWithoutField: %v", err)
	}
	m.Signature = manifest.Signature{Algorithm: "ed25519", Value: signing.SignCanonical(priv, canonicalBytes)}
}

func hasFailureField(r *Report, field string) bool {
	for _, f := range r.Failures {
		if f.Field == field {
			return true
		}
	}
	return false
}

func hasKind(r *Report, kind Kind) bool {
	for _, f := range r.Failures {
		if f.Kind == kind {
			return true
		}
	}
	return false
}
