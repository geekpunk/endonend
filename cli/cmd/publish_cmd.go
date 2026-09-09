package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"endonend/cli/internal/ghpublish"
	"endonend/protocol/manifest"
)

type publishGithubArgs struct {
	manifestPath string
	historyPath  string
	assetsDir    string
	outDir       string
	branch       string
	private      bool
}

func parsePublishGithubArgs(args []string) (publishGithubArgs, error) {
	fs := flag.NewFlagSet("publish github", flag.ContinueOnError)
	a := publishGithubArgs{}
	fs.StringVar(&a.manifestPath, "manifest", manifestOutPath, "Signed manifest to publish")
	fs.StringVar(&a.historyPath, "history", historyOutPath, "Paired history log to publish")
	fs.StringVar(&a.assetsDir, "assets-dir", "bandcamp-import", "Local directory holding album assets, per <assets-dir>/<albumId>/<filename>")
	fs.StringVar(&a.outDir, "out-dir", "publish", "Where the assembled, publish-ready site is written (kept between runs so re-publishing is a fast-forward push)")
	fs.StringVar(&a.branch, "branch", "main", "Branch to push and serve GitHub Pages from")
	fs.BoolVar(&a.private, "private", false, "Create the repo as private (GitHub Pages on a private repo needs GitHub Pro or an org plan)")
	if err := fs.Parse(args); err != nil {
		return publishGithubArgs{}, err
	}
	return a, nil
}

func cmdPublish(args []string) int {
	if len(args) == 0 || args[0] != "github" {
		fmt.Fprintln(os.Stderr, "publish: expected a target, e.g. \"publish github\"")
		return 1
	}
	return cmdPublishGithub(args[1:])
}

func cmdPublishGithub(args []string) int {
	parsed, err := parsePublishGithubArgs(args)
	if err != nil {
		return 1
	}
	if err := publishGithub(parsed, ghpublish.DefaultRunner); err != nil {
		fmt.Fprintln(os.Stderr, "publish github:", err)
		return 1
	}
	return 0
}

// publishGithub assembles parsed.manifestPath's declared site tree and
// pushes it to the GitHub Pages repo its identity.url implies, creating
// the repo and enabling Pages if needed. Shared by the "publish github"
// subcommand and the interactive menu's equivalent option. runner is a
// parameter (rather than always ghpublish.DefaultRunner) so tests can
// verify this orchestration without actually invoking git or gh.
func publishGithub(parsed publishGithubArgs, runner ghpublish.Runner) error {
	raw, err := os.ReadFile(parsed.manifestPath)
	if err != nil {
		return fmt.Errorf("read %s (run generate first): %w", parsed.manifestPath, err)
	}
	var m manifest.Manifest
	if err := json.Unmarshal(raw, &m); err != nil {
		return fmt.Errorf("parse %s: %w", parsed.manifestPath, err)
	}

	target, err := ghpublish.ParsePagesURL(m.Identity.URL)
	if err != nil {
		return err
	}

	fmt.Printf("Publishing %s to %s ...\n", m.Identity.URL, target.URL())

	fmt.Println("Assembling site files...")
	if err := ghpublish.Stage(&m, ghpublish.StageOptions{
		ManifestPath: parsed.manifestPath,
		HistoryPath:  parsed.historyPath,
		AssetsDir:    parsed.assetsDir,
		OutDir:       parsed.outDir,
	}); err != nil {
		return fmt.Errorf("assemble site: %w", err)
	}

	created, err := ghpublish.EnsureRepo(runner, target, parsed.private)
	if err != nil {
		return err
	}
	if created {
		fmt.Printf("Created GitHub repo %s.\n", target.FullName())
	}

	fmt.Println("Pushing...")
	if err := ghpublish.PushDir(runner, parsed.outDir, target, parsed.branch); err != nil {
		return fmt.Errorf("push to GitHub: %w", err)
	}

	if err := ghpublish.EnablePages(runner, target, parsed.branch); err != nil {
		return fmt.Errorf("enable GitHub Pages: %w", err)
	}

	fmt.Printf("Done. GitHub Pages can take a minute or two to go live at %s\n", m.Identity.URL)
	return nil
}

func menuPublishGithub(p *prompter) {
	parsed := publishGithubArgs{
		manifestPath: manifestOutPath,
		historyPath:  historyOutPath,
		assetsDir:    "bandcamp-import",
		outDir:       "publish",
		branch:       "main",
	}
	parsed.assetsDir = p.ask("\nLocal directory with downloaded album assets", parsed.assetsDir)
	parsed.outDir = p.ask("Directory to assemble the publish-ready site into", parsed.outDir)
	parsed.private = p.askYesNo("Create the GitHub repo as private? (needs GitHub Pro/an org plan for Pages)", false)

	fmt.Println()
	if err := publishGithub(parsed, ghpublish.DefaultRunner); err != nil {
		fmt.Println("Error:", err)
	}
}
