# Container infrastructure

```
Status: Proposed
Date: 2026-09-07
Type: Spec
References: [0001-purpose.md](./0001-purpose.md), [0002-architecture.md](./0002-architecture.md)
```

## Overview

[0002-architecture.md](./0002-architecture.md) already commits, at a prose level, to Docker containers for the endonend platform: a `docker-compose.yml` for small self-run instances, a Helm chart for larger or cloud deployments, multi-arch OCI images, and signed images built in CI (see its "Deployment architecture" and "CI/CD" sections). This document makes those commitments concrete: an image build pattern, a compose service list, a Helm chart layout, a tagging and signing scheme, and the shape of the CI pipeline that produces all of it.

**In scope**: the Dockerfile pattern for Go services, `docker-compose.yml`, the Helm chart, multi-arch OCI image publishing, image signing, and the generic build/push/sign CI pipeline.

**Explicitly out of scope**:

- GCP-specific deployment (Cloud Run, Cloud SQL, Secret Manager, Workload Identity Federation). [0002-architecture.md](./0002-architecture.md)'s "Initial deployment (GCP)" section already sketches this at a narrative level; a dedicated deployment spec is future work, not this document.
- The backend service's own code and business logic (the crawler, catalog API, and search sync described in 0002's "Languages and stack" section). No code exists yet under `api/`; the patterns below are written to work whatever that code becomes, not to prescribe it.
- Mobile and Kotlin Multiplatform build and release pipelines, already covered separately in 0002's CI/CD section and unrelated to containers.
- Actually creating any Dockerfile, compose file, Helm chart, or CI workflow file. This document describes what will be built; writing those artifacts is a separate implementation task.

## Image build strategy

A multi-stage build: a `golang:<pinned-version>` build stage, version matching the eventual backend module's `go.mod` (the same pinning convention `cli/go.mod` already uses), and a `gcr.io/distroless/static-debian12:nonroot` runtime stage.

Go binaries compiled with `CGO_ENABLED=0` are statically linked, so a distroless-static runtime has everything they need, including the CA certificates the crawler requires for outbound HTTPS fetches of artist manifests. This is a smaller attack surface than an Alpine base (no shell, no package manager to invoke) without scratch's rough edges (manually copying CA certificates in). The `nonroot` tag runs as a non-root numeric UID by default, matching container security best practice.

One shared, parameterizable Dockerfile pattern covers every Go service this platform ships (per 0002, the backend was chosen as Go specifically for "small container images, low resource footprint, and strong concurrency"):

```
# build stage
FROM golang:<pinned-version> AS build
...
RUN go build -trimpath -ldflags="-s -w" -o /out/<binary> ./cmd/<binary>

# runtime stage
FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/<binary> /<binary>
ENTRYPOINT ["/<binary>"]
```

Whether the crawler, catalog API, and search sync end up as one binary with multiple `cmd/` entrypoints, or as separate Go modules entirely, is left open (see Open questions below); this pattern works either way, one Dockerfile per binary, or one parameterized by a `BINARY` build argument.

## docker-compose.yml design

Lives at the repo root, so a self-hoster can run `docker compose up` right after cloning, with no flags.

Services:

- `backend`, a generic name for the endonend platform's container(s). At MVP scale this may be a single combined service; nothing here prevents splitting it into separate `crawler`, `api`, and `search-sync` services later without changing the underlying image.
- `postgres`, the official pinned Postgres image, the source of truth per 0002's "Persistence layer" section.
- `meilisearch`, the official pinned Meilisearch image, kept in sync from Postgres per the same section.

`backend` declares `depends_on` with `condition: service_healthy` against both `postgres` and `meilisearch`, so it never races a dependency that isn't ready yet; `postgres` and `meilisearch` each get a healthcheck (`pg_isready`, and Meilisearch's own `/health` endpoint).

Named volumes, `postgres-data` and `meilisearch-data`, persist both data stores across `docker compose down` (not `down -v`).

Configuration flows through a `.env` file, with `.env.example` committed and the real `.env` gitignored, matching 0002's "Configuration is via environment variables or mounted config files" line.

Compose targets a single host running one, unscaled replica of each service, a small self-run endonend instance or local testing. The Helm chart below targets larger-scale or cloud-orchestrated deployment instead (multiple replicas, resource limits, ingress, rolling upgrades). The two never diverge on what runs inside a container, only on how it's orchestrated: both consume the exact same published images.

## Helm chart structure

Lives at `infra/helm/endonend-platform/`, using the `infra/` directory already set aside for deployment and infrastructure config, and named for the "endonend platform" term 0001 and 0002 already use throughout.

```
infra/helm/endonend-platform/
  Chart.yaml
  values.yaml
  templates/
    _helpers.tpl
    backend-deployment.yaml
    backend-service.yaml
    backend-configmap.yaml
    backend-secret.yaml
    ingress.yaml
    meilisearch-deployment.yaml
    meilisearch-service.yaml
    meilisearch-pvc.yaml
    NOTES.txt
  README.md
```

Postgres is a chart *dependency* (for example, the Bitnami `postgresql` subchart), not a hand-rolled template: production-grade Postgres on Kubernetes, replication, backups, persistent volume claims, is a solved problem elsewhere, and this platform gains nothing by reimplementing it. Meilisearch has no equivalently established community chart, so it gets a small hand-rolled Deployment, PersistentVolumeClaim, and Service within this chart, consistent with 0002 describing it as "a single lightweight binary/container to run."

`values.yaml` templates: image repository and tag; replica count per backend component; resource requests and limits, left as commented-out placeholders since exact sizing needs real workload data (see Open questions); the Postgres subchart's passthrough values; the Meilisearch PVC size; environment configuration via a ConfigMap for non-secret values and a Secret reference for credentials, never inlined plaintext; and an ingress block whose controller and TLS strategy are left as a later decision.

The chart never builds a new image; it only references the tags the CI pipeline (see below) already published. Compose and the chart are two consumers of one build output, not two things that can independently drift.

## Multi-arch OCI image publishing

Images publish to GitHub Container Registry (`ghcr.io/<org>/<image>`) as the initial registry: no extra account to set up for a GitHub-hosted open-source project already using GitHub Actions per 0002's CI/CD section, and native `GITHUB_TOKEN`/OIDC integration for both pushing and signing. This mirrors how 0002 already frames its GCP deployment as "initial," not permanent; the registry choice is revisitable (see Open questions).

Builds use `docker buildx build --platform linux/amd64,linux/arm64 --push`, matching 0002's "multi-arch (amd64/arm64)... platform can run on inexpensive hardware" commitment, for example arm64-class hardware a small self-run endonend instance might use.

Tagging scheme:

- On merge to `main`: a mutable `edge` tag, plus an immutable `sha-<short-git-sha>` tag for traceability and rollback.
- On a git tag matching semver (`v1.2.3`): the semver cascade `1.2.3`, `1.2`, `1`, plus `latest` pointed at the newest stable release.
- Pull request builds compile the image to validate the Dockerfile, but never push, matching 0002's "build and push... on merge and on tag," not on every pull request.

Self-hosters should pin a specific semver tag in their compose file or Helm `values.yaml` for anything beyond casual testing; `latest` and `edge` are mutable and can change under a running deployment without notice.

## Image signing

Images are signed with cosign's keyless signing: GitHub Actions' own OIDC identity token, Sigstore's Fulcio for short-lived certificate issuance, and Rekor as the public transparency log, rather than a long-lived cosign key pair stored as a repository secret. This is the same principle 0002 already applies to its GCP deployment ("authenticating via Workload Identity Federation rather than a long-lived service account key"), applied here to image signing instead: nothing static to leak or rotate.

Verifying a published image is a concrete command, cashing out the project's transparency principle for anyone pulling it:

```
cosign verify \
  --certificate-identity-regexp "^https://github.com/<org>/<repo>/.github/workflows/.*" \
  --certificate-oidc-issuer https://token.actions.githubusercontent.com \
  ghcr.io/<org>/<image>:<tag>
```

This confirms the image was built by this repository's own GitHub Actions workflow, not a compromised registry account or an unrelated image sharing the same name and tag, and that the signing event itself is logged in the public Rekor transparency log rather than resting on unverifiable trust. Documenting this exact command (or wrapping it in a script) means a self-hoster never has to learn cosign's flag syntax themselves.

## CI/CD pipeline shape

A new workflow, sibling to the existing `.github/workflows/go.yml` (which stays lint, vet, test, and build on every pull request, per `CLAUDE.md`'s Go section): described here in prose only, no workflow YAML is written as part of this document.

Triggers: push to `main` and push of a `v*` tag, not pull requests, matching 0002's "build and push... on merge and on tag."

Steps, in order: checkout; `docker/setup-qemu-action` and `docker/setup-buildx-action`; registry login against `ghcr.io` via `GITHUB_TOKEN` or OIDC, no separate personal access token; `docker/build-push-action` with `platforms: linux/amd64,linux/arm64` and tags computed per the scheme above (for example via `docker/metadata-action`); a cosign keyless signing step against the pushed image digest, using the job's own OIDC token (`id-token: write` permission).

This workflow is the seam a future GCP-deployment spec's own deploy job would consume, "deploying the same signed images the public registry already published, not a GCP-specific build," per 0002, without building that seam now. Like `go.yml`'s `matrix.module`, this workflow will need a `matrix.image` entry per containerized component once the backend module exists and if it ships as more than one image, mirroring the existing comment convention in `go.yml`.

## Forward reference: where the backend module lives

The future endonend platform backend lives under `api/` as its own Go module (`api/go.mod`, module path `endonend/api`), matching the convention `CLAUDE.md`'s Go section already documents for `cli/`. Its internal structure, one module or several, one `cmd/` entrypoint or many, is explicitly out of scope here and deferred to whatever spec introduces it. This section exists only so the Dockerfile pattern and CI pipeline above don't assume a layout that contradicts it.

## Open questions

- Still open: GCP-specific deployment (Cloud Run, Cloud SQL, Secret Manager, Workload Identity Federation), deferred to a dedicated future spec per 0002's existing "Initial deployment (GCP)" section.
- Still open: the backend module's own internal code structure, deferred to whatever spec introduces `api/`.
- Still open: exact resource requests and limits for the Helm chart and compose file, which need real workload data this project doesn't have yet.
- **Ingress and TLS termination strategy.** Resolved: ingress-controller-agnostic. The chart's `ingress.yaml` templates a standard Kubernetes `Ingress` resource with `values.yaml`-driven, optional `cert-manager` annotations (`cert-manager.io/cluster-issuer`, left blank/omitted by default), so it works unmodified with nginx-ingress, Traefik, or any other controller a self-hoster already runs, and cert-manager is a convenience a self-hoster opts into rather than a hard dependency.
- Still open, and intentionally left revisitable: whether `ghcr.io` remains the registry long-term, per `KB/README.md`'s definition of a Decision as one specific, revisitable choice.
- **Postgres subchart dependency and versioning.** Resolved: keeps depending on an external community subchart (for example Bitnami's `postgresql`), version-pinned to a specific chart version in `Chart.yaml`, not a floating range, matching the deliberate-bump precedent already set for `golangci-lint-action` in `.github/workflows/go.yml`. Bumping it is a deliberate, reviewed change, never an automatic floating upgrade.
- **One container image or three.** Resolved: one image for MVP, with the crawler, catalog API, and search sync as separate `cmd/` entrypoints in the single future `api/` module, matching this document's own compose section already describing `backend` as "may be a single combined service" at MVP scale. The Dockerfile's `BINARY` build argument (see Image build strategy above) already supports building any one of them from the same pattern, so splitting into separate images later is a CI/Helm change, not a code or protocol change.
- **Backup and restore strategy for the compose Postgres volume.** Resolved: no backup automation is built by the platform for compose. Self-run instances are pointed to a documented `pg_dump`-based example (a short script and a cron example, written alongside the actual `docker-compose.yml` implementation), the standard approach for a single-host Postgres container, rather than the platform inventing its own backup mechanism. The Helm chart's Postgres subchart may separately expose its own backup options (for example Bitnami's optional pgBackRest integration); using them is left to the operator, not required for MVP.

## References

- [0001-purpose.md](./0001-purpose.md): the container-based deployment and no-vendor-lock-in principles this document implements.
- [0002-architecture.md](./0002-architecture.md): the deployment architecture and CI/CD commitments this document makes concrete.
- [KB/README.md](./README.md): KB conventions this document follows.
