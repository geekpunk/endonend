# MVP scope

```
Status: Proposed
Date: 2026-09-08
Type: Spec
References: [0001-purpose.md](./0001-purpose.md), [0002-architecture.md](./0002-architecture.md), [0003-manifest.md](./0003-manifest.md), [0004-endonend-artist-cli.md](./0004-endonend-artist-cli.md), [0006-container-infrastructure.md](./0006-container-infrastructure.md)
```

## Overview

[0002-architecture.md](./0002-architecture.md) already calls itself "the MVP architecture," but it describes the full system the protocol and platform are eventually meant to support, not the specific cut of it built first. This document draws that cut line: what actually gets built to validate the core mechanism with real artists, what stays deliberately deferred, and what "MVP validated" means. Today only `cli/` has real code; `api/`, `web/`, `mobile/`, `data/`, and `infra/` are empty placeholders. Everything below is what fills `api/` and `web/` first.

The bar is deliberately low: a minimal live instance, no polish, proving the mechanism end-to-end with a handful of real artists, not a production-grade launch.

## In scope

- **A crawler + GraphQL catalog API** (new `api/` Go module, per [0006-container-infrastructure.md](./0006-container-infrastructure.md)'s forward reference to where the backend lives). 0002 never pinned REST vs. GraphQL for this API; GraphQL is the MVP choice, see "API" below. The crawler fetches and verifies artist manifests on a simple interval and stores results in Postgres, reusing the shared validator in the `protocol/` module (see "Shared-validator module structure" below) rather than a second implementation.
- **An admin console**, gated by a simple shared-secret/basic-auth, not linked from any public listener-facing page. This is the concrete mechanism for onboarding artists at MVP scale (5-10, hand-curated by the founder), replacing the public, API-key-gated submission endpoint 0002 describes for later:
  - **By URL**: the same "submit for indexing" flow 0002 already specs (validate now, add to the crawler's poll list on success), just admin-gated instead of public/API-keyed. The public submission endpoint's API-key issuance and SSRF-hardening work stays deferred; the admin console gets the same immediate-feedback UX without needing either.
  - **By hand/upload**: paste raw manifest JSON or upload a `.json` file, run it through the same shared validator for a dry-run/inspect view. Useful before an artist's manifest is actually live at its claimed URL. This mode is for validation feedback; an identity only joins the crawler's poll list once its content is confirmed fetchable at its own `identity.url`, consistent with the protocol's "never trust, always verify at the source" rule.
- **A minimal web app** (new `web/`) with two views:
  - **A browse/list view**, built from scratch: lists indexed artists and albums, links to each album's playback page. No search engine; Meilisearch sync is out of scope (see below).
  - **A per-album playback page** that reuses the [geekpunk/albumn-page-generator](https://github.com/geekpunk/albumn-page-generator) template rather than being built from scratch. That repo is a small Vite + React app (`src/App.jsx`/`App.css`) that fetches an `album-config.json` at runtime and renders a full player: play/pause/seek, prev/next, art that flips front/back, per-track lyrics viewer, tracklist with per-song and full-ZIP downloads, and social links. Its config shape maps almost directly onto this project's own manifest fields (`albumName`/`artistName`/`pageTitle`/`images.front`+`images.back`/`colors`/`tracks[]`/`credits`/`links`/`footer`/`downloadZip`, per [0003-manifest.md](./0003-manifest.md)). `web/` vendors `App.jsx`/`App.css` and adds a small adapter mapping a manifest's `(identity, album)` pair into that same config shape, defaulting `colors` when `presentation.colors` is absent (the template requires that field). The existing component then renders directly against catalog-API data instead of a static per-repo JSON file. Per-album theming via `presentation.colors` is a side effect of this reuse, not separate build work, even though artist-facing storefront *generation* (0002's "build my site" item) stays out of scope.
- **Full protocol enforcement**, exactly as already committed to in 0002: signature verification, key pinning/rotation, hash-chained history verification, and label/split mutual-attestation display (verified / unverified / disputed). `merch` and `purchaseLinks` render as plain outbound links.
- **One `docker-compose.yml`**, per 0006's existing design, bringing up the entire MVP with a single `docker compose up`: a `backend` service (crawler + catalog API), a `web` service (both views above), and `postgres`. The Meilisearch service can be omitted or left unused since search isn't in MVP. The web app is a compose service like the others, not something deployed or hosted separately.

## Out of scope

Each of these is already speced or acknowledged elsewhere; this just marks it as deliberately deferred past MVP, not forgotten:

- **Search/discovery via Meilisearch** ([0002-architecture.md](./0002-architecture.md)'s Persistence layer section).
- **Native mobile apps** (0002's Languages and stack section). Web only for MVP.
- **Storefront generation** for artists to self-host their own branded site (0002's "build my site" open item). Distinct from the per-album theming above, which is in scope as a side effect of reusing the album-page-generator template.
- **Media edge cache and any listen/analytics signal**, including the play beacon (0002's Caching and Analytics sections). No listen counts appear anywhere in the MVP web app.
- **A public, self-serve submission endpoint** with API key issuance and SSRF hardening (0002's Protocol enforcement section). The admin console above covers "submit for indexing" for MVP, gated to the founder instead of open to any artist.
- **Signed multi-arch image publishing, GHCR, the Helm chart, and public GCP deployment** ([0006-container-infrastructure.md](./0006-container-infrastructure.md), 0002's "Initial deployment (GCP)"). MVP runs locally/self-hosted via `docker-compose` only.
- **Cross-instance federation** ([0009-federation.md](./0009-federation.md)) and **payments**, both already deferred to their own future work.

## Data model (Postgres)

Raw manifest JSON is preserved verbatim in a `jsonb` column per identity, for exact re-verification and audit. Fields that need querying or joining are pulled into relational tables/columns alongside it:

| Table | Key columns |
|---|---|
| `identities` | `url` (pk), `type`, `name`, `public_key` (currently pinned key), `contact_email`, `manifest_version`, `raw_manifest jsonb`, `fetched_at`, `ttl_seconds`, `history_head_hash`, and an artist's own nullable `affiliated_label_url`/`split_artist_pct`/`split_label_pct` |
| `label_rosters` | `label_url` (fk), `artist_url`, `split_artist_pct`, `split_label_pct`, one row per roster entry on a label manifest |
| `albums` | `id` (pk), `identity_url` (fk), `album_id`, `album_version`, `album_name`, `page_title`, `release_date`, `images_front`, `images_back`, `images_insert jsonb`, `download_zip`, `credits jsonb`, `presentation jsonb` |
| `tracks` | `id` (pk), `album_pk` (fk), `track_id`, `side`, `number`, `name`, `duration`, `file_url`, `lyrics` |
| `album_splits` | `album_pk` (fk), `manifest_url`, `role`, `percentage` |
| `contributions` | `identity_url` (fk, the confirming party), `manifest_url` (party being confirmed), `album_id`, `album_version`, `role`, `percentage` |
| `merch_links` | `identity_url` (fk), `label`, `url` |
| `purchase_links` | `album_pk` (fk), `format`, `url` |
| `history_entries` | `identity_url` (fk), `sequence_number`, `timestamp`, `type`, `data jsonb`, `previous_hash`, `signature jsonb` |

Verified/unverified/disputed status, for a `roster`/`affiliatedLabel` pair or an `album_splits`/`contributions` pair, is computed by joining these tables at query time (a materialized view is a later optimization, not an MVP requirement), never stored as a separately-trusted flag, so it can't drift from what the two source manifests actually say.

## API (GraphQL)

Resolves 0002's unspecified REST-vs-GraphQL choice for the catalog API. The public `Query` root serves the browse view and playback page; `Mutation` is admin-console-only and gated separately, never exposed to public listeners.

```graphql
enum IdentityType { ARTIST LABEL }
enum VerificationStatus { VERIFIED UNVERIFIED DISPUTED PENDING }

type Split { artist: Float! label: Float! }
type RosterEntry { artist: Identity! split: Split! verified: VerificationStatus! }
type AlbumSplit { manifestUrl: String! role: String! percentage: Float! verified: VerificationStatus! }
type PurchaseLink { format: String! url: String! }
type MerchLink { label: String! url: String! }
type Track { trackId: String! side: String number: String! name: String! duration: String! file: String! lyrics: String }
type Colors { primary: String heroBackground: String background: String text: String }
type Presentation { colors: Colors links: JSON footer: String }

type Identity {
  url: String!
  type: IdentityType!
  name: String!
  contactEmail: String!
  affiliatedLabel: Identity
  labelSplit: Split
  roster: [RosterEntry!]      # labels only
  merch: [MerchLink!]!
  albums: [Album!]!           # artists only
}

type Album {
  identity: Identity!
  albumId: String!
  albumVersion: Int!
  albumName: String!
  pageTitle: String
  releaseDate: String!
  imagesFront: String!
  imagesBack: String!
  imagesInsert: [String!]!
  downloadZip: String
  credits: JSON
  splits: [AlbumSplit!]!
  purchaseLinks: [PurchaseLink!]!
  presentation: Presentation
  tracks: [Track!]!
}

type Query {
  artists(limit: Int, offset: Int): [Identity!]!
  labels(limit: Int, offset: Int): [Identity!]!
  identity(url: String!): Identity
  albums(limit: Int, offset: Int): [Album!]!             # browse view
  album(identityUrl: String!, albumId: String!): Album   # playback page, feeds the album-page-generator adapter
}

# Admin-console-only.
type Mutation {
  submitManifestUrl(url: String!): SubmitResult!
  validateManifest(rawJson: String!): ValidationReport!   # dry-run, doesn't index
}

type SubmitResult { accepted: Boolean! errors: [String!]! identity: Identity }
type ValidationReport { valid: Boolean! errors: [ValidationError!]! }
type ValidationError { field: String! message: String! }
```

## Shared-validator module structure

0002 commits to "one shared backend validator (Go)" used by both the CLI and the crawler/API. Before this document, that commitment had a real gap: `cli/`'s validation packages lived under `cli/internal/...`, which Go's internal-package rule makes unimportable from a separate `api` module, the exact layout 0006 already forward-references for the backend.

Resolved: a new top-level Go module, **`protocol/`** (module `endonend/protocol`), holding the manifest/signing/canonical/validate/history packages as normal exported packages, matching 0002's own framing of "Protocol" as a distinct component from both the CLI tool and the platform backend. `cli/go.mod` now depends on it (`require endonend/protocol`, `replace endonend/protocol => ../protocol`); the future `api/go.mod` does the same. `cli/internal/{bandcamp,generate,slug,spinner}` stay CLI-only, since the crawler and catalog API never generate or sign manifests, only fetch and verify already-published ones.

## Validation threshold

Resolves the "MVP validation threshold" open question in [0001-purpose.md](./0001-purpose.md): 5-10 real artists actually publishing through the CLI and getting real listens through the MVP web app. Qualitative feedback (do artists find publishing workable, do listeners actually come back) matters more than the raw count; this is intentionally a small, informal bar, not a growth target.

## References

- [0001-purpose.md](./0001-purpose.md): the principles this scope must stay consistent with, and the "MVP validation threshold" open question this resolves.
- [0002-architecture.md](./0002-architecture.md): the full architecture this document narrows to an MVP cut.
- [0003-manifest.md](./0003-manifest.md): the manifest fields the data model and API are built from.
- [0004-endonend-artist-cli.md](./0004-endonend-artist-cli.md): the CLI artists already use to publish; unchanged by this document.
- [0006-container-infrastructure.md](./0006-container-infrastructure.md): the compose design the MVP's single `docker-compose.yml` follows.
- [KB/README.md](./README.md): KB conventions this document follows.
