// Package bandcamp reads the public data a Bandcamp album page already
// embeds in its own HTML (title, tracklist, durations, release date, cover
// art) so it can seed a union.source.json, per
// KB/0005-bandcamp-import.md. It never writes a bandcamp.com or
// bcbits.com URL into a manifest: see that doc for why.
package bandcamp

import (
	"encoding/json"
	"fmt"
	"html"
	"io"
	"net/http"
	"regexp"
	"time"
)

type Track struct {
	Number    int
	Title     string
	Duration  time.Duration
	StreamURL string
}

type Album struct {
	ArtistName string
	Title      string
	// ReleaseDate is an ISO 8601 date (e.g. "2017-09-27"), matching
	// KB/0003-manifest.md's catalog[].releaseDate.
	ReleaseDate string
	ArtURL      string
	Tracks      []Track
}

var (
	tralbumAttr = regexp.MustCompile(`data-tralbum="([^"]*)"`)
	ogImageTag  = regexp.MustCompile(`<meta property="og:image" content="([^"]*)"`)
)

// tralbumJSON is the small subset of Bandcamp's data-tralbum blob this
// package actually needs; every other field is ignored.
type tralbumJSON struct {
	Artist  string `json:"artist"`
	Current struct {
		Title       string `json:"title"`
		ReleaseDate string `json:"release_date"`
	} `json:"current"`
	TrackInfo []struct {
		Title    string            `json:"title"`
		TrackNum int               `json:"track_num"`
		Duration float64           `json:"duration"`
		File     map[string]string `json:"file"`
	} `json:"trackinfo"`
}

// bandcampReleaseDateLayout matches Bandcamp's own release_date rendering,
// e.g. "27 Sep 2017 08:39:16 GMT".
const bandcampReleaseDateLayout = "02 Jan 2006 15:04:05 MST"

// Fetch downloads a Bandcamp album page and parses it.
func Fetch(client *http.Client, albumURL string) (*Album, error) {
	if client == nil {
		client = http.DefaultClient
	}
	resp, err := client.Get(albumURL)
	if err != nil {
		return nil, fmt.Errorf("fetch %s: %w", albumURL, err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fetch %s: unexpected status %s", albumURL, resp.Status)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", albumURL, err)
	}
	return Parse(body)
}

// Parse extracts an Album from a Bandcamp album page's raw HTML.
func Parse(pageHTML []byte) (*Album, error) {
	m := tralbumAttr.FindSubmatch(pageHTML)
	if m == nil {
		return nil, fmt.Errorf("could not find album data on this page; is it a Bandcamp album URL?")
	}
	var raw tralbumJSON
	if err := json.Unmarshal([]byte(html.UnescapeString(string(m[1]))), &raw); err != nil {
		return nil, fmt.Errorf("parse album data: %w", err)
	}
	if len(raw.TrackInfo) == 0 {
		return nil, fmt.Errorf("no tracks found on this album page")
	}

	album := &Album{
		ArtistName: raw.Artist,
		Title:      raw.Current.Title,
	}
	if t, err := time.Parse(bandcampReleaseDateLayout, raw.Current.ReleaseDate); err == nil {
		album.ReleaseDate = t.Format("2006-01-02")
	}
	if og := ogImageTag.FindSubmatch(pageHTML); og != nil {
		album.ArtURL = html.UnescapeString(string(og[1]))
	}

	for _, t := range raw.TrackInfo {
		album.Tracks = append(album.Tracks, Track{
			Number:    t.TrackNum,
			Title:     t.Title,
			Duration:  time.Duration(t.Duration * float64(time.Second)),
			StreamURL: t.File["mp3-128"],
		})
	}
	return album, nil
}

// FormatDuration renders a display-only duration hint in the style
// KB/0003-manifest.md's example manifest uses, e.g. 3m30s -> `3'30"`.
func FormatDuration(d time.Duration) string {
	total := int(d.Round(time.Second).Seconds())
	return fmt.Sprintf(`%d'%02d"`, total/60, total%60)
}
