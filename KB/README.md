 # Knowledge Base

`KB/` is the single source of truth for this project's specs and decisions. Every document here traces back (directly or transitively) to [0001-purpose.md](./0001-purpose.md), the foundational charter. Nothing about how this system works should live only in code or only in someone's head; it should be written down here first.

## Naming convention

Files are named `NNNN-kebab-case-title.md`, where `NNNN` is a zero-padded, sequential 4-digit number assigned when the document is created (`0001`, `0002`, ...). Numbers are never reused or renumbered, even if a document is later superseded or deprecated; the sequence is a permanent record of order, not a live index.

## Document types

Kept informal for now (plain files, not folders; this can be revisited into subfolders if the doc count grows):

- **Purpose**: the single foundational charter. There is exactly one: `0001-purpose.md`.
- **Spec**: a feature or system design (e.g. the MVP, the artist-hosting protocol, discovery/search, label/artist split transparency, cross-instance interop, later playlists/social/radio).
- **Decision**: a narrow, ADR-style record of one specific technical or governance choice, including the alternatives considered and why.

## Document frontmatter

Every document should open with a short metadata block:

```
Status: Draft | Proposed | Accepted | Superseded
Date: YYYY-MM-DD
Type: Purpose | Spec | Decision
Supersedes: (optional, link to a doc this replaces)
Superseded-by: (optional, link to a doc that replaces this)
References: (links to other KB docs this one depends on or extends)
```

## Cross-referencing rule

Every document must link to the documents it builds on, using relative markdown links (e.g. `[0001-purpose.md](./0001-purpose.md)`) so links work both on GitHub and locally. At minimum, every non-purpose document links back to `0001-purpose.md`. If a spec depends on a decision or another spec, link it explicitly in `References` and inline where relevant, rather than assuming the reader has the full picture.

## Status lifecycle

`Draft` (being written/discussed) → `Proposed` (ready for review) → `Accepted` (in effect) → `Superseded` (replaced by a newer doc, linked via `Superseded-by`). A document is never silently rewritten to reverse a prior decision; it's superseded by a new numbered doc instead, so the history of *why* stays intact.

## Index

| # | Title | Type | Status | Summary |
|---|-------|------|--------|---------|
| [0001](./0001-purpose.md) | Purpose | Purpose | Accepted | Why this system exists: a direct-to-artist, transparent, open-source alternative to major streaming platforms. |
| [0002](./0002-architecture.md) | Architecture | Spec | Proposed | MVP architecture: languages, deployment, CI/CD, persistence, and how the static-file artist/label protocol is enforced. |
| [0003](./0003-manifest.md) | Manifest | Spec | Proposed | The signed manifest file format artists and labels publish: identity, catalog, splits, history, and storefront presentation. |
| [0004](./0004-endtoend-artist-cli.md) | endtoend-artist-cli | Spec | Proposed | The Go CLI that generates, signs, and validates a manifest, interactive menu by default, flags for CI. |
| [0005](./0005-bandcamp-import.md) | Bandcamp import | Spec | Proposed | `import bandcamp` prefills `union.source.json` from an existing Bandcamp album page, downloading art/audio locally rather than linking to Bandcamp's hosting. |
| [0006](./0006-container-infrastructure.md) | Container infrastructure | Spec | Proposed | Generic container build, compose, Helm, and multi-arch signed-image publishing for the Union Platform backend; GCP-specific deployment deferred. |
| [0007](./0007-license.md) | License | Decision | Accepted | The project is MIT-licensed, recorded in `LICENSE`; alternatives (AGPL-3.0, Apache-2.0) considered and why MIT fits the protocol-interoperability goal best. |
