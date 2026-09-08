package main

import "fmt"

const usage = `endtoend-artist-cli: generate, sign, and validate a Union manifest.

Running with no arguments launches an interactive menu.

Usage:
  endtoend-artist-cli                              Launch the interactive menu.
  endtoend-artist-cli generate [flags]              Write a freshly signed manifest.json and history.json.
  endtoend-artist-cli validate <path-or-url> [flags] Validate a manifest.
  endtoend-artist-cli keygen [flags]                Generate or rotate a signing key.
  endtoend-artist-cli help                          Show this message.

generate flags:
  --source string        Path to union.source.json (default "union.source.json")
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

Exit codes: 0 on success or a valid manifest, 1 on failure or an invalid manifest.
`

func printHelp() {
	fmt.Print(usage)
}
