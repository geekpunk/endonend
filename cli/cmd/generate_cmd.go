package main

import (
	"flag"
	"fmt"
	"os"

	"endonend/cli/internal/generate"
)

func parseGenerateArgs(args []string) (generate.Options, error) {
	fs := flag.NewFlagSet("generate", flag.ContinueOnError)
	source := fs.String("source", sourcePath, "Path to endonend.source.json")
	manifestOut := fs.String("manifest-out", manifestOutPath, "Where to write the signed manifest")
	historyOut := fs.String("history-out", historyOutPath, "Where to write the history log")
	if err := fs.Parse(args); err != nil {
		return generate.Options{}, err
	}
	return generate.Options{
		SourcePath:      *source,
		ManifestOutPath: *manifestOut,
		HistoryOutPath:  *historyOut,
	}, nil
}

func cmdGenerate(args []string) int {
	opts, err := parseGenerateArgs(args)
	if err != nil {
		return 1
	}
	result, err := generate.Run(opts)
	if err != nil {
		fmt.Fprintln(os.Stderr, "generate:", err)
		return 1
	}
	fmt.Printf("Wrote %s and %s.\n", opts.ManifestOutPath, opts.HistoryOutPath)
	if result.KeyGenerated {
		fmt.Printf("Generated a new signing key at %s. Keep it out of version control.\n", result.PrivateKeyPath)
	}
	if len(result.NewEntries) > 0 {
		fmt.Printf("Recorded %d new history entr(ies).\n", len(result.NewEntries))
	}
	return 0
}
