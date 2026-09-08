# Cross-instance federation

```
Status: Draft
Date: 2026-09-07
Type: Spec
References: [0001-purpose.md](./0001-purpose.md), [0002-architecture.md](./0002-architecture.md), [0003-manifest.md](./0003-manifest.md), [0008-governance.md](./0008-governance.md)
```

## Overview

This is a first-pass sketch of how independently-run endonend instances discover each other and interoperate, answering [0001-purpose.md](./0001-purpose.md)'s "Cross-instance interoperability / federation protocol" open question and the cross-instance half of its "Discovery/search centralization vs. decentralization" question. It stays `Draft`: the overall shape is settled, but wire formats, pagination, and rate-limiting details are still open below.

## Design goals

- **No live push API, matching 0002.** Instances federate the same way the crawler already talks to artists: fetch-and-verify over plain HTTP, not a request/response protocol either side must keep running to serve.
- **Opt-in, directional peering, matching 0008.** No central directory, no approval process, no entity with standing to admit or reject a peer. An instance operator lists whichever peers they want; peering need not be reciprocal.
- **Never weakens per-artist verification.** A peer can only ever suggest which artist URLs might be worth checking. It can never assert that a manifest is valid; every instance still independently fetches and signature-checks every artist manifest at that artist's own origin, per 0003.

## Instance manifest

Each instance optionally publishes its own manifest at:

```
<instance-url>/.well-known/endonend/instance.json
```

Carrying the instance's identity (name, URL, `publicKey`, reusing the same identity shape 0003 already defines for artists and labels) and a URL to its catalog index feed, below. Publishing this file is what makes an instance federatable at all; an instance that never publishes one simply isn't discoverable as a peer, which is fine, nothing about running a standalone instance requires it.

## Catalog index feed

A paginated, cursor-based JSON feed, served by the instance's own catalog API (not a static file, since the reference implementation already runs a live API per 0002), listing the artist (and label) manifest URLs this instance has successfully indexed. This is a lead list, not a data export: it contains URLs, not manifest contents.

## Peering

An instance operator configures a list of peer `instance.json` URLs to also poll, on the same TTL/conditional-request caching model 0002 already uses for artist manifests. Peering is a subscription, not a mutual-attestation relationship like label affiliation in 0003: instance A can follow B's feed without B following A's, since there's no shared claim (like a revenue split) that needs both sides to confirm, only a candidate list one operator chose to trust as a source of leads.

## Trust model

A peer's feed is never trusted data, only a hint. When instance A learns of a new artist URL from instance B's feed, A fetches and verifies that artist's manifest directly from the artist's own origin, exactly as if A had discovered the URL any other way. A compromised or malicious peer can at most waste another instance's crawl cycles pointing it at URLs that fail validation or don't exist; it can never inject a fake catalog entry, since nothing from a peer ever bypasses the independent verification 0003 already requires.

## Search and discovery scope

Each instance keeps running its own Postgres and Meilisearch, synced from its own crawl results, per 0002's persistence layer. Federation only widens the set of artist URLs an instance's crawler knows to go check; it does not create a shared or merged search index. A listener searching on instance A only ever searches what instance A itself has independently verified and indexed, even when the lead that got an artist onto A's radar originally came from instance B's feed.

## Cross-instance playback

Needs no protocol work at all. Every track's `file` URL in the manifest (per 0003) always points directly at the artist's own hosted file. Any client on any instance can already play any artist it has a verified manifest for, regardless of which instance's crawler first discovered that artist.

## Open questions

- The exact wire format and cursor/pagination scheme for the catalog index feed.
- Whether the feed should be cached with the same conditional-request (`ETag`/`If-None-Match`) approach 0002 already uses for artist manifests, to keep polling a large peer's feed cheap.
- Whether federation includes label manifests and roster data, and if so, whether that changes anything about how affiliation/split verification is checked (it shouldn't: verification always happens against the source manifest regardless of how an instance learned the URL).
- Whether `instance.json` needs its own signature (parallel to `identity.publicKey`) so a spoofed instance manifest can't be used to feed junk URLs at scale, versus relying on operators only peering with instances they already know, given the blast radius of a malicious peer is already limited to wasted crawl cycles.
- Whether there should be a maximum peer count or feed size, so a small self-run instance's crawler can't be overwhelmed by peering with an instance hosting a much larger catalog.
- The cold-start problem: how a listener on instance A ever sees an artist known only to instance B before A's crawler has caught up on that peer's feed for the first time.
- Whether peer instances should ever be discoverable through any shared list or registry, or purely through operators manually configuring peers they already know about; per [0008-governance.md](./0008-governance.md), even a voluntary "known good instances" aggregator risks becoming an implicit gatekeeper.

## References

- [0001-purpose.md](./0001-purpose.md): the cross-instance interoperability principle and open question this addresses.
- [0002-architecture.md](./0002-architecture.md): the fetch-and-verify crawler model this builds on.
- [0003-manifest.md](./0003-manifest.md): the artist/label manifest format every federated lead still resolves to and is independently verified against.
- [0008-governance.md](./0008-governance.md): the no-central-authority stance that makes peering opt-in and directional rather than requiring approval.
- [KB/README.md](./README.md): KB conventions this document follows.
