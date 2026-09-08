package manifest

import (
	"encoding/json"
	"testing"
)

// exampleManifestJSON mirrors the example manifest in
// KB/0003-manifest.md, used to catch json-tag typos across the whole
// struct tree in one round trip.
const exampleManifestJSON = `{
  "manifestVersion": "1.0",
  "identity": {
    "type": "artist",
    "name": "Ligatures",
    "url": "https://ligatures.example",
    "publicKey": "ed25519:AbCdEf1234567890...",
    "contactEmail": "band@ligatures.example"
  },
  "refresh": { "ttlSeconds": 21600 },
  "label": {
    "affiliatedLabel": "https://smalllabel.example/.well-known/endonend/manifest.json",
    "split": { "artist": 85, "label": 15 }
  },
  "beacon": { "url": "https://ligatures.example/plays" },
  "merch": [ { "label": "Official Store", "url": "https://ligatures.example/store" } ],
  "history": { "url": "https://ligatures.example/.well-known/endonend/history.json", "headHash": "sha256:9f8e7d..." },
  "presentation": {
    "colors": { "primary": "#5B96C7", "heroBackground": "#5B96C7", "background": "#1a1a1a", "text": "#e0e0e0" },
    "links": { "bandcamp": "https://ligatures.bandcamp.com/" },
    "footer": "Independently released."
  },
  "catalog": [
    {
      "albumId": "agency-2024",
      "albumVersion": 1,
      "albumName": "Agency",
      "pageTitle": "Ligatures - Agency",
      "releaseDate": "2024-05-01",
      "images": { "front": "https://x/front.png", "back": "https://x/back.png", "insert": [] },
      "downloadZip": "https://x/agency.zip",
      "purchaseLinks": [ { "format": "vinyl", "url": "https://x/vinyl" } ],
      "credits": { "paragraphs": [ "Written by Ligatures.", [ { "text": "Mastered at " }, { "text": "Cool Studio", "url": "https://coolstudio.example" } ] ] },
      "splits": [ { "manifestUrl": "https://ligatures.example/.well-known/endonend/manifest.json", "role": "primary", "percentage": 100 } ],
      "tracks": [
        { "trackId": "a1", "side": "A", "number": "A1", "name": "Opening", "duration": "3'30\"", "file": "https://x/opening.mp3", "lyrics": "First\nSecond" }
      ]
    }
  ],
  "signature": { "algorithm": "ed25519", "value": "base64signature" }
}`

func TestManifest_RoundTripsExampleJSON(t *testing.T) {
	var m Manifest
	if err := json.Unmarshal([]byte(exampleManifestJSON), &m); err != nil {
		t.Fatalf("Unmarshal returned error: %v", err)
	}

	checks := []struct {
		name string
		got  any
		want any
	}{
		{"identity.type", m.Identity.Type, "artist"},
		{"identity.publicKey", m.Identity.PublicKey, "ed25519:AbCdEf1234567890..."},
		{"refresh.ttlSeconds", m.Refresh.TTLSeconds, 21600},
		{"label.affiliatedLabel", m.Label.AffiliatedLabel, "https://smalllabel.example/.well-known/endonend/manifest.json"},
		{"label.split.artist", m.Label.Split.Artist, float64(85)},
		{"beacon.url", m.Beacon.URL, "https://ligatures.example/plays"},
		{"merch[0].label", m.Merch[0].Label, "Official Store"},
		{"history.headHash", m.History.HeadHash, "sha256:9f8e7d..."},
		{"presentation.colors.primary", m.Presentation.Colors.Primary, "#5B96C7"},
		{"presentation.links.bandcamp", m.Presentation.Links["bandcamp"], "https://ligatures.bandcamp.com/"},
		{"catalog[0].albumId", m.Catalog[0].AlbumID, "agency-2024"},
		{"catalog[0].images.insert (non-nil empty)", m.Catalog[0].Images.Insert == nil, false},
		{"catalog[0].purchaseLinks[0].format", m.Catalog[0].PurchaseLinks[0].Format, "vinyl"},
		{"catalog[0].splits[0].percentage", m.Catalog[0].Splits[0].Percentage, float64(100)},
		{"catalog[0].tracks[0].lyrics", m.Catalog[0].Tracks[0].Lyrics, "First\nSecond"},
		{"signature.algorithm", m.Signature.Algorithm, "ed25519"},
	}
	for _, c := range checks {
		if c.got != c.want {
			t.Errorf("%s = %v, want %v", c.name, c.got, c.want)
		}
	}

	if len(m.Catalog[0].Credits.Paragraphs) != 2 {
		t.Fatalf("got %d credits paragraphs, want 2", len(m.Catalog[0].Credits.Paragraphs))
	}
	if m.Catalog[0].Credits.Paragraphs[0].PlainText != "Written by Ligatures." {
		t.Errorf("credits paragraph 0 = %+v", m.Catalog[0].Credits.Paragraphs[0])
	}
	if len(m.Catalog[0].Credits.Paragraphs[1].Segments) != 2 {
		t.Errorf("credits paragraph 1 segments = %+v, want 2", m.Catalog[0].Credits.Paragraphs[1].Segments)
	}

	// Re-marshal and unmarshal again to confirm the struct's own
	// MarshalJSON output is itself valid input.
	raw, err := json.Marshal(&m)
	if err != nil {
		t.Fatalf("re-Marshal returned error: %v", err)
	}
	var again Manifest
	if err := json.Unmarshal(raw, &again); err != nil {
		t.Fatalf("re-Unmarshal returned error: %v", err)
	}
	if again.Identity.Name != m.Identity.Name {
		t.Errorf("round trip lost identity.name: got %q, want %q", again.Identity.Name, m.Identity.Name)
	}
}

func TestSource_OmitsToolManagedFields(t *testing.T) {
	src := Source{
		ManifestVersion: "1.0",
		Identity: SourceIdentity{
			Type: "artist", Name: "Ligatures", URL: "https://ligatures.example", ContactEmail: "band@ligatures.example",
		},
		Refresh: Refresh{TTLSeconds: 21600},
	}
	raw, err := json.Marshal(src)
	if err != nil {
		t.Fatalf("Marshal returned error: %v", err)
	}
	var generic map[string]any
	if err := json.Unmarshal(raw, &generic); err != nil {
		t.Fatalf("Unmarshal returned error: %v", err)
	}
	identity, ok := generic["identity"].(map[string]any)
	if !ok {
		t.Fatalf("identity missing or wrong type: %v", generic["identity"])
	}
	if _, present := identity["publicKey"]; present {
		t.Error("Source JSON must not carry identity.publicKey, it's tool-managed")
	}
	if _, present := generic["signature"]; present {
		t.Error("Source JSON must not carry signature, it's tool-managed")
	}
}
