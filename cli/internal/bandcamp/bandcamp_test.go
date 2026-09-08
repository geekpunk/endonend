package bandcamp

import (
	"encoding/json"
	"fmt"
	"html"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// buildFixturePage renders a minimal Bandcamp-shaped page around the given
// tralbum JSON payload, HTML-attribute-encoded the same way Bandcamp's own
// server-rendered markup is (double quotes escaped, safe to re-extract with
// the package's own regexp + html.UnescapeString).
func buildFixturePage(t *testing.T, tralbum any, ogImage string) []byte {
	t.Helper()
	raw, err := json.Marshal(tralbum)
	if err != nil {
		t.Fatalf("marshal fixture tralbum: %v", err)
	}
	encoded := html.EscapeString(string(raw))
	page := fmt.Sprintf(`<!doctype html><html><head>
<meta property="og:image" content="%s">
</head><body>
<div data-tralbum="%s"></div>
</body></html>`, ogImage, encoded)
	return []byte(page)
}

func sampleTralbum() map[string]any {
	return map[string]any{
		"artist": "fatal flaw",
		"current": map[string]any{
			"title":        "Demo",
			"release_date": "27 Sep 2017 08:39:16 GMT",
		},
		"trackinfo": []map[string]any{
			{
				"title": "PARACIDIC", "track_num": 1, "duration": 178.559,
				"file": map[string]string{"mp3-128": "https://t4.bcbits.com/stream/aaa/mp3-128/1"},
			},
			{
				"title": "LOST AND FOUND", "track_num": 2, "duration": 219.734,
				"file": map[string]string{"mp3-128": "https://t4.bcbits.com/stream/bbb/mp3-128/2"},
			},
		},
	}
}

func TestParse_ExtractsAlbumAndTracks(t *testing.T) {
	page := buildFixturePage(t, sampleTralbum(), "https://f4.bcbits.com/img/a123_5.jpg")

	album, err := Parse(page)
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
	if album.ArtistName != "fatal flaw" || album.Title != "Demo" {
		t.Errorf("album = %+v, want artist \"fatal flaw\" and title \"Demo\"", album)
	}
	if album.ReleaseDate != "2017-09-27" {
		t.Errorf("ReleaseDate = %q, want %q", album.ReleaseDate, "2017-09-27")
	}
	if album.ArtURL != "https://f4.bcbits.com/img/a123_5.jpg" {
		t.Errorf("ArtURL = %q", album.ArtURL)
	}
	if len(album.Tracks) != 2 {
		t.Fatalf("got %d tracks, want 2", len(album.Tracks))
	}
	tr := album.Tracks[0]
	if tr.Number != 1 || tr.Title != "PARACIDIC" || tr.StreamURL != "https://t4.bcbits.com/stream/aaa/mp3-128/1" {
		t.Errorf("track 0 = %+v", tr)
	}
	if tr.Duration != time.Duration(178.559*float64(time.Second)) {
		t.Errorf("track 0 duration = %v, want ~178.559s", tr.Duration)
	}
}

func TestParse_MissingTralbumErrors(t *testing.T) {
	if _, err := Parse([]byte(`<html><body>nothing here</body></html>`)); err == nil {
		t.Error("Parse with no data-tralbum attribute: want error, got nil")
	}
}

func TestParse_MalformedJSONErrors(t *testing.T) {
	page := []byte(`<div data-tralbum="not json"></div>`)
	if _, err := Parse(page); err == nil {
		t.Error("Parse with malformed tralbum JSON: want error, got nil")
	}
}

func TestParse_NoTracksErrors(t *testing.T) {
	tralbum := sampleTralbum()
	tralbum["trackinfo"] = []map[string]any{}
	page := buildFixturePage(t, tralbum, "")
	if _, err := Parse(page); err == nil {
		t.Error("Parse with an empty trackinfo array: want error, got nil")
	}
}

func TestParse_UnparseableReleaseDateLeavesItEmptyWithoutError(t *testing.T) {
	tralbum := sampleTralbum()
	tralbum["current"].(map[string]any)["release_date"] = "not a date"
	page := buildFixturePage(t, tralbum, "")

	album, err := Parse(page)
	if err != nil {
		t.Fatalf("Parse returned error: %v", err)
	}
	if album.ReleaseDate != "" {
		t.Errorf("ReleaseDate = %q, want empty for an unparseable date", album.ReleaseDate)
	}
	if album.Title != "Demo" {
		t.Errorf("Title = %q, want the rest of the album to still parse", album.Title)
	}
}

func TestFetch_ParsesFromHTTPServer(t *testing.T) {
	page := buildFixturePage(t, sampleTralbum(), "https://f4.bcbits.com/img/a123_5.jpg")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(page)
	}))
	defer server.Close()

	album, err := Fetch(server.Client(), server.URL)
	if err != nil {
		t.Fatalf("Fetch returned error: %v", err)
	}
	if album.Title != "Demo" {
		t.Errorf("Title = %q, want \"Demo\"", album.Title)
	}
}

func TestFetch_NonOKStatusErrors(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	if _, err := Fetch(server.Client(), server.URL); err == nil {
		t.Error("Fetch against a 404: want error, got nil")
	}
}

func TestFormatDuration(t *testing.T) {
	cases := map[time.Duration]string{
		0:                             `0'00"`,
		30 * time.Second:              `0'30"`,
		90 * time.Second:              `1'30"`,
		3*time.Minute + 5*time.Second: `3'05"`,
	}
	for d, want := range cases {
		if got := FormatDuration(d); got != want {
			t.Errorf("FormatDuration(%v) = %q, want %q", d, got, want)
		}
	}
}
