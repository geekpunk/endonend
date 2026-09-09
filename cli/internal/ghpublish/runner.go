package ghpublish

import (
	"bytes"
	"fmt"
	"os/exec"
)

// Runner executes an external command in dir, capturing combined output.
// Production code uses DefaultRunner (real git/gh); tests substitute a
// fake to verify the right commands are issued without actually invoking
// either.
type Runner interface {
	Run(dir, name string, args ...string) (output string, err error)
}

type execRunner struct{}

func (execRunner) Run(dir, name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf
	err := cmd.Run()
	if err != nil {
		return buf.String(), fmt.Errorf("%s %v: %w: %s", name, args, err, buf.String())
	}
	return buf.String(), nil
}

// DefaultRunner is the Runner production code uses.
var DefaultRunner Runner = execRunner{}
