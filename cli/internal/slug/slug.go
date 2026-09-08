// Package slug turns free-text titles into stable, filesystem- and
// URL-safe identifiers, used for deriving albumId/trackId values and
// filenames when importing a catalog from an external source.
package slug

import (
	"regexp"
	"strings"
)

var (
	nonAlnum = regexp.MustCompile(`[^a-z0-9]+`)
	trimDash = regexp.MustCompile(`^-+|-+$`)
)

// Slugify lowercases s and replaces every run of non-alphanumeric
// characters with a single hyphen, e.g. "PARACIDIC" -> "paracidic" and
// "Anything, Anything!" -> "anything-anything".
func Slugify(s string) string {
	lower := strings.ToLower(s)
	hyphenated := nonAlnum.ReplaceAllString(lower, "-")
	return trimDash.ReplaceAllString(hyphenated, "")
}
