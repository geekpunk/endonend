package ghpublish

import (
	"fmt"
	"strings"
)

// call records one invocation a fakeRunner received.
type call struct {
	dir  string
	name string
	args []string
}

func (c call) String() string {
	return c.name + " " + strings.Join(c.args, " ")
}

// fakeRunner is a Runner test double: it records every call and returns a
// canned error keyed by the exact "name arg1 arg2 ..." command string, so
// tests can verify exactly which commands publish.go issues without
// actually invoking git or gh.
type fakeRunner struct {
	calls  []call
	errors map[string]error
}

func newFakeRunner() *fakeRunner {
	return &fakeRunner{errors: map[string]error{}}
}

func (f *fakeRunner) Run(dir, name string, args ...string) (string, error) {
	c := call{dir: dir, name: name, args: args}
	f.calls = append(f.calls, c)
	if err, ok := f.errors[c.String()]; ok {
		return "", err
	}
	return "", nil
}

// failOn makes the fake return err the next time it sees a call matching
// exactly name followed by args.
func (f *fakeRunner) failOn(err error, name string, args ...string) {
	f.errors[call{name: name, args: args}.String()] = err
}

// calledWith reports whether any recorded call matches exactly name
// followed by args.
func (f *fakeRunner) calledWith(name string, args ...string) bool {
	for _, c := range f.calls {
		if c.String() == (call{name: name, args: args}).String() {
			return true
		}
	}
	return false
}

var errFake = fmt.Errorf("fake command failure")
