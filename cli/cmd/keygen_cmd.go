package main

import (
	"flag"
	"fmt"
	"os"

	"endonend/cli/internal/generate"
	"endonend/protocol/signing"
)

type keygenArgs struct {
	url         string
	rotate      bool
	manifestOut string
	historyOut  string
}

func parseKeygenArgs(args []string) (keygenArgs, error) {
	fs := flag.NewFlagSet("keygen", flag.ContinueOnError)
	url := fs.String("url", "", "Identity URL to generate a first-time key for")
	rotate := fs.Bool("rotate", false, "Rotate the key already in use")
	manifestOut := fs.String("manifest-out", manifestOutPath, "Manifest to rotate, when --rotate is set")
	historyOut := fs.String("history-out", historyOutPath, "History log to update, when --rotate is set")
	if err := fs.Parse(args); err != nil {
		return keygenArgs{}, err
	}
	if !*rotate && *url == "" {
		return keygenArgs{}, fmt.Errorf("--url is required unless --rotate is set")
	}
	return keygenArgs{url: *url, rotate: *rotate, manifestOut: *manifestOut, historyOut: *historyOut}, nil
}

func cmdKeygen(args []string) int {
	parsed, err := parseKeygenArgs(args)
	if err != nil {
		fmt.Fprintln(os.Stderr, "keygen:", err)
		return 1
	}
	if parsed.rotate {
		result, err := generate.Rotate(generate.RotateOptions{
			ManifestPath: parsed.manifestOut,
			HistoryPath:  parsed.historyOut,
		})
		if err != nil {
			fmt.Fprintln(os.Stderr, "keygen:", err)
			return 1
		}
		fmt.Println("Rotated key.")
		fmt.Println("Old public key:", result.OldPublicKey)
		fmt.Println("New public key:", result.NewPublicKey)
		return 0
	}

	keyPath, err := signing.KeyPath(parsed.url)
	if err != nil {
		fmt.Fprintln(os.Stderr, "keygen:", err)
		return 1
	}
	if signing.KeyExists(keyPath) {
		fmt.Fprintf(os.Stderr, "keygen: a key already exists at %s; use --rotate to replace it\n", keyPath)
		return 1
	}
	pub, priv, err := signing.GenerateKeypair()
	if err != nil {
		fmt.Fprintln(os.Stderr, "keygen:", err)
		return 1
	}
	if err := signing.SavePrivateKey(keyPath, priv); err != nil {
		fmt.Fprintln(os.Stderr, "keygen:", err)
		return 1
	}
	fmt.Println("Public key:", signing.EncodePublicKey(pub))
	fmt.Println("Private key written to", keyPath)
	return 0
}
