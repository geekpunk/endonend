# Manifest

```
Status: Proposed
Date: 2026-09-07
Type: Spec
References: [0001-purpose.md](./0001-purpose.md), [0002-architecture.md](./0002-architecture.md)
```

## Overview

This document defines the concrete file format for the signed manifest every artist and label publishes, the file referenced throughout [0002-architecture.md](./0002-architecture.md)'s Protocol enforcement section. It is a single JSON file that serves two purposes at once: it is the protocol's source of truth for identity, catalog, splits, and history, and it is the input the optional generated storefront (see 0002's Languages and stack) renders into a static site. One file, not two formats to keep in sync.

Format: JSON, UTF-8, published as a static file at a well-known path relative to a URL the artist or label controls:

```
<identity.url>/.well-known/endonend/manifest.json
```

with its paired history log at:

```
<identity.url>/.well-known/endonend/history.json
```

This is deliberately a URL, not a bare domain: not every artist owns a domain, but many can still control a specific path, a GitHub Pages project URL, for example, even on a host they don't own outright. Identity is anchored to whatever URL prefix they actually control, not to domain registration.

## Design goals

Every top-level section exists because a specific principle or architecture decision requires it, not by default:

- `identity`: the "verifiable artists and labels, open history" principle in 0001, and the DID:web-style verification model in 0002, generalized from a domain to a URL so artists without their own domain can still participate.
- `refresh`: the TTL-driven crawler refresh model in 0002.
- `identity.contactEmail`: the failure-notification mechanism in 0002.
- `catalog`: the artist-owned infrastructure principle in 0001, expressed as static files rather than a service.
- `label`: the "no legacy publisher/distributor layer" and radical transparency principles in 0001.
- `catalog[].splits` and `contributions`: the same radical transparency principle, extended past the default artist/label split to per-release collaborations, features, and production credits, which a single catalog-wide split can't represent honestly.
- `history`: the same open-history principle as `identity`, made concrete as an append-only log, including its `key_rotated` entries, which exist so a URL takeover can't silently become an identity takeover.
- `beacon`: the analytics delegation model in 0002.
- `merch` and `catalog[].purchaseLinks`: the "direct-to-artist" mission in 0001, the system never sits in the artist's revenue path, so these are plain outbound links to a storefront the artist already controls, not a payments or fulfillment feature of the platform itself.
- `presentation` (manifest-wide and per-album): the optional generated storefront capability in 0002's Languages and stack. The per-album override exists so a single-album page, the common case, matches its own art and identity rather than inheriting an artist-wide default.

Nothing here exists purely for developer convenience; a field that doesn't trace back to one of the above shouldn't be in v1.

## Top-level structure

| Field             | Type   | Required              | Description                                                                                   |
| ----------------- | ------ | --------------------- | --------------------------------------------------------------------------------------------- |
| `manifestVersion` | string | Yes                   | Schema version, e.g. `"1.0"`. See Versioning below.                                           |
| `identity`        | object | Yes                   | Who publishes this manifest.                                                                  |
| `refresh`         | object | Yes                   | How often the crawler should re-check this manifest.                                          |
| `label`           | object | No                    | Label affiliation and split (artist manifests) or roster (label manifests).                   |
| `contributions`   | array  | No                    | Releases by other parties this identity confirms contributing to, attesting to its own share. |
| `beacon`          | object | No                    | Optional play-event notification endpoint.                                                    |
| `merch`           | array  | No                    | General merch links, not tied to one album.                                                   |
| `history`         | object | Yes                   | Reference to the append-only history log.                                                     |
| `presentation`    | object | No                    | Storefront theming and links, only needed if generating a storefront.                         |
| `catalog`         | array  | Artist manifests only | Albums and tracks. Omitted or empty on label manifests.                                       |
| `signature`       | object | Yes                   | Detached signature over the rest of the manifest.                                             |

### `identity`

| Field          | Type   | Required | Description                                                                                        |
| -------------- | ------ | -------- | -------------------------------------------------------------------------------------------------- |
| `type`         | string | Yes      | `"artist"` or `"label"`.                                                                           |
| `name`         | string | Yes      | Display name.                                                                                      |
| `url`          | string (URL) | Yes | The base URL this identity publishes under, e.g. `"https://ligatures.example"` or `"https://someartist.github.io/bandname"`. The manifest must actually live at `<url>/.well-known/endonend/manifest.json`. |
| `publicKey`    | string | Yes      | E.g. `"ed25519:<base64>"`. The key whose private half signs this manifest and every history entry. |
| `contactEmail` | string | Yes      | Where validation-failure notifications (see 0002) are sent.                                        |

### `refresh`

| Field | Type | Required | Description |
|---|---|---|---|
| `ttlSeconds` | integer | Yes | How often the crawler should re-check this manifest. Clamped by the platform to a min/max (exact bounds are an open question in 0002). |

### `label`

Shape depends on `identity.type`.

On an **artist** manifest, declares an optional affiliation:

| Field | Type | Required | Description |
|---|---|---|---|
| `affiliatedLabel` | string (URL) | No | The label's manifest URL. Absent means fully independent. |
| `split` | object | If `affiliatedLabel` is present | `{ "artist": number, "label": number }`, percentages summing to 100. |

On a **label** manifest, declares the roster it claims:

| Field | Type | Required | Description |
|---|---|---|---|
| `roster` | array | No | List of `{ "artistManifestUrl": string, "split": { "artist": number, "label": number } }`. |

**Mutual attestation.** An affiliation is only shown as verified when both sides agree: the artist's manifest names the label and a split in its own `label` object, and the label's own manifest lists the same artist and the same split in its `roster`. A one-sided claim (an artist naming a label that doesn't list them back, or the reverse) is not hidden, it's surfaced as unverified or disputed, consistent with radical transparency: show the disagreement rather than silently resolve it in either party's favor. (This differs from the `contributions` mechanism below, which exists for parties, like a featured artist or producer, who have no roster of their own to list a claim in.)

### `contributions`

The reciprocal of both label roster entries and album-level `splits`: how a party actually confirms a split someone else declared about them. Each entry:

| Field | Type | Required | Description |
|---|---|---|---|
| `manifestUrl` | string (URL) | Yes | The manifest of the party whose catalog this release lives in (the primary artist or label). |
| `albumId` | string | Yes | The `albumId` from that party's `catalog`. |
| `albumVersion` | integer | Yes | The specific revision this confirmation applies to. |
| `role` | string | Yes | Must match the role that manifest's `splits` entry lists for this identity. |
| `percentage` | number | Yes | Must match the percentage that manifest's `splits` entry lists for this identity. |

A crawler considers an album-level split entry verified only when it finds a matching `contributions` entry (same `manifestUrl`, `albumId`, `albumVersion`, role, and percentage) in that contributing party's own manifest. This is the mechanism, not just the intent, behind the mutual-attestation rule for album `splits` described under `catalog` below. Label affiliation uses its own dedicated `roster` field instead, since a label already has a natural place on its own manifest to list every artist it claims.

### `beacon`

| Field | Type | Required | Description |
|---|---|---|---|
| `url` | string (URL) | No | Endpoint clients ping on play start, per the Analytics section of 0002. |

### `merch`

Array of general merch links, not tied to any one album, for things like apparel or other non-music items sold outside the catalog.

| Field | Type | Required | Description |
|---|---|---|---|
| `label` | string | Yes | Display label, e.g. `"Official Store"`. |
| `url` | string (URL) | Yes | Where it sends the listener. The platform never handles the purchase itself. |

### `history`

| Field | Type | Required | Description |
|---|---|---|---|
| `url` | string (URL) | Yes | Location of the history log file. |
| `headHash` | string | No | Hash of the latest history entry, so the crawler can detect a changed log without re-verifying the whole chain every cycle. |

### `presentation`

Only needed if the artist or label wants a generated storefront.

| Field | Type | Required | Description |
|---|---|---|---|
| `colors` | object | No | `{ "primary", "heroBackground", "background", "text" }`, all hex color strings. |
| `links` | object | No | Social links, keyed by platform (e.g. `bandcamp`, `instagram`, `github`, `spotify`, `youtube`, `website`). Unrecognized keys are ignored by renderers rather than rejected, so new platforms can be added without a schema bump. |
| `footer` | string | No | Free-text footer content. |

These are manifest-wide defaults. Any album can override some or all of them for its own generated page, see `catalog[].presentation` below, since a single-album storefront (the common case) benefits from its own colors, links, and footer rather than always inheriting the artist's manifest-wide ones.

### `catalog`

Array of album objects. Required (non-empty) on artist manifests; omitted on label manifests, since labels don't host tracks themselves, each affiliated artist hosts their own.

**Album object:**

| Field | Type | Required | Description |
|---|---|---|---|
| `albumId` | string | Yes | Stable identifier, referenced by history entries. Never changes across revisions of the same release. |
| `albumVersion` | integer | Yes | Starts at `1`; incremented every time this album's content changes (tracklist, metadata, files). Lets clients and caches detect a specific revision without depending on the manifest-level poll cycle, and lets history entries reference exactly which revision they describe. |
| `albumName` | string | Yes | |
| `pageTitle` | string | No | Defaults to `"{identity.name} - {albumName}"`. |
| `releaseDate` | string | Yes | ISO 8601 date, used for sorting and discovery. |
| `images` | object | Yes | Cover art. See below. |
| `downloadZip` | string (URL) | No | Full-album archive download. |
| `credits` | object | No | `{ "paragraphs": [...] }`. Each paragraph is either a plain string, or an array of segments `{ "text": string, "url": string (optional) }` for inline links. |
| `splits` | array | No | Per-release split, for collaborations, features, or compilations that differ from the artist's default label split. See below. Absent means the top-level `label.split` (or 100% to the artist, if independent) applies. |
| `purchaseLinks` | array | No | Where to buy this release as a physical product. See below. |
| `presentation` | object | No | Same shape as the top-level `presentation` (`colors`, `links`, `footer`). Any field present here overrides the manifest-wide default for this album's own generated page; any field absent falls back to the manifest-wide value. |
| `tracks` | array | Yes | See below. |

**Album `images`:**

| Field | Type | Required | Description |
|---|---|---|---|
| `front` | string (URL) | Yes | Front cover. |
| `back` | string (URL) | Yes | Back cover. |
| `insert` | array of string (URL) | No | Insert or booklet pages (liner notes, lyric sheets, a poster, and so on). Defaults to an empty array; not every release has one. |

**Album `splits` entry**, one per contributing party:

| Field | Type | Required | Description |
|---|---|---|---|
| `manifestUrl` | string (URL) | Yes | The contributing artist's or label's own manifest URL. |
| `role` | string | Yes | E.g. `"primary"`, `"feature"`, `"producer"`, `"label"`. |
| `percentage` | number | Yes | This party's share. All entries in the array sum to 100. |

**Album `purchaseLinks` entry**, one per physical format offered:

| Field | Type | Required | Description |
|---|---|---|---|
| `format` | string | Yes | E.g. `"vinyl"`, `"cd"`, `"cassette"`, `"bundle"`. |
| `url` | string (URL) | Yes | Where to buy it. Fulfillment and payment are entirely the artist's or label's own, the platform is never in that path. |

Album-level splits are verified the same way label affiliation is, in spirit: a split is shown as verified only when every named party confirms it independently. Concretely, each party listed in `splits` needs a matching entry in their own manifest's `contributions` (same `manifestUrl`, `albumId`, `albumVersion`, role, and percentage). Any party who hasn't confirmed, or who lists a different number, shows as unverified or disputed rather than silently trusted.

**Track object:**

| Field | Type | Required | Description |
|---|---|---|---|
| `trackId` | string | Yes | Stable identifier, referenced by history entries. |
| `side` | string | No | Any label (`"A"`, `"B"`, ...); renderers group by it. |
| `number` | string | Yes | Display label, e.g. `"A1"`. |
| `name` | string | Yes | Track title. |
| `duration` | string | Yes | Display hint only, e.g. `"3'30\""`; not authoritative, clients read actual duration from the audio file. |
| `file` | string (URL) | Yes | Absolute, or relative to the manifest's own location. |
| `lyrics` | string | No | `\n`-separated. Omit or `null` to hide the lyrics view for that track. |

### `signature`

| Field | Type | Required | Description |
|---|---|---|---|
| `algorithm` | string | Yes | Must be exactly `"ed25519"` for v1. See allowlist note below. |
| `value` | string | Yes | Base64 signature over the canonicalized manifest with this `signature` object excluded. |

Verifying the signature requires a deterministic serialization of the rest of the manifest, the same bytes every time, on every implementation, so the Go backend validator and the Kotlin Multiplatform mobile module (see 0002) compute the same thing to check. Canonicalization uses **RFC 8785, the JSON Canonicalization Scheme (JCS)**: a standard, already-specified algorithm (sorted object keys, a fixed number and string representation, no insignificant whitespace), so there's no bespoke canonicalization format for either implementation to get subtly wrong. Both the Go and Kotlin Multiplatform validators should use an existing, well-tested JCS library in their respective ecosystems rather than a hand-rolled implementation.

**Algorithm is an allowlist, not free text.** Unlike an unrecognized optional field, which the Versioning section below says a validator should tolerate, an unrecognized `signature.algorithm` (or a history entry's `signature.algorithm`) is always rejected outright, on any manifest version. Algorithm choice is a security decision, not a convenience one: silently accepting whatever string shows up risks an algorithm-confusion or downgrade attack, a manifest claiming a weak or meaningless scheme (or literally `"none"`) that a lenient validator waves through. `"ed25519"` is the only accepted value for v1; adding a second algorithm later is a deliberate, explicit change to this spec, not something a validator infers on its own.

## History log format

A separate static JSON file at `history.url`: an array of entries, append-only, never rewritten. `release_added` and `release_removed` cover an album appearing or disappearing from the catalog; `release_updated` covers an album that stays present but whose content changed (a new track, corrected metadata), the case those two don't.

| Field | Type | Description |
|---|---|---|
| `timestamp` | string | ISO 8601. |
| `type` | string | `"identity_updated"`, `"split_changed"`, `"release_added"`, `"release_updated"`, `"release_removed"`, or `"key_rotated"`. |
| `data` | object | Type-specific payload (for example, `split_changed` carries the old and new split; `release_updated` carries `{ "albumId": string, "oldAlbumVersion": integer, "newAlbumVersion": integer }`; `key_rotated` carries `{ "oldPublicKey": string, "newPublicKey": string }`). |
| `previousHash` | string | Hash of the prior entry, chaining the log. Empty or absent on the first (genesis) entry. |
| `signature` | object | Same shape as the manifest's `signature`. Signed by `identity.publicKey`, **except** a `key_rotated` entry, which is signed by the *old* key it's retiring. See Key rotation below. |

A valid log verifies as an unbroken chain from genesis (or from a previously cached `headHash`) to the `headHash` declared in the current manifest. A gap or a hash mismatch means the log itself failed verification, handled the same way any other validation failure is: rejected or flagged, and the contact email is notified per 0002.

### Key rotation

A domain or URL can change hands. Without something to prevent it, whoever controls the URL next could publish a new `identity.publicKey` and silently take over an artist's history and reputation. Rather than trusting whatever key shows up, verification pins a key over time and only accepts a change when the *previous* key vouches for it:

- The first time a crawler (or the CLI's own validator) sees a manifest at a given `identity.url`, it trusts and pins whatever key is there. This is an accepted trust-on-first-use bootstrap, the same tradeoff SSH host keys make.
- On every later check, if `identity.publicKey` still matches the pinned key, verification proceeds as normal.
- If `identity.publicKey` has changed, the change is only accepted when the history log contains a `key_rotated` entry whose `data.oldPublicKey` matches the pinned key, whose `data.newPublicKey` matches the new key, and whose `signature` verifies against the *old* key, proof that whoever controlled the previous key authorized this specific handoff, not just that a new key showed up. Once accepted, the pin updates to the new key.
- A key change with no matching, properly signed `key_rotated` entry is treated as a possible hijack: rejected, and a failure notification (per 0002) goes to the **last known good** contact email, the one associated with the previously pinned key, not whatever `identity.contactEmail` the new, unverified manifest claims, since that field could itself be part of the hijack.
- If the old private key is genuinely lost rather than just changed, no valid rotation is possible by construction. That mirrors losing an SSH or PGP key: there is no recovery path that doesn't involve some out-of-band trust decision, which this protocol deliberately does not try to solve.

Example `key_rotated` entry, signed by the key it retires:

```json
{
  "timestamp": "2025-11-02T00:00:00Z",
  "type": "key_rotated",
  "data": {
    "oldPublicKey": "ed25519:AbCdEf1234567890...",
    "newPublicKey": "ed25519:QrStUv9876543210..."
  },
  "previousHash": "sha256:7c6d5e...",
  "signature": {
    "algorithm": "ed25519",
    "value": "base64-signature-computed-with-the-OLD-private-key"
  }
}
```

## Versioning and forward compatibility

`manifestVersion` is a `"major.minor"` string. A validator rejects a manifest whose major version it doesn't understand, but tolerates and preserves fields it doesn't recognize within a known major version. This lets an additive minor change (a new optional field) reach the ecosystem without every validator needing to upgrade in lockstep, while a breaking change still requires a major version bump that older validators correctly refuse.

## Conformance test vectors

Schema drift between the Go backend validator and the Kotlin Multiplatform mobile module is already a tracked concern (see 0002); canonicalization and signing need the same guardrail, since a subtle bug there doesn't fail loudly, it just makes one implementation quietly reject manifests the other accepts. The repository maintains a small, fixed set of test vectors, checked in alongside the code, that both implementations run against in CI:

- A sample unsigned manifest (the full JSON content minus `signature`).
- Its expected canonical JCS byte output for that exact content.
- A fixed test keypair, clearly labeled as test-only in the repository and never treated as a trusted identity by either validator.
- The expected valid signature over that canonical output, produced with the test private key.
- At least one deliberately invalid case per rule in Validation rules below (a bad signature, a disallowed `signature.algorithm`, a broken history hash chain, and so on), so both implementations are proven to reject the same things, not just accept the same things.

Both the Go and Kotlin Multiplatform test suites load these vectors and assert byte-for-byte and signature-verification agreement. This runs as part of the CI/CD pipeline described in 0002, so a canonicalization or signing divergence is caught there, not discovered later as a confusing cross-platform verification failure.

## Example manifest

```json
{
  "manifestVersion": "1.0",
  "identity": {
    "type": "artist",
    "name": "Ligatures",
    "url": "https://ligatures.example",
    "publicKey": "ed25519:AbCdEf1234567890...",
    "contactEmail": "band@ligatures.example"
  },
  "refresh": {
    "ttlSeconds": 21600
  },
  "label": {
    "affiliatedLabel": "https://smalllabel.example/.well-known/endonend/manifest.json",
    "split": { "artist": 85, "label": 15 }
  },
  "beacon": {
    "url": "https://ligatures.example/plays"
  },
  "merch": [
    { "label": "Official Store", "url": "https://ligatures.example/store" }
  ],
  "history": {
    "url": "https://ligatures.example/.well-known/endonend/history.json",
    "headHash": "sha256:9f8e7d..."
  },
  "presentation": {
    "colors": {
      "primary": "#5B96C7",
      "heroBackground": "#5B96C7",
      "background": "#1a1a1a",
      "text": "#e0e0e0"
    },
    "links": {
      "bandcamp": "https://ligatures.bandcamp.com/",
      "instagram": "https://www.instagram.com/ligatures"
    },
    "footer": "Independently released, in partnership with Small Label."
  },
  "catalog": [
    {
      "albumId": "agency-2024",
      "albumVersion": 1,
      "albumName": "Agency",
      "pageTitle": "Ligatures - Agency",
      "releaseDate": "2024-05-01",
      "images": {
        "front": "https://ligatures.example/albums/agency/front.png",
        "back": "https://ligatures.example/albums/agency/back.png",
        "insert": []
      },
      "downloadZip": "https://ligatures.example/albums/agency/agency.zip",
      "purchaseLinks": [
        { "format": "vinyl", "url": "https://ligatures.example/store/agency-vinyl" },
        { "format": "cd", "url": "https://ligatures.example/store/agency-cd" }
      ],
      "credits": {
        "paragraphs": [
          "Written and performed by Ligatures.",
          [
            { "text": "Mastered at " },
            { "text": "Cool Studio", "url": "https://coolstudio.example" },
            { "text": "." }
          ]
        ]
      },
      "splits": [
        { "manifestUrl": "https://ligatures.example/.well-known/endonend/manifest.json", "role": "primary", "percentage": 70 },
        { "manifestUrl": "https://janedoe.example/.well-known/endonend/manifest.json", "role": "feature", "percentage": 20 },
        { "manifestUrl": "https://smalllabel.example/.well-known/endonend/manifest.json", "role": "label", "percentage": 10 }
      ],
      "presentation": {
        "colors": {
          "primary": "#C74B4B",
          "heroBackground": "#8B2020",
          "background": "#1a1a1a",
          "text": "#e0e0e0"
        },
        "footer": "Agency, out now on Small Label."
      },
      "tracks": [
        {
          "trackId": "agency-2024-a1",
          "side": "A",
          "number": "A1",
          "name": "Opening",
          "duration": "3'30\"",
          "file": "https://ligatures.example/albums/agency/songs/opening.mp3",
          "lyrics": "First line\nSecond line"
        }
      ]
    }
  ],
  "signature": {
    "algorithm": "ed25519",
    "value": "base64-signature-over-the-canonicalized-manifest"
  }
}
```

### Example label manifest

A label manifest carries no `catalog`, since it hosts no tracks itself. It exists to declare identity and a roster: the label's own claim about which artists it works with and at what split.

```json
{
  "manifestVersion": "1.0",
  "identity": {
    "type": "label",
    "name": "Small Label",
    "url": "https://smalllabel.example",
    "publicKey": "ed25519:LmNoPq1234567890...",
    "contactEmail": "hello@smalllabel.example"
  },
  "refresh": {
    "ttlSeconds": 21600
  },
  "label": {
    "roster": [
      {
        "artistManifestUrl": "https://ligatures.example/.well-known/endonend/manifest.json",
        "split": { "artist": 85, "label": 15 }
      },
      {
        "artistManifestUrl": "https://anotherband.example/.well-known/endonend/manifest.json",
        "split": { "artist": 80, "label": 20 }
      }
    ]
  },
  "history": {
    "url": "https://smalllabel.example/.well-known/endonend/history.json",
    "headHash": "sha256:1a2b3c..."
  },
  "presentation": {
    "links": {
      "website": "https://smalllabel.example"
    },
    "footer": "A small, transparent label. Every split is public."
  },
  "signature": {
    "algorithm": "ed25519",
    "value": "base64-signature-over-the-canonicalized-manifest"
  }
}
```

This roster entry for Ligatures matches Ligatures' own `label.split` (`{ "artist": 85, "label": 15 }`) in the artist example above, so the affiliation shows as verified: both sides declared the same split independently.

An artist without their own domain identifies the same way, just with a URL that includes a path instead of being the domain root, for example:

```json
"identity": {
  "type": "artist",
  "name": "Small Combo",
  "url": "https://smallcombo.github.io/small-combo",
  "publicKey": "ed25519:XyZ...",
  "contactEmail": "band@smallcombo.example"
}
```

Their manifest lives at `https://smallcombo.github.io/small-combo/.well-known/endonend/manifest.json`. The validation rule is identical: fetch location must equal `identity.url` plus the fixed suffix. Nothing about verification cares whether the URL is a domain root or a path on shared hosting.

### Example contribution confirmation

The featured artist on "Agency" (`janedoe.example`) has no roster to list a claim in, so she confirms her 20% through `contributions` on her own manifest, an excerpt:

```json
{
  "contributions": [
    {
      "manifestUrl": "https://ligatures.example/.well-known/endonend/manifest.json",
      "albumId": "agency-2024",
      "albumVersion": 1,
      "role": "feature",
      "percentage": 20
    }
  ]
}
```

This matches the `splits` entry for her on Ligatures' "Agency" album above, so that entry also shows as verified.

## Validation rules

- Every required field listed above is present and of the correct type.
- Every field typed `string (URL)`, or an array of that type such as `images.insert`, (including `merch[].url`, `catalog[].purchaseLinks[].url`, and every other link in the manifest) is a well-formed `http` or `https` URL; no other schemes are accepted.
- `images.front` and `images.back` are both present on every album; `images.insert` may be an empty array but must be present as a field.
- `identity.url` plus the fixed suffix `/.well-known/endonend/manifest.json` equals the URL the manifest was actually fetched from, so a URL can't claim an identity it doesn't actually serve.
- `identity.publicKey` is well-formed, and `signature.value` verifies against it over the canonicalized manifest.
- Every `signature.algorithm`, on the manifest itself and on every history entry, is exactly `"ed25519"`; any other value is rejected outright, never tolerated as an unrecognized-but-harmless field.
- If `label.split` (or a roster entry's `split`) is present, its values sum to 100.
- If an album's `splits` is present, its `percentage` values sum to 100.
- Label affiliation is marked verified only under mutual attestation (the artist's `label` object and the label's `roster`), per the `label` section above; a mismatch is surfaced, not hidden.
- An album's `splits` entries are marked verified only when each has a matching `contributions` entry in that party's own manifest, per the `contributions` section above; a mismatch or missing confirmation is surfaced, not hidden.
- `history.headHash`, if present, matches the actual head of the log fetched from `history.url`, and that log's hash chain verifies unbroken.
- `identity.publicKey` matches the key already pinned for this `identity.url` from a prior check, or, if it has changed, the history log contains a `key_rotated` entry authorizing exactly that change, signed by the old key. See Key rotation above.
- `albumId` and `trackId` values are unique within the manifest, and `albumVersion` only ever increases for a given `albumId`.

Any failure here is exactly what triggers the crawler's or CLI's rejection-and-notification flow described in 0002's Protocol enforcement section.

## Open questions

- How long a one-sided label-affiliation claim is shown as "pending" before being treated as disputed, to cover the ordinary propagation delay between an artist publishing an update and the label's own manifest catching up on its next crawl.
- Whether `albumId` and `trackId` need any uniqueness guarantee beyond "stable within one artist's manifest," for example if a track is ever referenced from outside that artist's own catalog.
- The literal machine-readable JSON Schema file generated from this spec, the artifact both the Go and Kotlin Multiplatform validators actually compile or test against, per 0002's "kept honest by a shared schema" approach.
- Whether `contributions` needs an upper bound or pagination once a prolific session musician or producer accumulates confirmations across many artists' catalogs, so their own manifest doesn't grow unbounded.
- The same pending-versus-disputed grace period question as label affiliation, applied to album `splits` awaiting a `contributions` match.

## References

- [0001-purpose.md](./0001-purpose.md): the principles this format must satisfy.
- [0002-architecture.md](./0002-architecture.md): the protocol and system design this format implements.
- [KB/README.md](./README.md): KB conventions this document follows.
