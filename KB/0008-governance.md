# Governance

```
Status: Accepted
Date: 2026-09-07
Type: Decision
References: [0001-purpose.md](./0001-purpose.md), [0003-manifest.md](./0003-manifest.md), [0007-license.md](./0007-license.md)
```

## Decision

Governance of the protocol and the reference endonend platform is purely protocol-level: there is no foundation, nonprofit board, or elected body with authority over the protocol, and no entity has standing to grant or revoke another instance's ability to interoperate.

- The manifest schema, canonicalization rules, and shared conformance test vectors defined in [0003-manifest.md](./0003-manifest.md) are the single source of truth for what a valid manifest is. Conformance is mechanical: an instance interoperates if its manifests and crawler pass the shared schema and test vectors, not because any party approved it.
- The protocol evolves through ordinary open-source maintenance of this repository: pull requests against the KB and the shared Go/Kotlin Multiplatform validators, versioned by `manifestVersion` (major.minor, per 0003). A breaking change requires a major version bump; any implementer, including a competing discovery platform, can adopt it on their own timeline or not at all.
- Today, that maintenance is one person, the project's founder. "Open project" describes the intended long-term shape, not a present formal structure: there is no maintainer-admission process, BDFL charter, or maintainer council yet, and none is being pre-designed here. It grows organically as contributors show up.
- The entity that ends up operating the reference endonend platform instance (the still-open "Central entity business structure" question below) has no special authority over the protocol. It is the first and default implementer, not a gatekeeper: any other instance that passes the shared validator is equally part of the protocol.

## Context

[0001-purpose.md](./0001-purpose.md) left "Governance of the 'union' aspect" open: whether a foundation, nonprofit, or elected body sits behind the reference instance and protocol standard, or whether governance stays purely protocol-level with no central authority. Answering it also unblocks two things that were stuck behind it: [0007-license.md](./0007-license.md)'s note that "a future foundation, trademark policy, or protocol-conformance mark is a separate governance mechanism... and remains open," and the scope of the still-open "Central entity business structure" question, which risked being conflated with protocol governance instead of treated as the separate operating-entity question it actually is.

## Alternatives considered

- **A foundation or nonprofit board governing the protocol.** The ActivityPub/W3C model: a formal, accountable steward for the spec. Rejected for now, standing up a foundation is overhead this project doesn't need yet, and it would install exactly the kind of central authority [0001-purpose.md](./0001-purpose.md)'s "no central gatekeeper" principle argues against, before a second implementer even exists to govern relative to.
- **An elected or rotating maintainer body.** Defers "who decides" to a process instead of a person, but presumes a contributor base large enough to elect from; none exists yet.
- **Pure protocol-level governance, no central authority (chosen).** The protocol is defined entirely by its schema, canonicalization rules, and conformance test vectors; conformance is mechanical, never a trust relationship with any party. The reference implementation and this KB get ordinary open-source maintenance, today by one person, open to broadening later.

## Why this fits

This keeps governance consistent with 0001's "no central gatekeeper" principle from day one, instead of deferring it until a foundation feels necessary and retrofitting decentralization onto an already-central body. It also cleanly separates two questions that are easy to conflate: who has authority over the protocol (nobody, by design) and what legal/business entity operates the reference instance (still open, tracked in [0001-purpose.md](./0001-purpose.md)'s "Central entity business structure").

## References

- [0001-purpose.md](./0001-purpose.md): the governance question this resolves, and the business-structure question it's decoupled from.
- [0003-manifest.md](./0003-manifest.md): the schema, versioning, and conformance-test-vector mechanism that makes conformance mechanical rather than a trust decision.
- [0007-license.md](./0007-license.md): the code-license decision this complements; a future trademark or conformance-mark policy remains open and is a separate governance mechanism.
- [KB/README.md](./README.md): KB conventions this document follows.
