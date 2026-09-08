package main

import (
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"endonend/cli/internal/bandcamp"
	"endonend/cli/internal/manifest"
	"endonend/cli/internal/slug"
	"endonend/cli/internal/spinner"
)

type importBandcampArgs struct {
	albumURL     string
	identityURL  string
	contactEmail string
	identityType string
	sourcePath   string
	downloadDir  string
	skipDownload bool
}

// parseImportBandcampArgs requires flags before the positional album URL,
// standard Go flag.Parse behavior (it stops parsing at the first
// non-flag argument), documented in help.go's usage text.
func parseImportBandcampArgs(args []string) (importBandcampArgs, error) {
	fs := flag.NewFlagSet("import bandcamp", flag.ContinueOnError)
	url := fs.String("url", "", "Your own identity URL (where you will host these files)")
	email := fs.String("contact-email", "", "Contact email for validation notifications")
	idType := fs.String("type", "artist", "\"artist\" or \"label\"")
	source := fs.String("source", sourcePath, "Path to endonend.source.json to create or update")
	downloadDir := fs.String("download-dir", "bandcamp-import", "Local directory to download cover art and audio into")
	skipDownload := fs.Bool("skip-download", false, "Prefill metadata and placeholder URLs only, skip downloading files")
	if err := fs.Parse(args); err != nil {
		return importBandcampArgs{}, err
	}
	rest := fs.Args()
	if len(rest) != 1 {
		return importBandcampArgs{}, fmt.Errorf("expected exactly one <bandcamp-album-url> argument after any flags, got %d", len(rest))
	}
	return importBandcampArgs{
		albumURL: rest[0], identityURL: *url, contactEmail: *email,
		identityType: *idType, sourcePath: *source, downloadDir: *downloadDir, skipDownload: *skipDownload,
	}, nil
}

func cmdImport(args []string) int {
	if len(args) == 0 || args[0] != "bandcamp" {
		fmt.Fprintln(os.Stderr, "import: expected a source, e.g. \"import bandcamp <album-url>\"")
		return 1
	}
	return cmdImportBandcamp(args[1:])
}

func cmdImportBandcamp(args []string) int {
	parsed, err := parseImportBandcampArgs(args)
	if err != nil {
		fmt.Fprintln(os.Stderr, "import bandcamp:", err)
		return 1
	}
	if err := importBandcamp(parsed); err != nil {
		fmt.Fprintln(os.Stderr, "import bandcamp:", err)
		return 1
	}
	return 0
}

// importBandcamp fetches a Bandcamp album page, upserts it into
// parsed.sourcePath (creating it, with parsed's identity fields, if it
// doesn't exist yet), and downloads its assets locally unless
// parsed.skipDownload is set. Shared by the "import bandcamp" subcommand
// and the interactive menu's equivalent option.
func importBandcamp(parsed importBandcampArgs) error {
	sp := spinner.New(os.Stderr, fmt.Sprintf("Fetching %s...", parsed.albumURL))
	album, err := bandcamp.Fetch(http.DefaultClient, parsed.albumURL)
	if err != nil {
		sp.Stop()
		return err
	}
	sp.Update(fmt.Sprintf("Fetched %q by %s", album.Title, album.ArtistName))

	src := manifest.Source{}
	if existing := loadSourceFile(parsed.sourcePath); existing != nil {
		src = *existing
	} else {
		if parsed.identityURL == "" || parsed.contactEmail == "" {
			return fmt.Errorf("%s doesn't exist yet; an identity URL and contact email are required to create it", parsed.sourcePath)
		}
		name := album.ArtistName
		if name == "" {
			name = "Unknown Artist"
		}
		src.ManifestVersion = "1.0"
		src.Identity = manifest.SourceIdentity{Type: parsed.identityType, Name: name}
		src.Refresh = manifest.Refresh{TTLSeconds: 21600}
	}
	if parsed.identityURL != "" {
		src.Identity.URL = parsed.identityURL
	}
	if parsed.contactEmail != "" {
		src.Identity.ContactEmail = parsed.contactEmail
	}
	if src.Identity.URL == "" {
		return fmt.Errorf("no identity.url set for %s", parsed.sourcePath)
	}

	albumID := slug.Slugify(album.Title)
	if albumID == "" {
		albumID = "album"
	}
	remoteBase := strings.TrimRight(src.Identity.URL, "/") + "/albums/" + albumID
	localDir := filepath.Join(parsed.downloadDir, albumID)

	upsertAlbum(&src, manifest.Album{
		AlbumID:      albumID,
		AlbumVersion: 1,
		AlbumName:    album.Title,
		ReleaseDate:  album.ReleaseDate,
		Images: manifest.Images{
			Front:  remoteBase + "/cover.jpg",
			Back:   remoteBase + "/cover.jpg",
			Insert: []string{},
		},
		Tracks: buildTracksFromBandcamp(album, albumID, remoteBase),
	})

	if !parsed.skipDownload {
		if err := downloadAlbumAssets(album, localDir, sp); err != nil {
			sp.Stop()
			return err
		}
	}
	sp.Stop()

	if err := writeSourceFile(parsed.sourcePath, &src); err != nil {
		return err
	}

	printImportSummary(album, parsed, localDir, remoteBase)
	return nil
}

func menuImportBandcamp(p *prompter) {
	albumURL := p.askRequired("\nBandcamp album URL (e.g. https://yourband.bandcamp.com/album/x)")

	parsed := importBandcampArgs{
		albumURL:    albumURL,
		sourcePath:  sourcePath,
		downloadDir: "bandcamp-import",
	}
	if loadExistingSource() == nil {
		fmt.Printf("No %s found yet; I need a few details to create one.\n", sourcePath)
		parsed.identityType = p.ask("Are you an artist or a label? [artist/label]", "artist")
		parsed.identityURL = p.askRequired("What URL will you publish your manifest under?")
		parsed.contactEmail = p.askRequired("Contact email for validation notifications?")
	}
	parsed.skipDownload = !p.askYesNo("Download the cover art and audio locally now?", true)

	fmt.Println()
	if err := importBandcamp(parsed); err != nil {
		fmt.Println("Error:", err)
	}
}

func upsertAlbum(src *manifest.Source, album manifest.Album) {
	for i, a := range src.Catalog {
		if a.AlbumID == album.AlbumID {
			src.Catalog[i] = album
			return
		}
	}
	src.Catalog = append(src.Catalog, album)
}

func trackFilename(t bandcamp.Track) string {
	return fmt.Sprintf("%02d-%s.mp3", t.Number, slug.Slugify(t.Title))
}

func buildTracksFromBandcamp(album *bandcamp.Album, albumID, remoteBase string) []manifest.Track {
	tracks := make([]manifest.Track, len(album.Tracks))
	for i, t := range album.Tracks {
		tracks[i] = manifest.Track{
			TrackID:  fmt.Sprintf("%s-t%d", albumID, t.Number),
			Number:   strconv.Itoa(t.Number),
			Name:     t.Title,
			Duration: bandcamp.FormatDuration(t.Duration),
			File:     remoteBase + "/" + trackFilename(t),
		}
	}
	return tracks
}

// maxDownloadBytes caps a single downloaded file, generous for a cover
// image or one compressed audio track while still bounding a runaway
// response.
const maxDownloadBytes = 200 << 20

func downloadAlbumAssets(album *bandcamp.Album, localDir string, sp *spinner.Spinner) error {
	if err := os.MkdirAll(localDir, 0o755); err != nil {
		return fmt.Errorf("create %s: %w", localDir, err)
	}
	if album.ArtURL != "" {
		sp.Update("Downloading cover art...")
		if err := downloadFile(album.ArtURL, filepath.Join(localDir, "cover.jpg")); err != nil {
			return fmt.Errorf("download cover art: %w", err)
		}
	}
	for i, t := range album.Tracks {
		if t.StreamURL == "" {
			continue
		}
		sp.Update(fmt.Sprintf("Downloading track %d/%d: %s...", i+1, len(album.Tracks), t.Title))
		if err := downloadFile(t.StreamURL, filepath.Join(localDir, trackFilename(t))); err != nil {
			return fmt.Errorf("download %q: %w", t.Title, err)
		}
	}
	return nil
}

func downloadFile(url, dest string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status %s fetching %s", resp.Status, url)
	}
	f, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()
	_, err = io.Copy(f, io.LimitReader(resp.Body, maxDownloadBytes))
	return err
}

func printImportSummary(album *bandcamp.Album, parsed importBandcampArgs, localDir, remoteBase string) {
	fmt.Printf("Imported %q by %s (%d tracks) into %s.\n", album.Title, album.ArtistName, len(album.Tracks), parsed.sourcePath)
	if parsed.skipDownload {
		fmt.Println("Skipped downloading audio/art (--skip-download); the source file's URLs are placeholders you still need to create.")
	} else {
		fmt.Printf("Downloaded cover art and audio (128kbps streams, not your masters) to %s.\n", localDir)
	}
	fmt.Printf("Before running generate, upload those files to %s so they match what endonend.source.json now declares.\n", remoteBase)
	fmt.Println("Bandcamp doesn't expose a separate back cover, so images.front and images.back both point at the same cover.jpg; replace images.back if you have a real one.")
}
