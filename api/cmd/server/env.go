package main

import (
	"fmt"
	"os"
)

// requireEnv reads a required environment variable, returning an error
// (never exiting itself) so callers control how startup failure is
// reported. Per CLAUDE.md, a panic/fatal-style exit is reserved for main
// itself, at the one point that's actually unrecoverable.
func requireEnv(name string) (string, error) {
	v := os.Getenv(name)
	if v == "" {
		return "", fmt.Errorf("required environment variable %s is not set", name)
	}
	return v, nil
}

func envOr(name, fallback string) string {
	if v := os.Getenv(name); v != "" {
		return v
	}
	return fallback
}
