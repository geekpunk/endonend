// Package store persists crawled manifests into Postgres and serves the
// read queries the GraphQL API needs, per KB/0010-mvp-scope.md's "Data
// model (Postgres)" section. Verification status (verified/unverified/
// disputed/pending) is always computed here, by joining what's actually
// stored, never trusted as a flag written once and left to drift.
package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"endonend/protocol/manifest"
)

type Store struct {
	pool *pgxpool.Pool
}

func New(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool}
}

// VerificationStatus mirrors the GraphQL enum of the same name in
// KB/0010-mvp-scope.md's API section.
type VerificationStatus string

const (
	Verified   VerificationStatus = "VERIFIED"
	Unverified VerificationStatus = "UNVERIFIED"
	Disputed   VerificationStatus = "DISPUTED"
	Pending    VerificationStatus = "PENDING"
)

// UpsertManifest replaces everything derived from one identity's manifest:
// the identity row itself, and every album/track/split/roster/contribution/
// merch-link/history-entry row that came from it. A crawl always reflects
// the manifest's current, complete state, so this is delete-and-reinsert
// per identity rather than an incremental diff; the CLI already owns
// diffing for history-entry generation, the server just mirrors what the
// manifest and history.json say.
func (s *Store) UpsertManifest(ctx context.Context, m *manifest.Manifest, rawManifest []byte, history []manifest.HistoryEntry, fetchedAt time.Time) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin upsert: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	url := m.Identity.URL
	var affiliatedLabelURL string
	var splitArtist, splitLabel *float64
	if m.Label != nil {
		affiliatedLabelURL = m.Label.AffiliatedLabel
		if m.Label.Split != nil {
			a, l := m.Label.Split.Artist, m.Label.Split.Label
			splitArtist, splitLabel = &a, &l
		}
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO identities (url, type, name, public_key, contact_email, manifest_version, raw_manifest, fetched_at, ttl_seconds, history_head_hash, affiliated_label_url, split_artist_pct, split_label_pct, last_error)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, '')
		ON CONFLICT (url) DO UPDATE SET
			type = EXCLUDED.type, name = EXCLUDED.name, public_key = EXCLUDED.public_key,
			contact_email = EXCLUDED.contact_email, manifest_version = EXCLUDED.manifest_version,
			raw_manifest = EXCLUDED.raw_manifest, fetched_at = EXCLUDED.fetched_at,
			ttl_seconds = EXCLUDED.ttl_seconds, history_head_hash = EXCLUDED.history_head_hash,
			affiliated_label_url = EXCLUDED.affiliated_label_url,
			split_artist_pct = EXCLUDED.split_artist_pct, split_label_pct = EXCLUDED.split_label_pct,
			last_error = ''
	`, url, m.Identity.Type, m.Identity.Name, m.Identity.PublicKey, m.Identity.ContactEmail,
		m.ManifestVersion, rawManifest, fetchedAt, m.Refresh.TTLSeconds, m.History.HeadHash,
		affiliatedLabelURL, splitArtist, splitLabel)
	if err != nil {
		return fmt.Errorf("upsert identity: %w", err)
	}

	if _, err := tx.Exec(ctx, `DELETE FROM label_rosters WHERE label_url = $1`, url); err != nil {
		return fmt.Errorf("clear roster: %w", err)
	}
	if m.Label != nil {
		for _, r := range m.Label.Roster {
			_, err := tx.Exec(ctx, `
				INSERT INTO label_rosters (label_url, artist_url, split_artist_pct, split_label_pct)
				VALUES ($1, $2, $3, $4)
			`, url, r.ArtistManifestURL, r.Split.Artist, r.Split.Label)
			if err != nil {
				return fmt.Errorf("insert roster entry: %w", err)
			}
		}
	}

	if _, err := tx.Exec(ctx, `DELETE FROM contributions WHERE identity_url = $1`, url); err != nil {
		return fmt.Errorf("clear contributions: %w", err)
	}
	for _, c := range m.Contributions {
		_, err := tx.Exec(ctx, `
			INSERT INTO contributions (identity_url, manifest_url, album_id, album_version, role, percentage)
			VALUES ($1, $2, $3, $4, $5, $6)
		`, url, c.ManifestURL, c.AlbumID, c.AlbumVersion, c.Role, c.Percentage)
		if err != nil {
			return fmt.Errorf("insert contribution: %w", err)
		}
	}

	if _, err := tx.Exec(ctx, `DELETE FROM merch_links WHERE identity_url = $1`, url); err != nil {
		return fmt.Errorf("clear merch links: %w", err)
	}
	for _, mk := range m.Merch {
		if _, err := tx.Exec(ctx, `INSERT INTO merch_links (identity_url, label, url) VALUES ($1, $2, $3)`, url, mk.Label, mk.URL); err != nil {
			return fmt.Errorf("insert merch link: %w", err)
		}
	}

	if _, err := tx.Exec(ctx, `DELETE FROM history_entries WHERE identity_url = $1`, url); err != nil {
		return fmt.Errorf("clear history: %w", err)
	}
	for i, h := range history {
		_, err := tx.Exec(ctx, `
			INSERT INTO history_entries (identity_url, sequence_number, timestamp, type, data, previous_hash, signature)
			VALUES ($1, $2, $3, $4, $5, $6, $7)
		`, url, i, h.Timestamp, h.Type, h.Data, h.PreviousHash, h.Signature)
		if err != nil {
			return fmt.Errorf("insert history entry: %w", err)
		}
	}

	// albums cascades to tracks, album_splits, and purchase_links.
	if _, err := tx.Exec(ctx, `DELETE FROM albums WHERE identity_url = $1`, url); err != nil {
		return fmt.Errorf("clear albums: %w", err)
	}
	for _, a := range m.Catalog {
		var credits, presentation []byte
		if a.Credits != nil {
			credits, err = marshalJSON(a.Credits)
			if err != nil {
				return err
			}
		}
		if a.Presentation != nil {
			presentation, err = marshalJSON(a.Presentation)
			if err != nil {
				return err
			}
		}
		insertJSON, err := marshalJSON(a.Images.Insert)
		if err != nil {
			return err
		}

		var albumPK int64
		err = tx.QueryRow(ctx, `
			INSERT INTO albums (identity_url, album_id, album_version, album_name, page_title, release_date, images_front, images_back, images_insert, download_zip, credits, presentation)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
			RETURNING id
		`, url, a.AlbumID, a.AlbumVersion, a.AlbumName, a.PageTitle, a.ReleaseDate, a.Images.Front, a.Images.Back, insertJSON, a.DownloadZip, credits, presentation).Scan(&albumPK)
		if err != nil {
			return fmt.Errorf("insert album %s: %w", a.AlbumID, err)
		}

		for i, t := range a.Tracks {
			_, err := tx.Exec(ctx, `
				INSERT INTO tracks (album_pk, position, track_id, side, number, name, duration, file_url, lyrics)
				VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
			`, albumPK, i, t.TrackID, t.Side, t.Number, t.Name, t.Duration, t.File, t.Lyrics)
			if err != nil {
				return fmt.Errorf("insert track %s: %w", t.TrackID, err)
			}
		}
		for _, sp := range a.Splits {
			_, err := tx.Exec(ctx, `
				INSERT INTO album_splits (album_pk, manifest_url, role, percentage)
				VALUES ($1, $2, $3, $4)
			`, albumPK, sp.ManifestURL, sp.Role, sp.Percentage)
			if err != nil {
				return fmt.Errorf("insert album split: %w", err)
			}
		}
		for _, pl := range a.PurchaseLinks {
			_, err := tx.Exec(ctx, `
				INSERT INTO purchase_links (album_pk, format, url)
				VALUES ($1, $2, $3)
			`, albumPK, pl.Format, pl.URL)
			if err != nil {
				return fmt.Errorf("insert purchase link: %w", err)
			}
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit upsert: %w", err)
	}
	return nil
}

// RecordFetchError leaves an identity's existing data in place (it stays
// indexed and visible) but records why the most recent crawl attempt
// failed, for the admin console to surface.
func (s *Store) RecordFetchError(ctx context.Context, url string, fetchErr error) error {
	_, err := s.pool.Exec(ctx, `UPDATE identities SET last_error = $2 WHERE url = $1`, url, fetchErr.Error())
	return err
}

var ErrNotFound = errors.New("not found")

func marshalJSON(v any) ([]byte, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return nil, fmt.Errorf("marshal %T: %w", v, err)
	}
	return b, nil
}

// isNoRows reports whether err is the "no rows" sentinel pgx returns from
// QueryRow, so callers can turn it into the package's own ErrNotFound.
func isNoRows(err error) bool {
	return errors.Is(err, pgx.ErrNoRows)
}
