# endonend

Streaming platforms like Spotify, Apple Music, and YouTube Music now take a larger effective cut of an artist's revenue than the record labels they were supposed to disrupt, while staying opaque about payouts, splits, and infrastructure cost, and leaving artists with no ownership of the files or listener data their own work generates. endonend is a direct-to-listener alternative that gives that control back, without recreating the publisher/distributor middleman layer streaming was supposed to replace.

The core mechanism: **artists, or the label they choose to work with, host their own audio files, images, and other artifacts, and own their own serving and analytics.** endonend never sits in that revenue or storage path; instead it aggregates independent artists to give them collective reach and discoverability without ever intermediating their work or income. To make that as convenient as a closed platform, endonend is both an open **protocol** (so artist-hosted catalogs can be published, discovered, and played in a standard way) and a **centralized reference experience** (a search, discovery, and playback surface built on that protocol) that any independently run instance can also implement, since the whole system is open source and self-hostable.

This is paired with radical transparency (public label/artist splits, published operating costs), verifiable listening signals that are harder to fake than a plain counter, and verifiable artist/label identity with an open, tamper-evident history, rather than a central authority's approval.

Full context lives in [`KB/`](./KB), starting from [`KB/0001-purpose.md`](./KB/0001-purpose.md) (the project charter, problem statement, and core principles) and [`KB/0002-architecture.md`](./KB/0002-architecture.md) (MVP architecture, languages, CI/CD). Read `KB/README.md` before adding or changing any spec.

## Repository layout

| Path | Contents |
|------|----------|
| `KB/` | Specs and decisions: the single source of truth for how the system works |
| `cli/` | Go CLI that generates, signs, and validates an artist/label manifest ([`KB/0004`](./KB/0004-endonend-artist-cli.md)) |
| `api/` | endonend platform catalog API (planned) |
| `web/` | Web frontend: discovery site and generated artist storefronts (planned) |
| `mobile/` | Native iOS and Android clients (planned) |
| `data/` | endonend platform persistence layer (planned) |
| `infra/` | Deployment and infrastructure config (planned) |
| `tests/` | Cross-component and conformance tests (planned) |
| `scripts/` | Repo tooling, including `scripts/git-hooks/` |

Most of these directories are placeholders until the corresponding spec lands in `KB/`; only `cli/` has code today.

## Architecture

Full detail, rationale, and open questions live in [`KB/0002-architecture.md`](./KB/0002-architecture.md); this is a summary.

**System components.** There are two parts, not three: there is no artist-side service.

- **Artist catalog**: a signed manifest plus audio and image files, published as plain static files on any host the artist chooses (object storage, a static site host, their own server). Nothing to run or operate.
- **Optional artist/label storefront**: a static HTML/CSS/JS site generated from the same manifest, for artists who want their own branded presence.
- **endonend platform**: the central reference instance, a crawler/indexer that fetches and verifies artist manifests over plain HTTP, a catalog API, search/discovery, and the web and mobile clients. It never stores artist audio or image files, only metadata and pointers.
- **Protocol**: the manifest format and fetch-and-verify rules ([`KB/0003-manifest.md`](./KB/0003-manifest.md)) that let any artist's static files interoperate with any endonend platform instance.

**Languages and stack:**

- **endonend platform backend** (crawler, catalog API, search sync): Go, for small images and concurrent manifest fetching/verification.
- **Artist-side manifest CLI** ([`cli/`](./cli)): also Go, a single static binary, no service.
- **Web frontend**: TypeScript and React (e.g. Next.js), server-rendered for the discovery site and statically exportable per-artist as a self-hostable storefront.
- **Mobile**: native Swift/SwiftUI (iOS) and Kotlin/Jetpack Compose (Android), sharing manifest parsing, verification, and caching through a Kotlin Multiplatform module rather than a cross-platform UI framework.
- Both backend implementations (Go and Kotlin Multiplatform) validate against the same versioned manifest JSON Schema and a shared conformance fixture set, so "valid" means the same thing everywhere it's checked.

**Persistence and caching.** Postgres is the source of truth for everything the endonend platform manages centrally: catalog metadata, identity records, splits, and an append-only history/audit table. Meilisearch stays synced from Postgres for discovery. Audio, image, and other artifacts are never persisted centrally, only cached: a manifest cache using standard HTTP conditional requests, and an optional, opt-in, TTL'd media edge cache that always relays play events back to the artist so caching never creates a gap in their own analytics.

**Deployment.** Artists deploy nothing beyond uploading files. The endonend platform ships as a `docker-compose.yml` for small self-run instances and a Helm chart for larger deployments, with multi-arch (amd64/arm64) images on a public OCI registry and no hard dependency on a specific cloud. endonend's own first deployment runs on GCP (Cloud Run for the API and crawler, Cloud SQL for Postgres, Cloud CDN as the media edge cache), as one operational choice among many the same images support, not a requirement of the architecture.

**CI/CD.** GitHub Actions: lint, test, and build on every pull request ([`.github/workflows/go.yml`](./.github/workflows/go.yml) for the Go modules today); multi-arch container images built, signed, and pushed on merge and tag once there's a service to ship; the iOS and Android apps build on their own Xcode/Gradle pipelines once they exist.

## Getting started

Prerequisites: Go, matching the version pinned in `cli/go.mod`.

Clone the repository, then build and run the manifest CLI, the only component with code today:

```sh
cd cli
go build -o endonend-artist-cli ./cmd
./endonend-artist-cli              # launches an interactive menu
```

Or run individual commands directly:

```sh
./endonend-artist-cli generate                # sign a manifest.json + history.json from endonend.source.json
./endonend-artist-cli validate manifest.json  # validate a manifest
./endonend-artist-cli keygen --url https://example.com/artist
./endonend-artist-cli help                    # full command and flag reference
```

See [`KB/0004-endonend-artist-cli.md`](./KB/0004-endonend-artist-cli.md) for the CLI's design and [`KB/0003-manifest.md`](./KB/0003-manifest.md) for the manifest format it produces.

## Development

### Go

The Go CLI lives in `cli/` (module `endonend/cli`). From that directory:

```sh
go build ./...
go vet ./...
go test ./...
golangci-lint run ./...
```

See the Go section of [`CLAUDE.md`](./CLAUDE.md) for testing and linting conventions.

### Git hooks

One-time setup per clone:

```sh
git config core.hooksPath scripts/git-hooks
```

This runs `go test` and `golangci-lint` on affected Go modules before each commit and push, requires [`golangci-lint`](https://golangci-lint.run) on your `PATH` (e.g. `brew install golangci-lint`).

### CI

`.github/workflows/go.yml` runs the same build, vet, test, and lint steps on every push to `main` and every pull request.

## License

MIT, see [`LICENSE`](./LICENSE). See [`KB/0007-license.md`](./KB/0007-license.md) for the alternatives considered and why.
