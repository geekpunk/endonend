# Purpose

```
Status: Accepted
Date: 2026-09-07
Type: Purpose
References: [KB/README.md](./README.md)
```

## Problem statement

Spotify, Apple Music, and YouTube Music have become gatekeepers that extract a disproportionate share of the value created by artists: the streaming companies now take a larger effective cut than the record labels they were supposed to disrupt. This isn't limited to major-label acts: independent artists and independent labels, who already operate on thin margins, pay the same platform tax as everyone else, with no meaningful alternative that reaches listeners at scale.

Before streaming, it was quite possible to opt out of the traditional entertainment-industry structure entirely, through independently run labels, independently run record stores, independent distributors, or simply handling every component yourself. Streaming infrastructure has made that nearly impossible: it is now very difficult to exist as an independent artist without participating in a small number of closed platforms.

On top of the financial extraction, these platforms are opaque by design. Artists and fans have no visibility into how streams convert to payouts, what any label/distributor split actually is, or what infrastructure cost the platform is supposedly recouping. Artists also have no control over or ownership of the actual files that represent their work, or the analytics generated from their own fans listening to it; all of it lives inside a closed, proprietary system.  Thus the artist and the consumer are the products for another company.

The current systems are also highly manipulable. An artist's fan base and opportunities get abstracted into a listen count, but like any counter on the internet, that count can be manipulated to make an artist seem more popular than they are with real listeners, undermining the discovery and opportunity the platforms claim to provide and replacing it with a whole new media system to obscure artists from fans.

Finally, as record formats got cheaper (from records to tapes to CDs to streams) and production and marketing got more efficient, the relationship to art has been diluted, both in physical form and in what actually reaches the artist. The great irony of the open web and the near-elimination of physical production costs is that it was supposed to hand more power directly to artists. It didn't. We believe that can change, and there has never been an easier or cheaper time to change it, given how far the cost of compute, storage, and build time has fallen.

## Purpose / mission

This system exists to give artists a direct-to-listener streaming alternative that returns control and profit to artists, without recreating the publisher/distributor middleman layer that streaming was supposed to replace.

The core mechanism: **artists (or the label they choose to work with) host their own audio files, images, and other artifacts, and own their own serving and analytics.** The system does not sit in the artist's revenue or storage path. Instead, it represents and aggregates independent artists to give them collective reach, discoverability, and convenience for listeners, without ever owning or intermediating the artist's work or income the way streaming platforms traditionally would.

To make that work at the scale and convenience listeners expect from Spotify or Apple Music, the system is both a **protocol** (so artist-hosted catalogs can be published, discovered, and played in a standard way) and a **centralized reference experience** (a search, discovery, and playback surface built on top of that protocol, so listeners get one convenient place to find and play music without that convenience requiring artists to give up ownership of their files or data).

## Core principles

These are the tenets every future spec and decision in this KB must be consistent with:

- **Artist-owned infrastructure.** Artists and/or their labels host all song files, images, and other artifacts themselves. The system never becomes the custodian of an artist's files or the intermediary through which their revenue flows.
- **Artists do not need to be technologists.** Self-hosting is a principle, not a barrier to entry. The central platform builds the tools and managed capabilities needed for any artist to participate and contribute, regardless of their technical background. This is in direct tension with artist-owned infrastructure: the platform must make self-hosting achievable without requiring artists to run their own servers or write code. See open questions below.
- **Protocol + centralized experience, not just one or the other.** An open protocol defines how artist-hosted catalogs are published, discovered, and streamed. A reference centralized search/discovery/play experience is built on top of that protocol for listener convenience, but because the whole system is open source and self-hostable, that reference experience is not the only possible front door into the protocol.
- **No platform left behind.** The reference experience reaches listeners on web and as native mobile apps published to the standard app stores (iOS and Android), so no platform is treated as an afterthought and listeners get the convenience they already expect.
- **Radical transparency.** Label/artist revenue splits are publicly visible to listeners, per release. The system's own operating costs (compute, storage, bandwidth, and other infrastructure it directly uses) are published on an ongoing basis, so the system holds itself to the same transparency it demands of splits.
- **Verifiable, manipulation-resistant listening signals.** Listen counts, popularity, and discovery ranking must be meaningfully harder to fake than a plain counter. This is in direct tension with artist-owned infrastructure: because artists run their own serving, the system must be designed so listening signals can be trusted without requiring a central authority to control the serving layer. See open questions below.
- **Verifiable artists and labels, open history.** Artist and label identities must be verifiable, not merely self-asserted, and every profile carries an openly viewable history: past releases, label affiliations, and changes to those affiliations or splits over time. History is recorded, not quietly edited away or rewritten. See open questions below.
- **No legacy publisher/distributor layer.** A label construct is allowed (artists can choose to work with a label), but that relationship is a transparent, artist-consented split, not an imposed intermediary standing between the artist and their listeners or their money.  All publishing rights will be availabe to everyone.
- **Open source and self-hostable by design.** The full system can be deployed by anyone, not just operated as a single hosted service.
- **Container-based deployment.** Artist-hosted serving and any full system instance are deployable to any cloud or their own hardware without being locked into a specific vendor or hosting stack.
- **Cross-instance interoperability.** Because anyone can run their own instance, independently-run instances are expected to be able to interoperate with each other (discovery and playback across instances), rather than each instance being an isolated silo.

## Use of AI

The system itself (its infrastructure, tooling, and code) is built with the help of frontier AI models such as Claude and Codex. These are development tools used to build the system; that use is separate from what ships to artists and listeners as part of the product.

Using frontier models in development is not incidental: it is central to the hypothesis in this document that a system like this is far less costly and complex to build than it once was (see the problem statement above). That falling cost and complexity is what makes it realistic for a system built to benefit artists to catch up with, and compete with, an incumbent the scale of Spotify.

Any user-facing feature that is itself built on an LLM (for example, natural-language search, tagging, or moderation assistance) is built on open-weight models, not closed frontier APIs. This keeps the system's own operation independent of any single proprietary model vendor, consistent with the system's broader commitment to not depending on closed platforms it doesn't control.

Wherever a traditional, non-LLM ML approach can accomplish the same outcome as an LLM, it is preferred over an LLM-based approach, in order to reduce the energy footprint of running the system.

## Explicit non-goals for this document

This document establishes *why* the system exists and the principles it must honor. It deliberately does not:

- Define the MVP or any implementation scope; that is a separate spec.
- Design playlists, fan sharing, or Spotify-style shuffle/radio; these are explicitly deferred to later specs, once the core artist-hosting and discovery model is established.
- Define long term roadmap.
- Pick a technology stack or architecture.

## Open questions for future specs

The principles above commit the system to solving these, but the *how* is intentionally left to dedicated future specs rather than decided here. Several have since been answered by later specs; those are marked resolved below rather than removed, so the history of *why* stays intact per `KB/README.md`.

- **Split transparency mechanism.** Resolved by [0003-manifest.md](./0003-manifest.md): per-release `splits` and `contributions` with mutual attestation, surfaced to listeners as verified, unverified, or disputed rather than silently trusted.
- **Self-hosting accessibility.** Resolved by [0002-architecture.md](./0002-architecture.md)'s static-file artist catalog model and [0004-endonend-artist-cli.md](./0004-endonend-artist-cli.md)'s CLI, which generates, signs, and validates a manifest locally without an artist needing to run a server or hand-write JSON.
- **App store distribution constraints.** Resolved: both native apps (per [0002-architecture.md](./0002-architecture.md)'s Languages and stack) are discovery and playback only. Any purchase (`catalog[].purchaseLinks`, `merch` in [0003-manifest.md](./0003-manifest.md)) opens out to the artist's own storefront rather than an in-app purchase flow. Since nothing is sold through the app itself, there is no Apple/Google in-app-purchase commission to reconcile with the non-intermediary principle, and no store-review conflict over payment flows to design around.
- **Identity verification mechanism.** Resolved by [0002-architecture.md](./0002-architecture.md) and [0003-manifest.md](./0003-manifest.md): a DID:web-style trust-on-first-use key pinned to `identity.url`, with a key change only accepted via a `key_rotated` history entry signed by the old key. No party grants or revokes this; it's derived entirely from key possession.
- **History and audit trail integrity.** Resolved by [0003-manifest.md](./0003-manifest.md)'s append-only, hash-chained history log.
- **Listen verification / anti-manipulation mechanism.** Still open. How the system verifies or attests real listens and derives popularity/discovery signals from artist-hosted, self-reported serving infrastructure, given that the same decentralization that empowers artists also removes the platform's ability to unilaterally audit play counts the way a closed platform can.
- **Cross-instance interoperability / federation protocol.** First-pass sketch in [0009-federation.md](./0009-federation.md) (`Draft` status): opt-in peer instance manifests and catalog-index feeds treated only as leads, with every artist manifest still independently verified at the source and cross-instance playback needing no protocol work at all. Wire format, rate-limiting, and cold-start details remain open there.
- **Governance.** Resolved by [0008-governance.md](./0008-governance.md): purely protocol-level, no central authority; conformance to the protocol is mechanical (passing the shared schema and conformance test vectors), not a trust relationship with any party.
- **Central entity business structure.** Deliberately deferred: no formal legal entity behind the reference instance yet. It is run personally by the founder until there's real usage to justify the overhead, decoupled from governance ([0008-governance.md](./0008-governance.md)) so this stays a pure operating-entity question, not a protocol-authority one. Revisit once the MVP validation threshold below is actually met; the choice among a nonprofit, a cooperative owned by member artists and labels, a foundation, or a transparently-margined for-profit, and how it sustains itself financially, is still fully open at that point.
- **MVP validation threshold.** Resolved by [0010-mvp-scope.md](./0010-mvp-scope.md): 5-10 real artists actually publishing and getting real listens, with qualitative feedback mattering more than the raw count.
- **Centralized endonend deployment architecture.** Resolved by [0002-architecture.md](./0002-architecture.md)'s "Initial deployment (GCP)" section, with the remaining GCP-specific detail deferred to a dedicated future deployment spec per [0006-container-infrastructure.md](./0006-container-infrastructure.md)'s scope note.
- **Discovery/search centralization vs. decentralization.** Substantially answered for a single instance by [0002-architecture.md](./0002-architecture.md) (decentralized artist-hosted storage, centralized search index synced from Postgres). The cross-instance version of this question is now the same question as "Cross-instance interoperability / federation protocol" above.
- **Compute/storage transparency reporting.** Resolved by [0002-architecture.md](./0002-architecture.md)'s "Compute/storage transparency reporting" section: a monthly, plain-language public report bucketing GCP billing export data into compute, storage, bandwidth, and third-party services, scoped to the reference instance's own operating cost only. No self-hosted instance is required to publish one, per [0008-governance.md](./0008-governance.md); the same tooling is offered as optional for self-hosters who want to.

## What comes next

Subsequent specs will be added to this KB as sequentially numbered documents, each referencing this purpose document, starting with an MVP spec, followed by specs for the artist-hosting protocol, artist onboarding/self-hosting tooling, web and mobile client apps, discovery/search, split-transparency mechanism, identity verification and history, listen-verification/anti-manipulation mechanism, central entity governance and business structure, and cross-instance interoperability. Playlists, fan sharing, and radio/shuffle-style discovery are explicitly post-MVP and will get their own specs later.

## References

- [KB/README.md](./README.md): KB conventions this document follows.
