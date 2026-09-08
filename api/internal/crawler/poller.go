package crawler

import (
	"context"
	"time"
)

// PollDue polls every identity whose self-declared refresh TTL has
// elapsed. A failure on one identity is reported to onError and does not
// stop the rest from being polled.
func (c *Crawler) PollDue(ctx context.Context, onError func(url string, err error)) {
	due, err := c.storeDueIdentities(ctx)
	if err != nil {
		onError("", err)
		return
	}
	for _, identityURL := range due {
		// DueIdentities returns identity.url, the bare base URL per
		// KB/0003-manifest.md, not the manifest's actual fetch location.
		// Reconstruct the fixed well-known path PollOne needs to fetch.
		manifestURL := identityURL + "/.well-known/endonend/manifest.json"
		if err := c.PollOne(ctx, manifestURL); err != nil {
			onError(identityURL, err)
		}
	}
}

// RunLoop calls PollDue immediately, then again every checkInterval, until
// ctx is cancelled. checkInterval governs how often the poller wakes up to
// check what's due, not the per-identity refresh cadence itself, which is
// TTL-driven per KB/0002-architecture.md's Protocol enforcement section.
func (c *Crawler) RunLoop(ctx context.Context, checkInterval time.Duration, onError func(url string, err error)) {
	c.PollDue(ctx, onError)
	ticker := time.NewTicker(checkInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			c.PollDue(ctx, onError)
		}
	}
}

func (c *Crawler) storeDueIdentities(ctx context.Context) ([]string, error) {
	return c.store.DueIdentities(ctx, c.now())
}
