// Package manifest defines the JSON shapes described in KB/0003-manifest.md:
// the signed manifest, its paired history log, and the human-edited source
// file the CLI reads and writes instead of the signed manifest itself.
package manifest

// Identity describes who publishes a manifest.
type Identity struct {
	Type         string `json:"type"`
	Name         string `json:"name"`
	URL          string `json:"url"`
	PublicKey    string `json:"publicKey,omitempty"`
	ContactEmail string `json:"contactEmail"`
}

type Refresh struct {
	TTLSeconds int `json:"ttlSeconds"`
}

type Split struct {
	Artist float64 `json:"artist"`
	Label  float64 `json:"label"`
}

// RosterEntry is one artist a label manifest claims.
type RosterEntry struct {
	ArtistManifestURL string `json:"artistManifestUrl"`
	Split             Split  `json:"split"`
}

// Label carries either an artist's affiliation or a label's roster,
// depending on identity.type.
type Label struct {
	AffiliatedLabel string        `json:"affiliatedLabel,omitempty"`
	Split           *Split        `json:"split,omitempty"`
	Roster          []RosterEntry `json:"roster,omitempty"`
}

type Contribution struct {
	ManifestURL  string  `json:"manifestUrl"`
	AlbumID      string  `json:"albumId"`
	AlbumVersion int     `json:"albumVersion"`
	Role         string  `json:"role"`
	Percentage   float64 `json:"percentage"`
}

type Beacon struct {
	URL string `json:"url,omitempty"`
}

type MerchLink struct {
	Label string `json:"label"`
	URL   string `json:"url"`
}

type HistoryRef struct {
	URL      string `json:"url"`
	HeadHash string `json:"headHash,omitempty"`
}

type Colors struct {
	Primary        string `json:"primary,omitempty"`
	HeroBackground string `json:"heroBackground,omitempty"`
	Background     string `json:"background,omitempty"`
	Text           string `json:"text,omitempty"`
}

type Presentation struct {
	Colors *Colors           `json:"colors,omitempty"`
	Links  map[string]string `json:"links,omitempty"`
	Footer string            `json:"footer,omitempty"`
}

type Images struct {
	Front  string   `json:"front"`
	Back   string   `json:"back"`
	Insert []string `json:"insert"`
}

type AlbumSplitEntry struct {
	ManifestURL string  `json:"manifestUrl"`
	Role        string  `json:"role"`
	Percentage  float64 `json:"percentage"`
}

type PurchaseLink struct {
	Format string `json:"format"`
	URL    string `json:"url"`
}

// CreditSegment is one inline-linkable piece of a credits paragraph.
type CreditSegment struct {
	Text string `json:"text"`
	URL  string `json:"url,omitempty"`
}

// CreditParagraph is either a plain string or a list of segments; callers
// unmarshal into RawParagraph and interpret it based on its JSON kind.
type Credits struct {
	Paragraphs []RawParagraph `json:"paragraphs,omitempty"`
}

// RawParagraph preserves whatever JSON shape (string or segment array) a
// credits paragraph had, so generate/validate can inspect it without lossy
// normalization.
type RawParagraph struct {
	PlainText string
	Segments  []CreditSegment
}

type Track struct {
	TrackID  string `json:"trackId"`
	Side     string `json:"side,omitempty"`
	Number   string `json:"number"`
	Name     string `json:"name"`
	Duration string `json:"duration"`
	File     string `json:"file"`
	Lyrics   string `json:"lyrics,omitempty"`
}

type Album struct {
	AlbumID       string            `json:"albumId"`
	AlbumVersion  int               `json:"albumVersion"`
	AlbumName     string            `json:"albumName"`
	PageTitle     string            `json:"pageTitle,omitempty"`
	ReleaseDate   string            `json:"releaseDate"`
	Images        Images            `json:"images"`
	DownloadZip   string            `json:"downloadZip,omitempty"`
	Credits       *Credits          `json:"credits,omitempty"`
	Splits        []AlbumSplitEntry `json:"splits,omitempty"`
	PurchaseLinks []PurchaseLink    `json:"purchaseLinks,omitempty"`
	Presentation  *Presentation     `json:"presentation,omitempty"`
	Tracks        []Track           `json:"tracks"`
}

type Signature struct {
	Algorithm string `json:"algorithm"`
	Value     string `json:"value"`
}

// Manifest is the full signed manifest published at
// <identity.url>/.well-known/endonend/manifest.json.
type Manifest struct {
	ManifestVersion string         `json:"manifestVersion"`
	Identity        Identity       `json:"identity"`
	Refresh         Refresh        `json:"refresh"`
	Label           *Label         `json:"label,omitempty"`
	Contributions   []Contribution `json:"contributions,omitempty"`
	Beacon          *Beacon        `json:"beacon,omitempty"`
	Merch           []MerchLink    `json:"merch,omitempty"`
	History         HistoryRef     `json:"history"`
	Presentation    *Presentation  `json:"presentation,omitempty"`
	Catalog         []Album        `json:"catalog,omitempty"`
	Signature       Signature      `json:"signature"`
}

// Source is endonend.source.json: the same shape as Manifest minus the three
// tool-managed fields (identity.publicKey, history.headHash, signature),
// per KB/0004's "A source config, not hand-edited output" decision.
type Source struct {
	ManifestVersion string         `json:"manifestVersion"`
	Identity        SourceIdentity `json:"identity"`
	Refresh         Refresh        `json:"refresh"`
	Label           *Label         `json:"label,omitempty"`
	Contributions   []Contribution `json:"contributions,omitempty"`
	Beacon          *Beacon        `json:"beacon,omitempty"`
	Merch           []MerchLink    `json:"merch,omitempty"`
	History         *SourceHistory `json:"history,omitempty"`
	Presentation    *Presentation  `json:"presentation,omitempty"`
	Catalog         []Album        `json:"catalog,omitempty"`
}

type SourceIdentity struct {
	Type         string `json:"type"`
	Name         string `json:"name"`
	URL          string `json:"url"`
	ContactEmail string `json:"contactEmail"`
}

type SourceHistory struct {
	URL string `json:"url"`
}

// HistoryEntry is one append-only entry in history.json.
type HistoryEntry struct {
	Timestamp    string         `json:"timestamp"`
	Type         string         `json:"type"`
	Data         map[string]any `json:"data"`
	PreviousHash string         `json:"previousHash,omitempty"`
	Signature    Signature      `json:"signature"`
}

const (
	HistoryTypeIdentityUpdated = "identity_updated"
	HistoryTypeSplitChanged    = "split_changed"
	HistoryTypeReleaseAdded    = "release_added"
	HistoryTypeReleaseUpdated  = "release_updated"
	HistoryTypeReleaseRemoved  = "release_removed"
	HistoryTypeKeyRotated      = "key_rotated"
)

const SignatureAlgorithmEd25519 = "ed25519"
