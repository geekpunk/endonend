// Package crawler fetches and verifies artist/label manifests and stores
// the result, per KB/0002-architecture.md's Protocol enforcement section
// and KB/0010-mvp-scope.md's MVP scope.
package crawler

import (
	"context"
	"fmt"
	"time"

	"endonend/api/internal/store"
	"endonend/protocol/validate"
)

type Crawler struct {
	store *store.Store
	opts  validate.Options
	now   func() time.Time
}

func New(s *store.Store) *Crawler {
	return &Crawler{store: s, now: time.Now}
}

// PollOne fetches url and, if it passes structural, signature, and
// history-chain validation, indexes it. Deliberately not a Deep validate:
// mutual-attestation status (verified/unverified/disputed) for label
// affiliation and album splits is computed by the store's own read queries
// by joining against whatever's already been crawled, per
// KB/0010-mvp-scope.md's data model, rather than by the validator's live
// network cross-checks. That keeps a single unconfirmed attestation from
// making an otherwise-valid manifest fail to index at all.
func (c *Crawler) PollOne(ctx context.Context, url string) error {
	report, err := validate.ValidateURL(url, c.opts)
	if err != nil {
		// The manifest didn't even fetch, so there's no identity.url to
		// attach this failure to; RecordFetchError is a no-op against an
		// identity that was never successfully indexed in the first place.
		_ = c.store.RecordFetchError(ctx, url, err)
		return fmt.Errorf("fetch %s: %w", url, err)
	}
	if !report.Valid {
		valErr := fmt.Errorf("manifest failed validation: %s", report.Summary())
		// Record against the identity's own URL, not the fetch target,
		// since a re-crawl's failure needs to land on the same row the
		// last successful crawl created (identities.url = identity.url,
		// which can differ from the fetch URL's well-known suffix).
		if report.Manifest != nil && report.Manifest.Identity.URL != "" {
			_ = c.store.RecordFetchError(ctx, report.Manifest.Identity.URL, valErr)
		} else {
			_ = c.store.RecordFetchError(ctx, url, valErr)
		}
		return valErr
	}
	if err := c.store.UpsertManifest(ctx, report.Manifest, report.RawJSON, report.History, c.now()); err != nil {
		return fmt.Errorf("index %s: %w", url, err)
	}
	return nil
}
