# Bandcamp import

```
Status: Proposed
Date: 2026-09-08
Type: Spec
References: [0001-purpose.md](./0001-purpose.md), [0003-manifest.md](./0003-manifest.md), [0004-endonend-artist-cli.md](./0004-endonend-artist-cli.md)
```

## Overview

An artist migrating off Bandcamp already has their catalog metadata (album title, tracklist, order, durations, release date, cover art) sitting on a page they control the content of, just not the hosting of. `endonend-artist-cli import bandcamp <album-url>` reads that page and uses it to create or update `endonend.source.json`, the same source file [0004-endonend-artist-cli.md](./0004-endonend-artist-cli.md)'s `generate` command already reads, instead of making the artist retype every track title and duration by hand.

## What gets imported, and what doesn't

Bandcamp's album page embeds a `data-tralbum` JSON blob (title, release date, and a `trackinfo` array with title, track number, duration, and a signed streaming URL per track) plus an `og:image` cover URL. This import command reads:

- Artist/band name (used as `identity.name` only when creating a fresh source file; an existing one is left alone).
- Album title, release date, track titles, track order, and track durations.
- The cover art URL, for downloading a local copy.
- Each track's streaming URL, for downloading a local copy of the audio.

It deliberately does **not** write any `bandcamp.com`/`bcbits.com` URL into `endonend.source.json`. Two reasons, both tracing back to existing decisions rather than new ones:

- **Artist-owned infrastructure** ([0001-purpose.md](./0001-purpose.md)): a manifest's `catalog[].images` and `catalog[].tracks[].file` are supposed to point at files the artist hosts, not a hotlink to a platform this project exists as an alternative to.
- **The URLs don't last.** Bandcamp's per-track streaming links are signed with a short-lived token; a manifest built from them would validate today and silently break once they expire, exactly the kind of failure [0004](./0004-endonend-artist-cli.md)'s `validate` command exists to catch, not cause.

Instead, the command downloads the cover art and every track's audio (the streamed encode, typically 128kbps; not the artist's original masters, which Bandcamp doesn't expose) into a local directory, and fills `endonend.source.json` with the URLs those files *should* end up at once the artist uploads them to their own `identity.url`, following the same `<url>/albums/<albumId>/...` layout [0003-manifest.md](./0003-manifest.md)'s own example manifest uses. The command's final output tells the artist exactly what was downloaded and where it still needs to be uploaded before `generate` produces a manifest that will actually resolve.

## Command

```
endonend-artist-cli import bandcamp [flags] <bandcamp-album-url>
```

Flags precede the positional URL (standard Go `flag` package behavior):

- `--url` (required unless `endonend.source.json` already exists): the artist's own `identity.url`.
- `--contact-email` (required unless `endonend.source.json` already exists): `identity.contactEmail`.
- `--type`: `"artist"` or `"label"` (default `"artist"`).
- `--source`: path to create or update (default `endonend.source.json`, matching `generate`).
- `--download-dir`: where to save cover art and audio (default `./bandcamp-import`).
- `--skip-download`: prefill metadata and placeholder URLs only, without fetching any files.

If `--source` already exists, its `identity` is left as-is (an explicit `--url`/`--contact-email` still overrides it) and the imported album is upserted into its `catalog` by `albumId`, matching the same "add an album" pattern the interactive menu uses, rather than overwriting the whole file.

`albumId` and `trackId` are derived by slugifying the album/track titles (e.g. `demo`, `demo-t1`); like every other source field, an artist can hand-edit these afterward, since [0004](./0004-endonend-artist-cli.md) already treats `endonend.source.json` as human-editable.

## Interactive menu

The same behavior is also reachable as option 4, "Import an album from Bandcamp," on [0004](./0004-endonend-artist-cli.md)'s top-level menu, consistent with that menu being the primary interface. It asks for the Bandcamp album URL, then, only if `endonend.source.json` doesn't exist yet, the same artist/label/URL/contact-email questions "Create or update your manifest" asks; an existing source file's identity is left untouched. It finishes by asking whether to download the cover art and audio locally now (default yes), then upserts the album exactly as the flag-driven command does.

## Non-goals

- No re-encoding or quality upscaling of the downloaded audio; it's exactly what Bandcamp streams.
- No import of `label`, `splits`, `credits`, `purchaseLinks`, or `presentation`; Bandcamp's page doesn't carry this project's split/attestation model, so those fields are left for the artist to fill in by hand, the same "advanced fields are hand-edited" scope 0004 already draws for the interactive menu.
- No automatic upload to the artist's host. Publishing still means the artist puts files on infrastructure they control; this command only prepares them.

## References

- [0001-purpose.md](./0001-purpose.md): artist-owned infrastructure, the reason imported files are downloaded rather than hotlinked.
- [0003-manifest.md](./0003-manifest.md): the manifest fields this command populates, and the file-layout convention its placeholder URLs follow.
- [0004-endonend-artist-cli.md](./0004-endonend-artist-cli.md): `endonend.source.json` and `generate`, which this command is a new way to bootstrap.
