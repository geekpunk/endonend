// Command endtoend-artist-cli generates, signs, and validates the Union
// manifest described in KB/0003-manifest.md, per the CLI design in
// KB/0004-endtoend-artist-cli.md.
package main

import (
	"fmt"
	"os"
)

func main() {
	os.Exit(run(os.Args[1:]))
}

func run(args []string) int {
	if len(args) == 0 {
		runMenu()
		return 0
	}
	switch args[0] {
	case "generate":
		return cmdGenerate(args[1:])
	case "validate":
		return cmdValidate(args[1:])
	case "keygen":
		return cmdKeygen(args[1:])
	case "import":
		return cmdImport(args[1:])
	case "help", "--help", "-h":
		printHelp()
		return 0
	default:
		fmt.Fprintf(os.Stderr, "unknown command %q\n\n", args[0])
		printHelp()
		return 1
	}
}
