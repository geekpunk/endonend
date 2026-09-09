# endonend-artist-cli

```
Status: Proposed
Date: 2026-09-07
Type: Spec
References: [0001-purpose.md](./0001-purpose.md), [0002-architecture.md](./0002-architecture.md), [0003-manifest.md](./0003-manifest.md)
```

## Overview

`endonend-artist-cli` is the manifest-generator/signer CLI tool committed to in [0002-architecture.md](./0002-architecture.md): a single static Go binary, not a service, that an artist or label runs locally to create, update, sign, and validate their manifest. It imports the same Go validation package endonend's crawler and submission endpoint use, so "valid per this tool" and "valid per the platform" can never quietly drift apart, the same "one shared backend validator" commitment 0002 already makes.

## Implementation status

A first working version exists under `cli/` in this repository: real ed25519 key generation, RFC 8785 signing and verification, every validation rule below including the algorithm allowlist and the history hash chain, `--deep` network mutual-attestation checks, and an interactive menu. Two scope choices from that first pass, worth knowing before reading the rest of this doc as if every field were interactive:

- The menu prompts for the core fields (identity, contact, key bootstrap, label affiliation and split, albums and tracks), not every optional field in 0003 (`merch`, `purchaseLinks`, per-album `presentation`, `insert` images, `contributions`). Those remain fully supported in `endonend.source.json`, just hand-edited rather than prompted for, answering the "how much can an advanced user skip" open question below in the most direct way: everything past the core fields, today.
- Cross-run key pinning (remembering the key last seen for an `identity.url` across separate invocations) isn't implemented here, correctly: it needs persistent state this stateless local tool doesn't have, and belongs to endonend's crawler (0002) instead. This tool does still check that any `key_rotated` entry already in a manifest's history is properly signed by the key it retires, since that's a self-contained structural check.

## Key design decisions

**A source config, not hand-edited output.** The manifest defined in [0003-manifest.md](./0003-manifest.md) is signed; hand-editing `manifest.json` directly would invalidate its signature on every change and put history-log bookkeeping on the artist. Instead, the tool reads and writes a separate, human-friendly source file, `endonend.source.json`, the same content shape as the manifest minus `signature`, `history.headHash`, and `identity.publicKey`, all three tool-managed, not something an artist types in by hand, the key least of all. Generating diffs the new source against the previously generated manifest, if one exists at the output path, to compute the right history entries automatically, then writes a freshly signed `manifest.json` and an updated `history.json`. The artist is never asked to write a signature or a history entry by hand.

**An interactive menu as the primary interface, flags for automation.** Running `endonend-artist-cli` with no arguments launches a guided, question-by-question menu. This is the default experience, in keeping with "artists do not need to be technologists." The same functionality is also reachable through direct subcommands and flags, since the tool is also expected to run as a non-interactive CI step (see 0002's CI/CD section). One shared Go package implements the actual generate and validate logic; the menu and the flag-driven commands are both thin frontends onto it.

## Interactive menu

Running the binary with no arguments shows a top-level menu:

```
endonend-artist-cli
1) Create or update your manifest
2) Validate a manifest
3) Manage your signing key
4) Import an album from Bandcamp
5) Publish to GitHub Pages
6) Help
7) Exit
```

Option 6 prints the same usage text as `endonend-artist-cli help`, so help is identical from either interface.

**1) Create or update your manifest.** Asks plain-language questions in sequence: artist or label? name? what URL will you publish under (a domain you own, or a path on shared hosting like GitHub Pages)? contact email? If no local key exists yet, one is generated automatically at this point, with a short plain-language explanation of what just happened and why. It then walks through the catalog: add an album? title, release date, a front cover image (a back cover is asked for too, but optional, since not every release has a distinct one), an insert or two if there are any, add a track? (repeating per track, then offering to add another album), label affiliation and this release's split, merch links, and presentation (colors, links, footer), offering sensible defaults at each step rather than demanding every field. It shows a summary before writing anything, then writes `endonend.source.json` and runs the same logic as `generate`, producing a freshly signed `manifest.json` and updated `history.json`.

**2) Validate a manifest.** Asks for a local file path or a URL, then asks whether to run the deep check (fetches cross-referenced manifests to verify mutual attestation on label affiliation and album splits) or stay offline (schema, types, signature, and structural rules only). Prints a plain-language pass/fail report, not a raw error dump, each failure named in terms of what the artist should actually go fix.

**3) Manage your signing key.** View the current public key (to paste into a label's roster confirmation, for example), or rotate it. Rotating signs a `key_rotated` history entry with the *old* key before switching to the new one, per 0003's Key rotation mechanism, so the platform can tell this was an authorized handoff rather than a hijack. The tool warns plainly that this only works while the old key is still present locally: if it's already lost, there is no rotation path, only starting a new identity and reaching out to affected parties directly.

**4) Import an album from Bandcamp.** Bootstraps or updates `endonend.source.json` from an existing Bandcamp album page's own metadata, instead of retyping a tracklist by hand. Fully specified in [0005-bandcamp-import.md](./0005-bandcamp-import.md), including why it downloads audio and art locally rather than pointing the manifest at Bandcamp's own hosting. Bandcamp exposes no separate back cover, so `images.back` is left unset (see the Images field's optionality above).

**5) Publish to GitHub Pages.** For an `identity.url` already set to a `*.github.io` URL, assembles the already-generated `manifest.json`, `history.json`, and every album asset the manifest declares (matched up against a local assets directory, `bandcamp-import` by default) into the exact directory shape that URL must serve, then creates the GitHub repo if it doesn't exist yet, pushes it, and enables GitHub Pages. Re-running it after a catalog update is a normal, idempotent push, not a fresh repo each time. Requires the `git` and `gh` CLIs, with `gh` already authenticated (`gh auth login`); this is the one place the tool shells out to external programs rather than doing everything itself, since re-implementing a GitHub client and a Git implementation isn't worth it for what's fundamentally "create a repo, push some files." A manifest whose `identity.url` isn't a GitHub Pages URL isn't something this option can help with; the artist's own host is still always in reach through the flag-driven commands and hand-uploading, exactly as before this option existed.

## Non-interactive commands

For scripting and CI, reusing the exact same underlying logic as the menu:

- `endonend-artist-cli generate --source endonend.source.json`: writes signed `manifest.json` and updated `history.json`.
- `endonend-artist-cli validate <path-or-url> [--deep] [--json]`: runs the shared validator. `--deep` adds the network mutual-attestation check; `--json` prints machine-readable output.
- `endonend-artist-cli keygen`: generates a fresh keypair (first-time use) or, with `--rotate`, retires the current key by signing a `key_rotated` history entry with it before switching to a newly generated one.
- `endonend-artist-cli publish github [--manifest <path>] [--history <path>] [--assets-dir <dir>] [--out-dir <dir>] [--branch <name>] [--private]`: the same GitHub Pages publish flow as the menu option, for scripting a re-publish after every catalog update.
- `endonend-artist-cli help` (and `--help` on any subcommand): usage, flags, and a one-line description of every command, including a note that running with no arguments launches the interactive menu.
- Exit codes: `0` on success or a valid manifest, `1` on failure or an invalid manifest, so `validate` can gate a CI pipeline.

## Key management

The private key lives locally (for example `~/.endonend/keys/<slug>.key`, where `<slug>` is a filesystem-safe encoding of `identity.url`), file mode `0600`, never written into the source config and never committed to version control; `init`-style flows add a `.gitignore` entry for the key directory automatically. The public key is what gets embedded in the manifest's `identity.publicKey`, per 0003.

## Validation, in detail

Both the menu's "Validate a manifest" option and the `validate` command run the same shared Go package, structured exactly as 0003's own "Validation rules" section:

- Schema and type checks: every required field from 0003 is present and correctly typed.
- Signature check: `identity.publicKey` is well-formed and `signature.value` verifies against the canonicalized manifest. Canonicalization is RFC 8785 (JCS), per 0003; this tool uses an existing Go JCS library rather than a hand-rolled implementation, so it stays byte-for-byte compatible with the Kotlin Multiplatform mobile module without either side maintaining its own canonicalization code.
- Structural rules: `label.split` and any album `splits` sum to 100; `images.front` is present (`images.back` is optional; `images.insert` may be empty but must exist as a field); `albumId` and `trackId` are unique; `albumVersion` only increases; every URL field is a well-formed `http`/`https` URL.
- History integrity: if `history.headHash` is present, the fetched `history.json` chain verifies unbroken up to it.
- Deep-only checks (require network access, so they're opt-in with `--deep` or the menu's prompt): label affiliation is verified only when the label's own `roster` matches; an album's `splits` entries are verified only when each contributing party's own `contributions` matches, per 0003's mutual-attestation rules.

A failed check is reported the way the shared package classifies it, not as a raw parser error: which field, what was expected, and for a deep-check failure, whether it's unverified (no confirmation yet) or disputed (a confirmation exists but disagrees).

## Distribution

Prebuilt, cross-compiled binaries (macOS, Linux, Windows; amd64, arm64) attached to GitHub releases, built by the same CI/CD pipeline already defined in 0002. Also installable via `go install` for anyone with a Go toolchain. No local toolchain is required to just download and run the binary, per "artists do not need to be technologists."

## Example session

Interactive, the primary path:

```
$ endonend-artist-cli
1) Create or update your manifest
2) Validate a manifest
3) Manage your signing key
4) Import an album from Bandcamp
5) Publish to GitHub Pages
6) Help
7) Exit
> 1

Are you an artist or a label? [artist/label]: artist
What's your artist name?: Ligatures
What URL will you publish your manifest under?: https://ligatures.example
Contact email for validation notifications?: band@ligatures.example

No signing key found for ligatures.example. Generating one now...
Done. Your public key is ed25519:AbCdEf1234567890...
Keep the private key at ~/.endonend/keys/ligatures.example.key safe and out of version control.

Add an album? [y/N]: y
  Album title: Agency
  Release date (YYYY-MM-DD): 2024-05-01
  ...
```

Non-interactive, the CI equivalent:

```
$ endonend-artist-cli validate ./manifest.json --deep --json
{"valid": true, "checks": 14, "warnings": []}
```

## Open questions

- **Proactive notification of mutually-attested parties after key rotation.** Resolved: yes, best-effort, on by default with an easy skip. The rotation mechanism itself (a `key_rotated` history entry signed by the old key) is defined in [0003-manifest.md](./0003-manifest.md)'s Key rotation section; this tool additionally reads the rotating identity's own manifest for parties it already names, `label.affiliatedLabel` on an artist manifest, or every entry in `label.roster` on a label manifest, fetches each one's current manifest, and reads their `identity.contactEmail`. Rather than sending email itself, which would make this "single static binary, not a service" tool need SMTP configuration, it prints a ready-to-send plain-language notice for each affected party (and opens a `mailto:` link pre-filled with that notice where the local environment supports it), leaving the actual send to the artist. A lookup failure for one party is reported and skipped, never blocks the rotation itself from completing.
- **Rate-limiting and caching for deep/network validation.** Resolved: within a single `validate --deep` run, the tool deduplicates by URL so a manifest cross-referenced by several `splits` or `contributions` entries is only fetched once, and reuses the same conditional-request (`ETag`/`If-None-Match`) caching the crawler already uses in [0002-architecture.md](./0002-architecture.md). Deep checks only ever follow one hop, the manifests this manifest directly cross-references, never recursively following those manifests' own cross-references, which bounds the fan-out without needing a persistent cross-run cache in this stateless local tool.
- **Menu prompt coverage for less-common fields.** Resolved for now: hand-editing `endonend.source.json` for `merch`, `purchaseLinks`, per-album `presentation`, `insert` images, and `contributions` stays the answer, per Implementation status above. Revisit only if real usage shows artists actually struggling with hand-editing these, not preemptively.
- **Publishing to static hosts other than GitHub Pages.** "Publish to GitHub Pages" only ever targets `*.github.io` URLs; an artist on object storage, Netlify, or their own server still uploads by hand, exactly as before this option existed. Whether this tool ever grows an equivalent for another host is left open; GitHub Pages was picked first because it needs no host account beyond GitHub itself, which `gh auth login` already assumes for anyone who can install the CLI's own releases.

## References

- [0001-purpose.md](./0001-purpose.md): the "artists do not need to be technologists" principle this tool exists to satisfy.
- [0002-architecture.md](./0002-architecture.md): the CLI's role in the shared-validator and CI/CD design.
- [0003-manifest.md](./0003-manifest.md): the manifest shape and validation rules this tool implements.
- [0005-bandcamp-import.md](./0005-bandcamp-import.md): the menu's "Import an album from Bandcamp" option and its flag-driven equivalent.
- [KB/README.md](./README.md): KB conventions this document follows.
