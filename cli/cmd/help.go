package main

import "fmt"

const usage = `endonend-artist-cli: generate, sign, and validate an endonend manifest.

Running with no arguments launches an interactive menu.

Usage:
  endonend-artist-cli                              Launch the interactive menu.
  endonend-artist-cli generate [flags]              Write a freshly signed manifest.json and history.json.
  endonend-artist-cli validate <path-or-url> [flags] Validate a manifest.
  endonend-artist-cli keygen [flags]                Generate or rotate a signing key.
  endonend-artist-cli import bandcamp [flags] <url> Prefill endonend.source.json from a Bandcamp album page.
  endonend-artist-cli publish github [flags]        Push your manifest and assets to a GitHub Pages repo.
  endonend-artist-cli help                          Show this message.

generate flags:
  --source string        Path to endonend.source.json (default "endonend.source.json")
  --manifest-out string  Where to write the signed manifest (default "manifest.json")
  --history-out string   Where to write the history log (default "history.json")

validate flags:
  --deep   Also run network mutual-attestation checks against cross-referenced manifests.
  --json   Print machine-readable JSON instead of a plain-language report.

keygen flags:
  --url string            Identity URL to generate a first-time key for.
  --rotate                Rotate the key already in use, retiring it with a signed key_rotated history entry.
  --manifest-out string   Manifest to rotate, when --rotate is set (default "manifest.json")
  --history-out string    History log to update, when --rotate is set (default "history.json")

import bandcamp flags (flags must come before the album URL):
  --url string            Your own identity URL (required unless --source already exists)
  --contact-email string  Contact email (required unless --source already exists)
  --type string           "artist" or "label" (default "artist")
  --source string         endonend.source.json to create or update (default "endonend.source.json")
  --download-dir string   Where to save cover art and audio (default "bandcamp-import")
  --skip-download         Prefill metadata and placeholder URLs only, skip downloading files

publish github flags:
  --manifest string    Signed manifest to publish (default "manifest.json")
  --history string     Paired history log to publish (default "history.json")
  --assets-dir string  Local directory holding album assets, per <assets-dir>/<albumId>/<filename> (default "bandcamp-import")
  --out-dir string     Where the assembled, publish-ready site is written (default "publish")
  --branch string      Branch to push and serve GitHub Pages from (default "main")
  --private            Create the repo as private (GitHub Pages on it needs GitHub Pro or an org plan)

  Only publishes to a github.io URL: identity.url must already be set to
  where you want this hosted (e.g. https://you.github.io/yourband), same
  as any other identity.url. Requires the gh and git CLIs, with gh already
  authenticated (gh auth login).

Exit codes: 0 on success or a valid manifest, 1 on failure or an invalid manifest.
`

func printHelp() {
	fmt.Print(usage)
}
