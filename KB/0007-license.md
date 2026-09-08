# License

```
Status: Accepted
Date: 2026-09-08
Type: Decision
References: [0001-purpose.md](./0001-purpose.md)
```

## Decision

The project is licensed MIT, recorded in [`LICENSE`](../LICENSE) at the repository root and applying to every component (`cli/`, and every directory listed in the top-level `README.md` once it has code).

## Context

[0001-purpose.md](./0001-purpose.md) commits to the system being "open source and self-hostable by design," but doesn't pick a specific license; governance of the "union" aspect is explicitly left as an open question there. A license had to be chosen before any code could be meaningfully called open source.

## Alternatives considered

- **AGPL-3.0.** Network copyleft: a modified version run as a public service must release its changes. Would most directly reinforce 0001's anti-gatekeeper framing (the same reasoning projects like Mastodon use it), by making it harder for a company to fork the Union Platform into a closed, proprietary competing service.
- **Apache-2.0.** Permissive, with an explicit patent grant. Maximizes adoption of the manifest protocol and CLI, including by companies building closed-source products on top, at the cost of not requiring downstream changes to stay open.
- **MIT.** Same permissive trade-off as Apache-2.0, without the patent clause; shorter and more broadly recognized, especially for a CLI tool and a protocol meant to be implemented freely by third parties (per 0001's cross-instance interoperability goal).

## Why MIT

The project's core mechanism, per 0001, is a protocol other implementations are meant to interoperate with, not a service the project needs to protect from being run elsewhere. A copyleft license on the manifest format and CLI would cut against "any independently run instance can also implement" the protocol, since it would pressure every implementer, including ones with no interest in competing with the reference Union Platform, into copyleft terms just to parse a manifest. MIT keeps the barrier to adoption as low as possible for artists, third-party tooling, and alternate Union Platform instances alike, which matters more at this stage than foreclosing a future proprietary fork of the reference instance itself.

This doesn't resolve the "Governance of the 'union' aspect" open question in [0001-purpose.md](./0001-purpose.md); it only settles the code license. A future foundation, trademark policy, or protocol-conformance mark is a separate governance mechanism, not a licensing one, and remains open.

## References

- [0001-purpose.md](./0001-purpose.md): the open-source and cross-instance-interoperability principles this choice serves.
- [KB/README.md](./README.md): KB conventions this document follows.
