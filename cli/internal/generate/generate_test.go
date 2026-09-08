package generate

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"endonend/cli/internal/manifest"
	"endonend/cli/internal/signing"
)

func fixedNow() time.Time {
	return time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
}

func minimalSource() manifest.Source {
	return manifest.Source{
		ManifestVersion: "1.0",
		Identity: manifest.SourceIdentity{
			Type: "artist", Name: "Ligatures", URL: "https://ligatures.example", ContactEmail: "band@ligatures.example",
		},
		Refresh: manifest.Refresh{TTLSeconds: 21600},
		Catalog: []manifest.Album{
			{
				AlbumID: "agency-2024", AlbumName: "Agency", ReleaseDate: "2024-05-01",
				Images: manifest.Images{Front: "https://x/front.png", Back: "https://x/back.png", Insert: []string{}},
				Tracks: []manifest.Track{
					{TrackID: "a1", Number: "A1", Name: "Opening", Duration: "3'30\"", File: "https://x/opening.mp3"},
				},
			},
		},
	}
}

// testEnv sets up an isolated HOME (so key generation doesn't touch the
// real ~/.endonend/keys) and a scratch directory for source/manifest/history
// files.
func testEnv(t *testing.T) (dir string) {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
	return t.TempDir()
}

func writeSource(t *testing.T, dir string, src manifest.Source) string {
	t.Helper()
	path := filepath.Join(dir, "endonend.source.json")
	raw, err := json.MarshalIndent(src, "", "  ")
	if err != nil {
		t.Fatalf("marshal source: %v", err)
	}
	if err := os.WriteFile(path, raw, 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}
	return path
}

func opts(dir, sourcePath string) Options {
	return Options{
		SourcePath:      sourcePath,
		ManifestOutPath: filepath.Join(dir, "manifest.json"),
		HistoryOutPath:  filepath.Join(dir, "history.json"),
		Now:             fixedNow,
	}
}

func TestRun_FirstGenerate(t *testing.T) {
	dir := testEnv(t)
	sourcePath := writeSource(t, dir, minimalSource())
	o := opts(dir, sourcePath)

	result, err := Run(o)
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if !result.KeyGenerated {
		t.Error("KeyGenerated = false on a first generate, want true")
	}
	if len(result.History) != 1 || result.History[0].Type != manifest.HistoryTypeReleaseAdded {
		t.Fatalf("History = %+v, want a single release_added entry", result.History)
	}
	if result.Manifest.Catalog[0].AlbumVersion != 1 {
		t.Errorf("AlbumVersion = %d, want 1", result.Manifest.Catalog[0].AlbumVersion)
	}
	if result.Manifest.History.HeadHash == "" {
		t.Error("manifest.history.headHash is empty after generating with history entries")
	}

	pub, err := signing.DecodePublicKey(result.Manifest.Identity.PublicKey)
	if err != nil {
		t.Fatalf("DecodePublicKey: %v", err)
	}
	canonicalBytes, err := signing.CanonicalWithoutField(result.Manifest, "signature")
	if err != nil {
		t.Fatalf("CanonicalWithoutField: %v", err)
	}
	ok, err := signing.VerifyCanonical(pub, canonicalBytes, result.Manifest.Signature.Value)
	if err != nil {
		t.Fatalf("VerifyCanonical: %v", err)
	}
	if !ok {
		t.Error("manifest signature does not verify against its own identity.publicKey")
	}

	if _, err := os.Stat(o.ManifestOutPath); err != nil {
		t.Errorf("manifest was not written to disk: %v", err)
	}
	if _, err := os.Stat(o.HistoryOutPath); err != nil {
		t.Errorf("history was not written to disk: %v", err)
	}
}

func TestRun_SecondGenerateWithNoChangesAddsNoEntries(t *testing.T) {
	dir := testEnv(t)
	sourcePath := writeSource(t, dir, minimalSource())
	o := opts(dir, sourcePath)

	if _, err := Run(o); err != nil {
		t.Fatalf("first Run returned error: %v", err)
	}
	result, err := Run(o)
	if err != nil {
		t.Fatalf("second Run returned error: %v", err)
	}
	if result.KeyGenerated {
		t.Error("KeyGenerated = true on second generate, want false (key already existed)")
	}
	if len(result.NewEntries) != 0 {
		t.Errorf("NewEntries = %+v, want none for an unchanged source", result.NewEntries)
	}
	if len(result.History) != 1 {
		t.Errorf("total history length = %d, want 1 (unchanged from first generate)", len(result.History))
	}
}

func TestRun_AlbumContentChangeBumpsVersionAndRecordsUpdate(t *testing.T) {
	dir := testEnv(t)
	src := minimalSource()
	sourcePath := writeSource(t, dir, src)
	o := opts(dir, sourcePath)
	if _, err := Run(o); err != nil {
		t.Fatalf("first Run returned error: %v", err)
	}

	src.Catalog[0].Tracks[0].Name = "Opening (Remaster)"
	writeSource(t, dir, src)

	result, err := Run(o)
	if err != nil {
		t.Fatalf("second Run returned error: %v", err)
	}
	if len(result.NewEntries) != 1 || result.NewEntries[0].Type != manifest.HistoryTypeReleaseUpdated {
		t.Fatalf("NewEntries = %+v, want a single release_updated entry", result.NewEntries)
	}
	if result.Manifest.Catalog[0].AlbumVersion != 2 {
		t.Errorf("AlbumVersion = %d, want 2 after a content change", result.Manifest.Catalog[0].AlbumVersion)
	}
	data := result.NewEntries[0].Data
	if data["oldAlbumVersion"].(int) != 1 || data["newAlbumVersion"].(int) != 2 {
		t.Errorf("release_updated data = %+v, want old=1 new=2", data)
	}
}

func TestRun_AlbumRemovedRecordsRemoval(t *testing.T) {
	dir := testEnv(t)
	src := minimalSource()
	sourcePath := writeSource(t, dir, src)
	o := opts(dir, sourcePath)
	if _, err := Run(o); err != nil {
		t.Fatalf("first Run returned error: %v", err)
	}

	// An artist manifest can't have an empty catalog, so replace the
	// original album with a different one to exercise release_removed.
	src.Catalog = []manifest.Album{
		{
			AlbumID: "other-album", AlbumName: "Other", ReleaseDate: "2025-01-01",
			Images: manifest.Images{Front: "https://x/front2.png", Back: "https://x/back2.png", Insert: []string{}},
			Tracks: []manifest.Track{{TrackID: "b1", Number: "A1", Name: "Only Track", Duration: "1'00\"", File: "https://x/only.mp3"}},
		},
	}
	writeSource(t, dir, src)

	result, err := Run(o)
	if err != nil {
		t.Fatalf("second Run returned error: %v", err)
	}
	types := map[string]bool{}
	for _, e := range result.NewEntries {
		types[e.Type] = true
	}
	if !types[manifest.HistoryTypeReleaseAdded] || !types[manifest.HistoryTypeReleaseRemoved] {
		t.Errorf("NewEntries = %+v, want both release_added and release_removed", result.NewEntries)
	}
}

func TestRun_IdentityChangeRecordsUpdate(t *testing.T) {
	dir := testEnv(t)
	src := minimalSource()
	sourcePath := writeSource(t, dir, src)
	o := opts(dir, sourcePath)
	if _, err := Run(o); err != nil {
		t.Fatalf("first Run returned error: %v", err)
	}

	src.Identity.ContactEmail = "new@ligatures.example"
	writeSource(t, dir, src)

	result, err := Run(o)
	if err != nil {
		t.Fatalf("second Run returned error: %v", err)
	}
	if len(result.NewEntries) != 1 || result.NewEntries[0].Type != manifest.HistoryTypeIdentityUpdated {
		t.Fatalf("NewEntries = %+v, want a single identity_updated entry", result.NewEntries)
	}
}

func TestRun_MissingHistoryAlongsideExistingManifestErrors(t *testing.T) {
	dir := testEnv(t)
	sourcePath := writeSource(t, dir, minimalSource())
	o := opts(dir, sourcePath)
	if _, err := Run(o); err != nil {
		t.Fatalf("first Run returned error: %v", err)
	}
	if err := os.Remove(o.HistoryOutPath); err != nil {
		t.Fatalf("remove history: %v", err)
	}
	if _, err := Run(o); err == nil {
		t.Error("Run with a manifest but no history file: want error, got nil")
	}
}

func TestValidateSource_RejectsInvalidInput(t *testing.T) {
	valid := minimalSource()

	cases := map[string]func(*manifest.Source){
		"missing manifestVersion": func(s *manifest.Source) { s.ManifestVersion = "" },
		"bad identity.type":       func(s *manifest.Source) { s.Identity.Type = "band" },
		"missing name":            func(s *manifest.Source) { s.Identity.Name = "" },
		"missing url":             func(s *manifest.Source) { s.Identity.URL = "" },
		"missing contact email":   func(s *manifest.Source) { s.Identity.ContactEmail = "" },
		"non-positive ttl":        func(s *manifest.Source) { s.Refresh.TTLSeconds = 0 },
		"artist with no catalog":  func(s *manifest.Source) { s.Catalog = nil },
		"duplicate albumId": func(s *manifest.Source) {
			s.Catalog = append(s.Catalog, s.Catalog[0])
		},
		"missing front image": func(s *manifest.Source) { s.Catalog[0].Images.Front = "" },
		"missing trackId":     func(s *manifest.Source) { s.Catalog[0].Tracks[0].TrackID = "" },
	}

	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			s := valid
			s.Catalog = append([]manifest.Album{}, valid.Catalog...)
			s.Catalog[0] = valid.Catalog[0]
			s.Catalog[0].Tracks = append([]manifest.Track{}, valid.Catalog[0].Tracks...)
			mutate(&s)
			if err := validateSource(&s); err == nil {
				t.Errorf("validateSource(%s): want error, got nil", name)
			}
		})
	}
}

func TestValidateSource_RejectsCatalogOnLabelManifest(t *testing.T) {
	s := minimalSource()
	s.Identity.Type = "label"
	if err := validateSource(&s); err == nil {
		t.Error("validateSource with a label manifest carrying a catalog: want error, got nil")
	}
}

func TestValidateSource_AcceptsMinimalLabelManifest(t *testing.T) {
	s := manifest.Source{
		ManifestVersion: "1.0",
		Identity: manifest.SourceIdentity{
			Type: "label", Name: "Small Label", URL: "https://smalllabel.example", ContactEmail: "hello@smalllabel.example",
		},
		Refresh: manifest.Refresh{TTLSeconds: 21600},
	}
	if err := validateSource(&s); err != nil {
		t.Errorf("validateSource(minimal label): unexpected error: %v", err)
	}
}

func TestAlbumFingerprint_IgnoresAlbumVersion(t *testing.T) {
	a1 := manifest.Album{AlbumID: "x", AlbumVersion: 1, AlbumName: "X"}
	a2 := manifest.Album{AlbumID: "x", AlbumVersion: 2, AlbumName: "X"}
	f1, err := albumFingerprint(a1)
	if err != nil {
		t.Fatalf("albumFingerprint: %v", err)
	}
	f2, err := albumFingerprint(a2)
	if err != nil {
		t.Fatalf("albumFingerprint: %v", err)
	}
	if f1 != f2 {
		t.Error("albumFingerprint differs only because of albumVersion, want it ignored")
	}
}

func TestAlbumFingerprint_DiffersOnContentChange(t *testing.T) {
	a1 := manifest.Album{AlbumID: "x", AlbumVersion: 1, AlbumName: "X"}
	a2 := manifest.Album{AlbumID: "x", AlbumVersion: 1, AlbumName: "Y"}
	f1, err := albumFingerprint(a1)
	if err != nil {
		t.Fatalf("albumFingerprint: %v", err)
	}
	f2, err := albumFingerprint(a2)
	if err != nil {
		t.Fatalf("albumFingerprint: %v", err)
	}
	if f1 == f2 {
		t.Error("albumFingerprint identical for albums with different names")
	}
}

func TestRun_UsesExistingKeyOnSecondGenerate(t *testing.T) {
	dir := testEnv(t)
	sourcePath := writeSource(t, dir, minimalSource())
	o := opts(dir, sourcePath)

	first, err := Run(o)
	if err != nil {
		t.Fatalf("first Run returned error: %v", err)
	}
	second, err := Run(o)
	if err != nil {
		t.Fatalf("second Run returned error: %v", err)
	}
	if first.PublicKey != second.PublicKey {
		t.Error("public key changed between generate runs without a rotation")
	}
}
