package store

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

type Identity struct {
	URL                string
	Type               string
	Name               string
	ContactEmail       string
	AffiliatedLabelURL string
	SplitArtistPct     *float64
	SplitLabelPct      *float64
	LastError          string
}

type MerchLink struct {
	Label string
	URL   string
}

type RosterEntry struct {
	ArtistURL      string
	SplitArtistPct float64
	SplitLabelPct  float64
	Verified       VerificationStatus
}

type AlbumSummary struct {
	IdentityURL  string
	IdentityName string
	AlbumID      string
	AlbumVersion int
	AlbumName    string
	ReleaseDate  string
	ImagesFront  string
}

type Track struct {
	TrackID  string
	Side     string
	Number   string
	Name     string
	Duration string
	File     string
	Lyrics   string
}

type AlbumSplit struct {
	ManifestURL string
	Role        string
	Percentage  float64
	Verified    VerificationStatus
}

type PurchaseLink struct {
	Format string
	URL    string
}

type Album struct {
	IdentityURL      string
	AlbumID          string
	AlbumVersion     int
	AlbumName        string
	PageTitle        string
	ReleaseDate      string
	ImagesFront      string
	ImagesBack       string
	ImagesInsert     []string
	DownloadZip      string
	CreditsJSON      []byte
	PresentationJSON []byte
	Splits           []AlbumSplit
	PurchaseLinks    []PurchaseLink
	Tracks           []Track
}

const identityColumns = `url, type, name, contact_email, affiliated_label_url, split_artist_pct, split_label_pct, last_error`

func scanIdentity(row pgx.Row) (*Identity, error) {
	var id Identity
	err := row.Scan(&id.URL, &id.Type, &id.Name, &id.ContactEmail, &id.AffiliatedLabelURL, &id.SplitArtistPct, &id.SplitLabelPct, &id.LastError)
	if err != nil {
		return nil, err
	}
	return &id, nil
}

// GetIdentity returns nil, nil (not an error) when url isn't indexed, since
// "unknown identity" is an ordinary, expected outcome for callers like
// roster/split verification below, not a failure.
func (s *Store) GetIdentity(ctx context.Context, url string) (*Identity, error) {
	row := s.pool.QueryRow(ctx, `SELECT `+identityColumns+` FROM identities WHERE url = $1`, url)
	id, err := scanIdentity(row)
	if isNoRows(err) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get identity %s: %w", url, err)
	}
	return id, nil
}

func (s *Store) listIdentities(ctx context.Context, identityType string, limit, offset int) ([]Identity, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT `+identityColumns+` FROM identities WHERE type = $1
		ORDER BY name LIMIT $2 OFFSET $3
	`, identityType, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("list %s identities: %w", identityType, err)
	}
	defer rows.Close()

	var out []Identity
	for rows.Next() {
		id, err := scanIdentity(rows)
		if err != nil {
			return nil, fmt.Errorf("scan identity: %w", err)
		}
		out = append(out, *id)
	}
	return out, rows.Err()
}

// DueIdentities returns every identity URL whose self-declared refresh TTL
// has elapsed since its last successful fetch, per KB/0002-architecture.md's
// TTL-driven refresh model: "the platform clamps declared TTLs" happens in
// the crawler's own validate call, not here, this just applies each row's
// already-clamped ttl_seconds.
func (s *Store) DueIdentities(ctx context.Context, now time.Time) ([]string, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT url FROM identities
		WHERE fetched_at + (ttl_seconds || ' seconds')::interval <= $1
		ORDER BY fetched_at
	`, now)
	if err != nil {
		return nil, fmt.Errorf("list due identities: %w", err)
	}
	defer rows.Close()

	var out []string
	for rows.Next() {
		var url string
		if err := rows.Scan(&url); err != nil {
			return nil, fmt.Errorf("scan due identity: %w", err)
		}
		out = append(out, url)
	}
	return out, rows.Err()
}

func (s *Store) ListArtists(ctx context.Context, limit, offset int) ([]Identity, error) {
	return s.listIdentities(ctx, "artist", limit, offset)
}

func (s *Store) ListLabels(ctx context.Context, limit, offset int) ([]Identity, error) {
	return s.listIdentities(ctx, "label", limit, offset)
}

func (s *Store) MerchForIdentity(ctx context.Context, url string) ([]MerchLink, error) {
	rows, err := s.pool.Query(ctx, `SELECT label, url FROM merch_links WHERE identity_url = $1 ORDER BY id`, url)
	if err != nil {
		return nil, fmt.Errorf("list merch links: %w", err)
	}
	defer rows.Close()

	var out []MerchLink
	for rows.Next() {
		var m MerchLink
		if err := rows.Scan(&m.Label, &m.URL); err != nil {
			return nil, fmt.Errorf("scan merch link: %w", err)
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

// RosterForLabel returns labelURL's declared roster, each entry's
// verification status computed by checking whether the named artist's own
// manifest confirms the same affiliation and split back.
func (s *Store) RosterForLabel(ctx context.Context, labelURL string) ([]RosterEntry, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT artist_url, split_artist_pct, split_label_pct FROM label_rosters WHERE label_url = $1 ORDER BY id
	`, labelURL)
	if err != nil {
		return nil, fmt.Errorf("list roster: %w", err)
	}
	defer rows.Close()

	type raw struct {
		artistURL               string
		splitArtist, splitLabel float64
	}
	var entries []raw
	for rows.Next() {
		var r raw
		if err := rows.Scan(&r.artistURL, &r.splitArtist, &r.splitLabel); err != nil {
			return nil, fmt.Errorf("scan roster entry: %w", err)
		}
		entries = append(entries, r)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	out := make([]RosterEntry, 0, len(entries))
	for _, r := range entries {
		artist, err := s.GetIdentity(ctx, r.artistURL)
		if err != nil {
			return nil, err
		}
		var status VerificationStatus
		switch {
		case artist == nil:
			status = Pending
		case artist.AffiliatedLabelURL != labelURL:
			status = Unverified
		case artist.SplitArtistPct == nil || artist.SplitLabelPct == nil:
			status = Unverified
		case *artist.SplitArtistPct == r.splitArtist && *artist.SplitLabelPct == r.splitLabel:
			status = Verified
		default:
			status = Disputed
		}
		out = append(out, RosterEntry{ArtistURL: r.artistURL, SplitArtistPct: r.splitArtist, SplitLabelPct: r.splitLabel, Verified: status})
	}
	return out, nil
}

func (s *Store) ListAlbums(ctx context.Context, limit, offset int) ([]AlbumSummary, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT a.identity_url, i.name, a.album_id, a.album_version, a.album_name, a.release_date, a.images_front
		FROM albums a JOIN identities i ON i.url = a.identity_url
		ORDER BY a.release_date DESC LIMIT $1 OFFSET $2
	`, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("list albums: %w", err)
	}
	defer rows.Close()

	var out []AlbumSummary
	for rows.Next() {
		var a AlbumSummary
		if err := rows.Scan(&a.IdentityURL, &a.IdentityName, &a.AlbumID, &a.AlbumVersion, &a.AlbumName, &a.ReleaseDate, &a.ImagesFront); err != nil {
			return nil, fmt.Errorf("scan album summary: %w", err)
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

func (s *Store) AlbumsForIdentity(ctx context.Context, identityURL string) ([]AlbumSummary, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT a.identity_url, i.name, a.album_id, a.album_version, a.album_name, a.release_date, a.images_front
		FROM albums a JOIN identities i ON i.url = a.identity_url
		WHERE a.identity_url = $1
		ORDER BY a.release_date DESC
	`, identityURL)
	if err != nil {
		return nil, fmt.Errorf("list albums for identity: %w", err)
	}
	defer rows.Close()

	var out []AlbumSummary
	for rows.Next() {
		var a AlbumSummary
		if err := rows.Scan(&a.IdentityURL, &a.IdentityName, &a.AlbumID, &a.AlbumVersion, &a.AlbumName, &a.ReleaseDate, &a.ImagesFront); err != nil {
			return nil, fmt.Errorf("scan album summary: %w", err)
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

// GetAlbum returns nil, nil when the album isn't indexed.
func (s *Store) GetAlbum(ctx context.Context, identityURL, albumID string) (*Album, error) {
	var a Album
	var albumPK int64
	row := s.pool.QueryRow(ctx, `
		SELECT id, identity_url, album_id, album_version, album_name, page_title, release_date,
		       images_front, images_back, images_insert, download_zip, credits, presentation
		FROM albums WHERE identity_url = $1 AND album_id = $2
	`, identityURL, albumID)
	err := row.Scan(&albumPK, &a.IdentityURL, &a.AlbumID, &a.AlbumVersion, &a.AlbumName, &a.PageTitle, &a.ReleaseDate,
		&a.ImagesFront, &a.ImagesBack, &a.ImagesInsert, &a.DownloadZip, &a.CreditsJSON, &a.PresentationJSON)
	if isNoRows(err) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get album %s/%s: %w", identityURL, albumID, err)
	}

	trackRows, err := s.pool.Query(ctx, `
		SELECT track_id, side, number, name, duration, file_url, lyrics
		FROM tracks WHERE album_pk = $1 ORDER BY position
	`, albumPK)
	if err != nil {
		return nil, fmt.Errorf("list tracks: %w", err)
	}
	defer trackRows.Close()
	for trackRows.Next() {
		var t Track
		if err := trackRows.Scan(&t.TrackID, &t.Side, &t.Number, &t.Name, &t.Duration, &t.File, &t.Lyrics); err != nil {
			return nil, fmt.Errorf("scan track: %w", err)
		}
		a.Tracks = append(a.Tracks, t)
	}
	if err := trackRows.Err(); err != nil {
		return nil, err
	}

	purchaseRows, err := s.pool.Query(ctx, `SELECT format, url FROM purchase_links WHERE album_pk = $1 ORDER BY id`, albumPK)
	if err != nil {
		return nil, fmt.Errorf("list purchase links: %w", err)
	}
	defer purchaseRows.Close()
	for purchaseRows.Next() {
		var p PurchaseLink
		if err := purchaseRows.Scan(&p.Format, &p.URL); err != nil {
			return nil, fmt.Errorf("scan purchase link: %w", err)
		}
		a.PurchaseLinks = append(a.PurchaseLinks, p)
	}
	if err := purchaseRows.Err(); err != nil {
		return nil, err
	}

	splitRows, err := s.pool.Query(ctx, `SELECT manifest_url, role, percentage FROM album_splits WHERE album_pk = $1 ORDER BY id`, albumPK)
	if err != nil {
		return nil, fmt.Errorf("list album splits: %w", err)
	}
	type rawSplit struct {
		manifestURL, role string
		percentage        float64
	}
	var rawSplits []rawSplit
	for splitRows.Next() {
		var sp rawSplit
		if err := splitRows.Scan(&sp.manifestURL, &sp.role, &sp.percentage); err != nil {
			splitRows.Close()
			return nil, fmt.Errorf("scan album split: %w", err)
		}
		rawSplits = append(rawSplits, sp)
	}
	splitErr := splitRows.Err()
	splitRows.Close()
	if splitErr != nil {
		return nil, splitErr
	}

	for _, sp := range rawSplits {
		status, err := s.verifySplit(ctx, sp.manifestURL, identityURL, a.AlbumID, a.AlbumVersion, sp.role, sp.percentage)
		if err != nil {
			return nil, err
		}
		a.Splits = append(a.Splits, AlbumSplit{ManifestURL: sp.manifestURL, Role: sp.role, Percentage: sp.percentage, Verified: status})
	}

	return &a, nil
}

// verifySplit checks whether confirmingURL's own manifest carries a
// contributions entry matching this exact (primaryURL, albumID,
// albumVersion, role, percentage) claim, per KB/0003-manifest.md's
// mutual-attestation rule for album splits.
func (s *Store) verifySplit(ctx context.Context, confirmingURL, primaryURL, albumID string, albumVersion int, role string, percentage float64) (VerificationStatus, error) {
	confirming, err := s.GetIdentity(ctx, confirmingURL)
	if err != nil {
		return "", err
	}
	if confirming == nil {
		return Pending, nil
	}

	var gotRole string
	var gotPct float64
	row := s.pool.QueryRow(ctx, `
		SELECT role, percentage FROM contributions
		WHERE identity_url = $1 AND manifest_url = $2 AND album_id = $3 AND album_version = $4
	`, confirmingURL, primaryURL, albumID, albumVersion)
	err = row.Scan(&gotRole, &gotPct)
	if isNoRows(err) {
		return Unverified, nil
	}
	if err != nil {
		return "", fmt.Errorf("check contribution: %w", err)
	}
	if gotRole == role && gotPct == percentage {
		return Verified, nil
	}
	return Disputed, nil
}
