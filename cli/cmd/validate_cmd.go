package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

type validateArgs struct {
	target string
	deep   bool
	json   bool
}

// parseValidateArgs accepts --deep and --json in any position relative to
// the <path-or-url> argument, since 0004's own usage examples put them
// after it (`validate <path-or-url> [--deep] [--json]`), which the
// standard flag package's strict flags-before-positionals rule wouldn't
// otherwise allow.
func parseValidateArgs(args []string) (validateArgs, error) {
	var parsed validateArgs
	var targets []string
	for _, a := range args {
		switch a {
		case "--deep":
			parsed.deep = true
		case "--json":
			parsed.json = true
		default:
			if strings.HasPrefix(a, "-") {
				return validateArgs{}, fmt.Errorf("unknown flag %q", a)
			}
			targets = append(targets, a)
		}
	}
	if len(targets) != 1 {
		return validateArgs{}, fmt.Errorf("expected exactly one <path-or-url> argument, got %d", len(targets))
	}
	parsed.target = targets[0]
	return parsed, nil
}

func cmdValidate(args []string) int {
	parsed, err := parseValidateArgs(args)
	if err != nil {
		fmt.Fprintln(os.Stderr, "validate:", err)
		return 1
	}
	report, err := runValidate(parsed.target, parsed.deep)
	if err != nil {
		fmt.Fprintln(os.Stderr, "validate:", err)
		return 1
	}
	if parsed.json {
		enc, err := json.Marshal(report)
		if err != nil {
			fmt.Fprintln(os.Stderr, "validate:", err)
			return 1
		}
		fmt.Println(string(enc))
	} else {
		fmt.Print(report.Summary())
	}
	if !report.Valid {
		return 1
	}
	return 0
}
