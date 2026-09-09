package main

import (
	"bufio"
	"crypto/ed25519"
	"encoding/json"
	"fmt"
	"os"

	"endonend/cli/internal/generate"
	"endonend/cli/internal/spinner"
	"endonend/protocol/manifest"
	"endonend/protocol/signing"
	"endonend/protocol/validate"
)

const (
	sourcePath      = "endonend.source.json"
	manifestOutPath = "manifest.json"
	historyOutPath  = "history.json"
)

func runMenu() {
	p := newPrompter(bufio.NewReader(os.Stdin))
	for {
		fmt.Print(`
endonend-artist-cli
1) Create or update your manifest
2) Validate a manifest
3) Manage your signing key
4) Import an album from Bandcamp
5) Publish to GitHub Pages
6) Help
7) Exit
> `)
		switch p.line() {
		case "1":
			menuCreateOrUpdate(p)
		case "2":
			menuValidate(p)
		case "3":
			menuManageKey(p)
		case "4":
			menuImportBandcamp(p)
		case "5":
			menuPublishGithub(p)
		case "6":
			printHelp()
		case "7", "":
			return
		default:
			fmt.Println("Please choose 1-7.")
		}
	}
}

func loadExistingSource() *manifest.Source {
	return loadSourceFile(sourcePath)
}

func loadSourceFile(path string) *manifest.Source {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var src manifest.Source
	if err := json.Unmarshal(raw, &src); err != nil {
		return nil
	}
	return &src
}

func writeSourceFile(path string, src *manifest.Source) error {
	raw, err := json.MarshalIndent(src, "", "  ")
	if err != nil {
		return fmt.Errorf("encode %s: %w", path, err)
	}
	raw = append(raw, '\n')
	return os.WriteFile(path, raw, 0o644)
}

func menuCreateOrUpdate(p *prompter) {
	existing := loadExistingSource()
	src := manifest.Source{}
	if existing != nil {
		src = *existing
		fmt.Printf("\nFound an existing %s, using it as your defaults.\n\n", sourcePath)
	}

	typeDefault := src.Identity.Type
	if typeDefault == "" {
		typeDefault = "artist"
	}
	src.Identity.Type = p.ask("Are you an artist or a label? [artist/label]", typeDefault)
	src.Identity.Name = p.ask("What's your artist or label name?", src.Identity.Name)
	if src.Identity.Name == "" {
		src.Identity.Name = p.askRequired("What's your artist or label name?")
	}
	src.Identity.URL = p.ask("What URL will you publish your manifest under?", src.Identity.URL)
	if src.Identity.URL == "" {
		src.Identity.URL = p.askRequired("What URL will you publish your manifest under?")
	}
	src.Identity.ContactEmail = p.ask("Contact email for validation notifications?", src.Identity.ContactEmail)
	if src.Identity.ContactEmail == "" {
		src.Identity.ContactEmail = p.askRequired("Contact email for validation notifications?")
	}
	if src.ManifestVersion == "" {
		src.ManifestVersion = "1.0"
	}
	if src.Refresh.TTLSeconds == 0 {
		src.Refresh.TTLSeconds = p.askInt("Refresh TTL in seconds (how often should the crawler recheck?)", 21600)
	}

	keyPath, err := signing.KeyPath(src.Identity.URL)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	if !signing.KeyExists(keyPath) {
		fmt.Printf("\nNo signing key found for %s. Generating one now...\n", src.Identity.URL)
		pub, priv, err := signing.GenerateKeypair()
		if err != nil {
			fmt.Println("Error:", err)
			return
		}
		if err := signing.SavePrivateKey(keyPath, priv); err != nil {
			fmt.Println("Error:", err)
			return
		}
		fmt.Printf("Done. Your public key is %s\n", signing.EncodePublicKey(pub))
		fmt.Printf("Keep the private key at %s safe and out of version control.\n", keyPath)
	}

	if src.Identity.Type == "artist" {
		if p.askYesNo("\nSet or update label affiliation and split?", src.Label != nil && src.Label.AffiliatedLabel != "") {
			labelURL := ""
			if src.Label != nil {
				labelURL = src.Label.AffiliatedLabel
			}
			labelURL = p.ask("Label's manifest URL", labelURL)
			if labelURL == "" {
				src.Label = nil
			} else {
				artistPct := 85.0
				if src.Label != nil && src.Label.Split != nil {
					artistPct = src.Label.Split.Artist
				}
				artistPct = p.askFloat("Artist's percentage of the split", artistPct)
				src.Label = &manifest.Label{
					AffiliatedLabel: labelURL,
					Split:           &manifest.Split{Artist: artistPct, Label: 100 - artistPct},
				}
			}
		}

		fmt.Println("\nNow let's go through your catalog.")
		for p.askYesNo(fmt.Sprintf("Add an album? (%d so far)", len(src.Catalog)), len(src.Catalog) == 0) {
			album := menuAddAlbum(p)
			src.Catalog = append(src.Catalog, album)
		}
	} else {
		src.Catalog = nil
	}

	fmt.Println("\n--- Summary ---")
	summary, _ := json.MarshalIndent(src, "", "  ")
	fmt.Println(string(summary))
	if !p.askYesNo("Write this and generate your signed manifest?", true) {
		fmt.Println("Cancelled, nothing was written.")
		return
	}

	if err := writeSourceFile(sourcePath, &src); err != nil {
		fmt.Println("Error writing", sourcePath, ":", err)
		return
	}

	result, err := generate.Run(generate.Options{
		SourcePath:      sourcePath,
		ManifestOutPath: manifestOutPath,
		HistoryOutPath:  historyOutPath,
	})
	if err != nil {
		fmt.Println("Error generating manifest:", err)
		return
	}
	fmt.Printf("\nWrote %s and %s.\n", manifestOutPath, historyOutPath)
	if len(result.NewEntries) > 0 {
		fmt.Printf("Recorded %d new history entr(ies).\n", len(result.NewEntries))
	}
}

func menuAddAlbum(p *prompter) manifest.Album {
	var a manifest.Album
	a.AlbumID = p.askRequired("  Album ID (stable, never changes, e.g. \"agency-2024\")")
	a.AlbumName = p.askRequired("  Album title")
	a.ReleaseDate = p.askRequired("  Release date (YYYY-MM-DD)")
	a.Images.Front = p.askRequired("  Front cover image URL")
	a.Images.Back = p.ask("  Back cover image URL (optional, leave blank if there isn't one)", "")
	a.Images.Insert = []string{}
	for p.askYesNo("  Add an insert/booklet image?", false) {
		a.Images.Insert = append(a.Images.Insert, p.askRequired("    Insert image URL"))
	}
	for p.askYesNo(fmt.Sprintf("  Add a track? (%d so far)", len(a.Tracks)), len(a.Tracks) == 0) {
		var t manifest.Track
		t.TrackID = p.askRequired("    Track ID (stable, e.g. \"agency-2024-a1\")")
		t.Number = p.ask("    Display number (e.g. \"A1\")", fmt.Sprintf("%d", len(a.Tracks)+1))
		t.Name = p.askRequired("    Track title")
		t.Duration = p.ask("    Duration (display only, e.g. 3'30\")", "")
		t.File = p.askRequired("    Audio file URL")
		a.Tracks = append(a.Tracks, t)
	}
	return a
}

func menuValidate(p *prompter) {
	target := p.askRequired("\nLocal file path or URL to validate")
	deep := p.askYesNo("Run the deep check (fetches cross-referenced manifests)?", false)
	report, err := runValidate(target, deep)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	fmt.Println()
	fmt.Print(report.Summary())
}

func runValidate(target string, deep bool) (*validate.Report, error) {
	message := fmt.Sprintf("Validating %s...", target)
	if deep {
		message = fmt.Sprintf("Validating %s (deep checks, this fetches other manifests)...", target)
	}
	sp := spinner.New(os.Stderr, message)
	defer sp.Stop()

	opts := validate.Options{Deep: deep}
	if isURL(target) {
		return validate.ValidateURL(target, opts)
	}
	return validate.ValidatePath(target, opts)
}

func menuManageKey(p *prompter) {
	src := loadExistingSource()
	url := ""
	if src != nil {
		url = src.Identity.URL
	}
	url = p.ask("\nIdentity URL", url)
	if url == "" {
		fmt.Println("No identity URL to look up a key for.")
		return
	}
	keyPath, err := signing.KeyPath(url)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	if !signing.KeyExists(keyPath) {
		fmt.Println("No local key found for", url)
		return
	}
	priv, err := signing.LoadPrivateKey(keyPath)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}
	pub := priv.Public().(ed25519.PublicKey)
	fmt.Println("Public key:", signing.EncodePublicKey(pub))

	if p.askYesNo("Rotate this key now?", false) {
		fmt.Println("This only works while the old key is still present locally.")
		fmt.Println("If it's already lost, there is no rotation path, only starting a new identity and reaching out to affected parties directly.")
		if !p.askYesNo("Continue with rotation?", false) {
			return
		}
		result, err := generate.Rotate(generate.RotateOptions{
			ManifestPath: manifestOutPath,
			HistoryPath:  historyOutPath,
		})
		if err != nil {
			fmt.Println("Error:", err)
			return
		}
		fmt.Println("Rotated key.")
		fmt.Println("Old public key:", result.OldPublicKey)
		fmt.Println("New public key:", result.NewPublicKey)
	}
}

func isURL(s string) bool {
	return len(s) > 7 && (s[:7] == "http://" || (len(s) > 8 && s[:8] == "https://"))
}
