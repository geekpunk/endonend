// Package validate implements the shared validation logic described in
// KB/0004-endonend-artist-cli.md's "Validation, in detail" section, itself
// structured exactly as KB/0003-manifest.md's "Validation rules".
package validate

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"endonend/protocol/history"
	"endonend/protocol/manifest"
	"endonend/protocol/signing"
)

// Kind classifies a failure the way 0004 asks for: not a raw parser error,
// but named in terms of what the artist should go fix.
type Kind string

const (
	KindMissing    Kind = "missing"
	KindInvalid    Kind = "invalid"
	KindUnverified Kind = "unverified"
	KindDisputed   Kind = "disputed"
)

type Failure struct {
	Field    string `json:"field"`
	Kind     Kind   `json:"kind"`
	Expected string `json:"expected,omitempty"`
	Actual   string `json:"actual,omitempty"`
	Message  string `json:"message"`
}

type Report struct {
	Valid    bool      `json:"valid"`
	Checks   int       `json:"checks"`
	Failures []Failure `json:"failures,omitempty"`
	Warnings []string  `json:"warnings"`
}

type Options struct {
	Deep       bool
	HTTPClient *http.Client
}

func (o Options) client() *http.Client {
	if o.HTTPClient != nil {
		return o.HTTPClient
	}
	return &http.Client{Timeout: 10 * time.Second}
}

type source struct {
	report   *Report
	manifest *manifest.Manifest
	// rawJSON is the original manifest bytes, needed for presence checks
	// (e.g. images.insert) that Manifest's own struct tags can't
	// distinguish from a present-but-empty value once round-tripped.
	rawJSON []byte
	// selfURL is this manifest's own manifest URL, used when checking
	// deep, mutual-attestation entries elsewhere. For a URL target it's
	// the fetched URL; for a local file it's derived from identity.url.
	selfURL string
	history []manifest.HistoryEntry
	histErr error
}

// ValidatePath validates a local manifest.json file, looking for a sibling
// history.json alongside it.
func ValidatePath(path string, opts Options) (*Report, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	histPath := filepath.Join(filepath.Dir(path), "history.json")
	var entries []manifest.HistoryEntry
	var histErr error
	if histRaw, err := os.ReadFile(histPath); err != nil {
		histErr = err
	} else if err := json.Unmarshal(histRaw, &entries); err != nil {
		histErr = fmt.Errorf("parse %s: %w", histPath, err)
	}
	return validateBytes(raw, "", entries, histErr, opts)
}

// ValidateURL fetches a manifest and its paired history log over HTTP.
func ValidateURL(target string, opts Options) (*Report, error) {
	client := opts.client()
	raw, err := fetch(client, target)
	if err != nil {
		return nil, fmt.Errorf("fetch %s: %w", target, err)
	}
	var m manifest.Manifest
	var entries []manifest.HistoryEntry
	var histErr error
	if err := json.Unmarshal(raw, &m); err == nil && m.History.URL != "" {
		if histRaw, err := fetch(client, m.History.URL); err != nil {
			histErr = err
		} else if err := json.Unmarshal(histRaw, &entries); err != nil {
			histErr = fmt.Errorf("parse %s: %w", m.History.URL, err)
		}
	}
	return validateBytes(raw, target, entries, histErr, opts)
}

func fetch(client *http.Client, target string) ([]byte, error) {
	resp, err := client.Get(target)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status %s", resp.Status)
	}
	return io.ReadAll(resp.Body)
}

func validateBytes(raw []byte, fetchedFrom string, entries []manifest.HistoryEntry, histErr error, opts Options) (*Report, error) {
	r := &Report{Valid: true, Warnings: []string{}}
	var m manifest.Manifest
	if err := json.Unmarshal(raw, &m); err != nil {
		r.Valid = false
		r.Checks++
		r.Failures = append(r.Failures, Failure{
			Field: "$", Kind: KindInvalid,
			Message: fmt.Sprintf("not a valid manifest: %v", err),
		})
		return r, nil
	}

	s := &source{report: r, manifest: &m, rawJSON: raw, history: entries, histErr: histErr}
	if fetchedFrom != "" {
		s.selfURL = fetchedFrom
	} else {
		s.selfURL = m.Identity.URL + "/.well-known/endonend/manifest.json"
	}

	s.checkRequiredFields()
	s.checkURLFields()
	s.checkImages()
	s.checkFetchLocation(fetchedFrom)
	s.checkSignatureAlgorithmAllowlist()
	s.checkSignature()
	s.checkSplitsSumTo100()
	s.checkUniqueIDs()
	s.checkHistory()

	if opts.Deep {
		s.checkLabelAffiliation(opts)
		s.checkAlbumSplitAttestation(opts)
	}

	r.Valid = len(r.Failures) == 0
	return r, nil
}

func (s *source) fail(field string, kind Kind, format string, args ...any) {
	s.report.Failures = append(s.report.Failures, Failure{
		Field: field, Kind: kind, Message: fmt.Sprintf(format, args...),
	})
}

func isHTTPURL(s string) bool {
	u, err := url.Parse(s)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return false
	}
	return u.Scheme == "http" || u.Scheme == "https"
}

func (s *source) checkRequiredFields() {
	m := s.manifest
	s.report.Checks++
	if m.ManifestVersion == "" {
		s.fail("manifestVersion", KindMissing, "manifestVersion is required")
	} else if !strings.Contains(m.ManifestVersion, ".") {
		s.fail("manifestVersion", KindInvalid, "manifestVersion must be a \"major.minor\" string, got %q", m.ManifestVersion)
	} else if major := strings.SplitN(m.ManifestVersion, ".", 2)[0]; major != "1" {
		s.fail("manifestVersion", KindInvalid, "unsupported major manifest version %q", major)
	}

	s.report.Checks++
	if m.Identity.Type != "artist" && m.Identity.Type != "label" {
		s.fail("identity.type", KindInvalid, "identity.type must be \"artist\" or \"label\", got %q", m.Identity.Type)
	}
	s.report.Checks++
	if m.Identity.Name == "" {
		s.fail("identity.name", KindMissing, "identity.name is required")
	}
	s.report.Checks++
	if m.Identity.URL == "" {
		s.fail("identity.url", KindMissing, "identity.url is required")
	}
	s.report.Checks++
	if m.Identity.ContactEmail == "" {
		s.fail("identity.contactEmail", KindMissing, "identity.contactEmail is required")
	}
	s.report.Checks++
	if m.Refresh.TTLSeconds <= 0 {
		s.fail("refresh.ttlSeconds", KindMissing, "refresh.ttlSeconds is required and must be positive")
	}
	s.report.Checks++
	if m.History.URL == "" {
		s.fail("history.url", KindMissing, "history.url is required")
	}
	s.report.Checks++
	if m.Identity.Type == "artist" && len(m.Catalog) == 0 {
		s.fail("catalog", KindMissing, "artist manifests require a non-empty catalog")
	}
	s.report.Checks++
	for i, a := range m.Catalog {
		if len(a.Tracks) == 0 {
			s.fail(fmt.Sprintf("catalog[%d].tracks", i), KindMissing, "album %q has no tracks", a.AlbumID)
		}
	}
}

func (s *source) checkURLFields() {
	m := s.manifest
	check := func(field, value string) {
		if value == "" {
			return
		}
		s.report.Checks++
		if !isHTTPURL(value) {
			s.fail(field, KindInvalid, "%q is not a well-formed http(s) URL", value)
		}
	}
	check("identity.url", m.Identity.URL)
	check("history.url", m.History.URL)
	if m.Beacon != nil {
		check("beacon.url", m.Beacon.URL)
	}
	for i, mk := range m.Merch {
		check(fmt.Sprintf("merch[%d].url", i), mk.URL)
	}
	if m.Label != nil {
		check("label.affiliatedLabel", m.Label.AffiliatedLabel)
		for i, r := range m.Label.Roster {
			check(fmt.Sprintf("label.roster[%d].artistManifestUrl", i), r.ArtistManifestURL)
		}
	}
	for _, c := range m.Contributions {
		check("contributions[].manifestUrl", c.ManifestURL)
	}
	for _, links := range []map[string]string{presentationLinks(m.Presentation)} {
		for platform, l := range links {
			check(fmt.Sprintf("presentation.links.%s", platform), l)
		}
	}
	for ai, a := range m.Catalog {
		check(fmt.Sprintf("catalog[%d].images.front", ai), a.Images.Front)
		check(fmt.Sprintf("catalog[%d].images.back", ai), a.Images.Back)
		for ii, ins := range a.Images.Insert {
			check(fmt.Sprintf("catalog[%d].images.insert[%d]", ai, ii), ins)
		}
		check(fmt.Sprintf("catalog[%d].downloadZip", ai), a.DownloadZip)
		for pi, pl := range a.PurchaseLinks {
			check(fmt.Sprintf("catalog[%d].purchaseLinks[%d].url", ai, pi), pl.URL)
		}
		for si, sp := range a.Splits {
			check(fmt.Sprintf("catalog[%d].splits[%d].manifestUrl", ai, si), sp.ManifestURL)
		}
		for ti, t := range a.Tracks {
			check(fmt.Sprintf("catalog[%d].tracks[%d].file", ai, ti), t.File)
		}
		for platform, l := range presentationLinks(a.Presentation) {
			check(fmt.Sprintf("catalog[%d].presentation.links.%s", ai, platform), l)
		}
	}
}

func presentationLinks(p *manifest.Presentation) map[string]string {
	if p == nil {
		return nil
	}
	return p.Links
}

func (s *source) checkImages() {
	var raw map[string]any
	if err := json.Unmarshal(s.rawJSON, &raw); err != nil {
		return
	}
	catalog, _ := raw["catalog"].([]any)
	for i, ac := range catalog {
		album, _ := ac.(map[string]any)
		images, _ := album["images"].(map[string]any)
		s.report.Checks++
		if images == nil {
			s.fail(fmt.Sprintf("catalog[%d].images", i), KindMissing, "images object is required")
			continue
		}
		if _, ok := images["insert"]; !ok {
			s.fail(fmt.Sprintf("catalog[%d].images.insert", i), KindMissing, "images.insert must be present (an empty array is fine)")
		}
	}
}

func (s *source) checkFetchLocation(fetchedFrom string) {
	if fetchedFrom == "" {
		return
	}
	s.report.Checks++
	want := strings.TrimRight(s.manifest.Identity.URL, "/") + "/.well-known/endonend/manifest.json"
	if want != fetchedFrom {
		s.fail("identity.url", KindInvalid, "identity.url (%s) does not resolve to the URL this manifest was fetched from (%s)", s.manifest.Identity.URL, fetchedFrom)
	}
}

func (s *source) checkSignatureAlgorithmAllowlist() {
	s.report.Checks++
	if s.manifest.Signature.Algorithm != manifest.SignatureAlgorithmEd25519 {
		s.fail("signature.algorithm", KindInvalid, "unsupported signature algorithm %q; only %q is accepted", s.manifest.Signature.Algorithm, manifest.SignatureAlgorithmEd25519)
	}
	for i, e := range s.history {
		s.report.Checks++
		if e.Signature.Algorithm != manifest.SignatureAlgorithmEd25519 {
			s.fail(fmt.Sprintf("history[%d].signature.algorithm", i), KindInvalid, "unsupported signature algorithm %q; only %q is accepted", e.Signature.Algorithm, manifest.SignatureAlgorithmEd25519)
		}
	}
}

func (s *source) checkSignature() {
	s.report.Checks++
	pub, err := signing.DecodePublicKey(s.manifest.Identity.PublicKey)
	if err != nil {
		s.fail("identity.publicKey", KindInvalid, "%v", err)
		return
	}
	canonicalBytes, err := signing.CanonicalWithoutField(s.manifest, "signature")
	if err != nil {
		s.fail("signature.value", KindInvalid, "could not canonicalize manifest: %v", err)
		return
	}
	ok, err := signing.VerifyCanonical(pub, canonicalBytes, s.manifest.Signature.Value)
	if err != nil {
		s.fail("signature.value", KindInvalid, "%v", err)
		return
	}
	if !ok {
		s.fail("signature.value", KindInvalid, "signature does not verify against identity.publicKey")
	}
}

func splitSums100(vals []float64) bool {
	sum := 0.0
	for _, v := range vals {
		sum += v
	}
	return sum > 99.999 && sum < 100.001
}

func (s *source) checkSplitsSumTo100() {
	m := s.manifest
	if m.Label != nil {
		if m.Label.Split != nil {
			s.report.Checks++
			if !splitSums100([]float64{m.Label.Split.Artist, m.Label.Split.Label}) {
				s.fail("label.split", KindInvalid, "label.split must sum to 100")
			}
		}
		for i, r := range m.Label.Roster {
			s.report.Checks++
			if !splitSums100([]float64{r.Split.Artist, r.Split.Label}) {
				s.fail(fmt.Sprintf("label.roster[%d].split", i), KindInvalid, "roster split must sum to 100")
			}
		}
	}
	for ai, a := range m.Catalog {
		if len(a.Splits) == 0 {
			continue
		}
		s.report.Checks++
		vals := make([]float64, len(a.Splits))
		for i, sp := range a.Splits {
			vals[i] = sp.Percentage
		}
		if !splitSums100(vals) {
			s.fail(fmt.Sprintf("catalog[%d].splits", ai), KindInvalid, "album splits must sum to 100")
		}
	}
}

func (s *source) checkUniqueIDs() {
	albumIDs := map[string]bool{}
	trackIDs := map[string]bool{}
	s.report.Checks++
	for _, a := range s.manifest.Catalog {
		if albumIDs[a.AlbumID] {
			s.fail("catalog[].albumId", KindInvalid, "duplicate albumId %q", a.AlbumID)
		}
		albumIDs[a.AlbumID] = true
		for _, t := range a.Tracks {
			if trackIDs[t.TrackID] {
				s.fail("catalog[].tracks[].trackId", KindInvalid, "duplicate trackId %q", t.TrackID)
			}
			trackIDs[t.TrackID] = true
		}
	}
}

func (s *source) checkHistory() {
	s.report.Checks++
	if s.histErr != nil {
		s.fail("history", KindMissing, "could not load history log: %v", s.histErr)
		return
	}
	head, err := history.VerifyChain(s.history)
	if err != nil {
		s.fail("history", KindInvalid, "history chain is broken: %v", err)
		return
	}
	if s.manifest.History.HeadHash != "" && s.manifest.History.HeadHash != head {
		s.fail("history.headHash", KindInvalid, "manifest's history.headHash (%s) does not match the actual head of the log (%s)", s.manifest.History.HeadHash, head)
	}

	s.checkKeyRotationChain()
}

// checkKeyRotationChain verifies each history entry's signature against the
// key that should have been active when it was written, and that every
// key_rotated entry is signed by the key it retires. Since this tool keeps
// no persistent cross-run key pin (see KB/0004's "Implementation status"),
// the first key in effect is taken from the first key_rotated entry's
// oldPublicKey if one exists, or from identity.publicKey otherwise; the
// key in effect after the last entry must equal identity.publicKey.
func (s *source) checkKeyRotationChain() {
	if len(s.history) == 0 {
		return
	}
	s.report.Checks++

	currentKeyStr := s.manifest.Identity.PublicKey
	for _, e := range s.history {
		if e.Type == manifest.HistoryTypeKeyRotated {
			if old, ok := e.Data["oldPublicKey"].(string); ok {
				currentKeyStr = old
			}
			break
		}
	}

	for i, e := range s.history {
		if e.Type == manifest.HistoryTypeKeyRotated {
			oldStr, _ := e.Data["oldPublicKey"].(string)
			newStr, _ := e.Data["newPublicKey"].(string)
			if oldStr != currentKeyStr {
				s.fail(fmt.Sprintf("history[%d].data.oldPublicKey", i), KindInvalid,
					"key rotation's oldPublicKey (%s) does not match the key in effect at that point (%s)", oldStr, currentKeyStr)
			}
			oldPub, err := signing.DecodePublicKey(oldStr)
			if err != nil {
				s.fail(fmt.Sprintf("history[%d].data.oldPublicKey", i), KindInvalid, "%v", err)
			} else if canonicalBytes, err := signing.CanonicalWithoutField(e, "signature"); err == nil {
				ok, _ := signing.VerifyCanonical(oldPub, canonicalBytes, e.Signature.Value)
				if !ok {
					s.fail(fmt.Sprintf("history[%d].signature", i), KindInvalid,
						"key_rotated entry is not properly signed by the key it retires")
				}
			}
			currentKeyStr = newStr
			continue
		}

		pub, err := signing.DecodePublicKey(currentKeyStr)
		if err != nil {
			s.fail(fmt.Sprintf("history[%d].signature", i), KindInvalid, "%v", err)
			continue
		}
		canonicalBytes, err := signing.CanonicalWithoutField(e, "signature")
		if err != nil {
			continue
		}
		ok, _ := signing.VerifyCanonical(pub, canonicalBytes, e.Signature.Value)
		if !ok {
			s.fail(fmt.Sprintf("history[%d].signature", i), KindInvalid, "entry signature does not verify against the key in effect at that point")
		}
	}

	if currentKeyStr != s.manifest.Identity.PublicKey {
		s.fail("identity.publicKey", KindInvalid,
			"the key in effect at the end of the history log (%s) does not match identity.publicKey (%s); this may be an unauthorized key change",
			currentKeyStr, s.manifest.Identity.PublicKey)
	}
}

func (s *source) checkLabelAffiliation(opts Options) {
	if s.manifest.Label == nil || s.manifest.Label.AffiliatedLabel == "" {
		return
	}
	s.report.Checks++
	split := s.manifest.Label.Split
	if split == nil {
		s.fail("label.split", KindMissing, "label.split is required when affiliatedLabel is present")
		return
	}
	raw, err := fetch(opts.client(), s.manifest.Label.AffiliatedLabel)
	if err != nil {
		s.fail("label.affiliatedLabel", KindUnverified, "could not fetch label manifest: %v", err)
		return
	}
	var labelManifest manifest.Manifest
	if err := json.Unmarshal(raw, &labelManifest); err != nil {
		s.fail("label.affiliatedLabel", KindUnverified, "label manifest did not parse: %v", err)
		return
	}
	if labelManifest.Label == nil {
		s.fail("label.affiliatedLabel", KindUnverified, "label manifest declares no roster")
		return
	}
	for _, r := range labelManifest.Label.Roster {
		if r.ArtistManifestURL == s.selfURL {
			if r.Split == (manifest.Split{}) || (r.Split.Artist == split.Artist && r.Split.Label == split.Label) {
				return
			}
			s.fail("label.split", KindDisputed, "label's roster lists a split of %v/%v for this artist, this manifest declares %v/%v", r.Split.Artist, r.Split.Label, split.Artist, split.Label)
			return
		}
	}
	s.fail("label.affiliatedLabel", KindUnverified, "label's own roster does not list this artist back")
}

func (s *source) checkAlbumSplitAttestation(opts Options) {
	for ai, a := range s.manifest.Catalog {
		for si, sp := range a.Splits {
			s.report.Checks++
			raw, err := fetch(opts.client(), sp.ManifestURL)
			if err != nil {
				s.fail(fmt.Sprintf("catalog[%d].splits[%d]", ai, si), KindUnverified, "could not fetch contributing party's manifest: %v", err)
				continue
			}
			var other manifest.Manifest
			if err := json.Unmarshal(raw, &other); err != nil {
				s.fail(fmt.Sprintf("catalog[%d].splits[%d]", ai, si), KindUnverified, "contributing party's manifest did not parse: %v", err)
				continue
			}
			found := false
			for _, c := range other.Contributions {
				if c.ManifestURL != s.selfURL || c.AlbumID != a.AlbumID || c.AlbumVersion != a.AlbumVersion {
					continue
				}
				found = true
				if c.Role != sp.Role || c.Percentage != sp.Percentage {
					s.fail(fmt.Sprintf("catalog[%d].splits[%d]", ai, si), KindDisputed,
						"contributing party's confirmation (%s, %v%%) does not match this manifest's claim (%s, %v%%)", c.Role, c.Percentage, sp.Role, sp.Percentage)
				}
				break
			}
			if !found {
				s.fail(fmt.Sprintf("catalog[%d].splits[%d]", ai, si), KindUnverified, "no matching contributions entry found in %s", sp.ManifestURL)
			}
		}
	}
}

// Summary renders a plain-language pass/fail report, per 0004's "Prints a
// plain-language pass/fail report, not a raw error dump" requirement.
func (r *Report) Summary() string {
	var b strings.Builder
	if r.Valid {
		fmt.Fprintf(&b, "Valid manifest (%d checks passed)\n", r.Checks)
	} else {
		fmt.Fprintf(&b, "Invalid manifest: %d issue(s) found\n", len(r.Failures))
		for _, f := range r.Failures {
			fmt.Fprintf(&b, "  [%s] %s: %s\n", f.Kind, f.Field, f.Message)
		}
	}
	for _, w := range r.Warnings {
		fmt.Fprintf(&b, "  warning: %s\n", w)
	}
	return b.String()
}
