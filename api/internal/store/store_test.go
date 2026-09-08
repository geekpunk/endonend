package store

import (
	"context"
	"os"
	"testing"
	"time"

	"endonend/api/internal/db"
	"endonend/protocol/manifest"
)

// testStore returns a Store backed by a real Postgres, schema applied and
// every table truncated so each test starts from a clean slate. Skips when
// TEST_DATABASE_URL isn't set (for example, in CI, which doesn't run a
// Postgres service for this module).
func testStore(t *testing.T) *Store {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set, skipping Postgres integration test")
	}
	ctx := context.Background()
	pool, err := db.Open(ctx, dsn)
	if err != nil {
		t.Fatalf("db.Open: %v", err)
	}
	t.Cleanup(pool.Close)

	_, err = pool.Exec(ctx, `TRUNCATE identities, label_rosters, albums, tracks, album_splits, contributions, merch_links, purchase_links, history_entries RESTART IDENTITY CASCADE`)
	if err != nil {
		t.Fatalf("truncate tables: %v", err)
	}
	return New(pool)
}

func artistManifest(url, name string) *manifest.Manifest {
	return &manifest.Manifest{
		ManifestVersion: "1.0",
		Identity:        manifest.Identity{Type: "artist", Name: name, URL: url, PublicKey: "ed25519:test", ContactEmail: "band@" + name + ".example"},
		Refresh:         manifest.Refresh{TTLSeconds: 21600},
		History:         manifest.HistoryRef{URL: url + "/.well-known/endonend/history.json"},
		Merch:           []manifest.MerchLink{{Label: "Official Store", URL: url + "/store"}},
		Catalog: []manifest.Album{
			{
				AlbumID: "agency-2024", AlbumVersion: 1, AlbumName: "Agency", ReleaseDate: "2024-05-01",
				Images:        manifest.Images{Front: url + "/front.png", Back: url + "/back.png", Insert: []string{}},
				PurchaseLinks: []manifest.PurchaseLink{{Format: "vinyl", URL: url + "/store/vinyl"}},
				Tracks: []manifest.Track{
					{TrackID: "a1", Number: "A1", Name: "Opening", Duration: "3'30\"", File: url + "/opening.mp3"},
				},
			},
		},
	}
}

func TestUpsertManifest_ArtistRoundTrip(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	m := artistManifest("https://ligatures.example", "Ligatures")

	if err := s.UpsertManifest(ctx, m, []byte(`{"raw":true}`), nil, time.Now()); err != nil {
		t.Fatalf("UpsertManifest: %v", err)
	}

	got, err := s.GetIdentity(ctx, m.Identity.URL)
	if err != nil {
		t.Fatalf("GetIdentity: %v", err)
	}
	if got == nil {
		t.Fatal("GetIdentity returned nil after upsert")
	}
	if got.Name != "Ligatures" || got.Type != "artist" {
		t.Errorf("identity = %+v, want name=Ligatures type=artist", got)
	}

	artists, err := s.ListArtists(ctx, 10, 0)
	if err != nil {
		t.Fatalf("ListArtists: %v", err)
	}
	if len(artists) != 1 {
		t.Fatalf("ListArtists returned %d artists, want 1", len(artists))
	}

	albums, err := s.AlbumsForIdentity(ctx, m.Identity.URL)
	if err != nil {
		t.Fatalf("AlbumsForIdentity: %v", err)
	}
	if len(albums) != 1 || albums[0].AlbumName != "Agency" {
		t.Fatalf("AlbumsForIdentity = %+v, want one album named Agency", albums)
	}

	album, err := s.GetAlbum(ctx, m.Identity.URL, "agency-2024")
	if err != nil {
		t.Fatalf("GetAlbum: %v", err)
	}
	if album == nil {
		t.Fatal("GetAlbum returned nil for an album that was just upserted")
	}
	if len(album.Tracks) != 1 || album.Tracks[0].Name != "Opening" {
		t.Errorf("album.Tracks = %+v, want one track named Opening", album.Tracks)
	}
	if len(album.PurchaseLinks) != 1 || album.PurchaseLinks[0].Format != "vinyl" {
		t.Errorf("album.PurchaseLinks = %+v, want one vinyl link", album.PurchaseLinks)
	}

	merch, err := s.MerchForIdentity(ctx, m.Identity.URL)
	if err != nil {
		t.Fatalf("MerchForIdentity: %v", err)
	}
	if len(merch) != 1 || merch[0].Label != "Official Store" {
		t.Errorf("merch = %+v, want one Official Store link", merch)
	}
}

func TestUpsertManifest_ReplacesCatalogOnReupsert(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()
	m := artistManifest("https://ligatures.example", "Ligatures")

	if err := s.UpsertManifest(ctx, m, []byte("{}"), nil, time.Now()); err != nil {
		t.Fatalf("first UpsertManifest: %v", err)
	}

	m2 := artistManifest("https://ligatures.example", "Ligatures")
	m2.Catalog[0].AlbumID = "second-album"
	m2.Catalog[0].AlbumName = "Second Album"
	if err := s.UpsertManifest(ctx, m2, []byte("{}"), nil, time.Now()); err != nil {
		t.Fatalf("second UpsertManifest: %v", err)
	}

	albums, err := s.AlbumsForIdentity(ctx, m.Identity.URL)
	if err != nil {
		t.Fatalf("AlbumsForIdentity: %v", err)
	}
	if len(albums) != 1 {
		t.Fatalf("AlbumsForIdentity after re-upsert = %d albums, want 1 (old catalog should be replaced, not appended)", len(albums))
	}
	if albums[0].AlbumID != "second-album" {
		t.Errorf("AlbumsForIdentity()[0].AlbumID = %q, want second-album", albums[0].AlbumID)
	}

	old, err := s.GetAlbum(ctx, m.Identity.URL, "agency-2024")
	if err != nil {
		t.Fatalf("GetAlbum for the replaced album: %v", err)
	}
	if old != nil {
		t.Error("GetAlbum for the old album ID after re-upsert: want nil, got a result")
	}
}

func TestDueIdentities_ReflectsPerIdentityTTL(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()

	stale := artistManifest("https://stale.example", "Stale")
	stale.Refresh.TTLSeconds = 60
	fresh := artistManifest("https://fresh.example", "Fresh")
	fresh.Refresh.TTLSeconds = 3600

	now := time.Now()
	if err := s.UpsertManifest(ctx, stale, []byte("{}"), nil, now.Add(-2*time.Minute)); err != nil {
		t.Fatalf("upsert stale: %v", err)
	}
	if err := s.UpsertManifest(ctx, fresh, []byte("{}"), nil, now.Add(-2*time.Minute)); err != nil {
		t.Fatalf("upsert fresh: %v", err)
	}

	due, err := s.DueIdentities(ctx, now)
	if err != nil {
		t.Fatalf("DueIdentities: %v", err)
	}
	if len(due) != 1 || due[0] != stale.Identity.URL {
		t.Errorf("DueIdentities = %v, want only %s (60s TTL fetched 2m ago)", due, stale.Identity.URL)
	}
}

func TestRosterForLabel_VerificationStatuses(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()

	label := &manifest.Manifest{
		ManifestVersion: "1.0",
		Identity:        manifest.Identity{Type: "label", Name: "Small Label", URL: "https://smalllabel.example", ContactEmail: "hello@smalllabel.example"},
		Refresh:         manifest.Refresh{TTLSeconds: 21600},
		History:         manifest.HistoryRef{URL: "https://smalllabel.example/.well-known/endonend/history.json"},
		Label: &manifest.Label{Roster: []manifest.RosterEntry{
			{ArtistManifestURL: "https://verified.example", Split: manifest.Split{Artist: 85, Label: 15}},
			{ArtistManifestURL: "https://disputed.example", Split: manifest.Split{Artist: 85, Label: 15}},
			{ArtistManifestURL: "https://unverified.example", Split: manifest.Split{Artist: 85, Label: 15}},
			{ArtistManifestURL: "https://unknown.example", Split: manifest.Split{Artist: 85, Label: 15}},
		}},
	}
	if err := s.UpsertManifest(ctx, label, []byte("{}"), nil, time.Now()); err != nil {
		t.Fatalf("upsert label: %v", err)
	}

	verifiedArtist := artistManifest("https://verified.example", "Verified")
	verifiedArtist.Label = &manifest.Label{AffiliatedLabel: label.Identity.URL, Split: &manifest.Split{Artist: 85, Label: 15}}
	disputedArtist := artistManifest("https://disputed.example", "Disputed")
	disputedArtist.Label = &manifest.Label{AffiliatedLabel: label.Identity.URL, Split: &manifest.Split{Artist: 70, Label: 30}}
	unverifiedArtist := artistManifest("https://unverified.example", "Unverified")
	unverifiedArtist.Label = &manifest.Label{AffiliatedLabel: "https://someoneelse.example", Split: &manifest.Split{Artist: 85, Label: 15}}
	// https://unknown.example is deliberately never crawled/upserted, to exercise Pending.

	for _, m := range []*manifest.Manifest{verifiedArtist, disputedArtist, unverifiedArtist} {
		if err := s.UpsertManifest(ctx, m, []byte("{}"), nil, time.Now()); err != nil {
			t.Fatalf("upsert %s: %v", m.Identity.URL, err)
		}
	}

	roster, err := s.RosterForLabel(ctx, label.Identity.URL)
	if err != nil {
		t.Fatalf("RosterForLabel: %v", err)
	}
	got := map[string]VerificationStatus{}
	for _, r := range roster {
		got[r.ArtistURL] = r.Verified
	}
	want := map[string]VerificationStatus{
		"https://verified.example":   Verified,
		"https://disputed.example":   Disputed,
		"https://unverified.example": Unverified,
		"https://unknown.example":    Pending,
	}
	for url, wantStatus := range want {
		if got[url] != wantStatus {
			t.Errorf("roster verification for %s = %s, want %s", url, got[url], wantStatus)
		}
	}
}

func TestGetAlbum_SplitVerificationStatuses(t *testing.T) {
	s := testStore(t)
	ctx := context.Background()

	primary := artistManifest("https://ligatures.example", "Ligatures")
	primary.Catalog[0].Splits = []manifest.AlbumSplitEntry{
		{ManifestURL: "https://verified.example", Role: "feature", Percentage: 20},
		{ManifestURL: "https://disputed.example", Role: "feature", Percentage: 20},
		{ManifestURL: "https://unverified.example", Role: "feature", Percentage: 20},
		{ManifestURL: "https://unknown.example", Role: "feature", Percentage: 20},
	}
	if err := s.UpsertManifest(ctx, primary, []byte("{}"), nil, time.Now()); err != nil {
		t.Fatalf("upsert primary: %v", err)
	}

	verified := artistManifest("https://verified.example", "Verified")
	verified.Contributions = []manifest.Contribution{
		{ManifestURL: primary.Identity.URL, AlbumID: "agency-2024", AlbumVersion: 1, Role: "feature", Percentage: 20},
	}
	disputed := artistManifest("https://disputed.example", "Disputed")
	disputed.Contributions = []manifest.Contribution{
		{ManifestURL: primary.Identity.URL, AlbumID: "agency-2024", AlbumVersion: 1, Role: "feature", Percentage: 99},
	}
	unverified := artistManifest("https://unverified.example", "Unverified")
	// https://unknown.example is deliberately never crawled/upserted, to exercise Pending.

	for _, m := range []*manifest.Manifest{verified, disputed, unverified} {
		if err := s.UpsertManifest(ctx, m, []byte("{}"), nil, time.Now()); err != nil {
			t.Fatalf("upsert %s: %v", m.Identity.URL, err)
		}
	}

	album, err := s.GetAlbum(ctx, primary.Identity.URL, "agency-2024")
	if err != nil {
		t.Fatalf("GetAlbum: %v", err)
	}
	got := map[string]VerificationStatus{}
	for _, sp := range album.Splits {
		got[sp.ManifestURL] = sp.Verified
	}
	want := map[string]VerificationStatus{
		"https://verified.example":   Verified,
		"https://disputed.example":   Disputed,
		"https://unverified.example": Unverified,
		"https://unknown.example":    Pending,
	}
	for url, wantStatus := range want {
		if got[url] != wantStatus {
			t.Errorf("split verification for %s = %s, want %s", url, got[url], wantStatus)
		}
	}
}
