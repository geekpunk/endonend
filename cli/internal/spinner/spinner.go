// Package spinner shows a terminal spinner for a long-running operation
// (a network fetch, a multi-file download) without corrupting output
// that isn't an interactive terminal: piped/redirected output, CI logs,
// and machine-readable output like `validate --json` all see nothing.
package spinner

import (
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"
)

var frameChars = [...]rune{'⠋', '⠙', '⠹', '⠸', '⠼', '⠴', '⠦', '⠧', '⠇', '⠏'}

const defaultInterval = 100 * time.Millisecond

// Spinner animates a rotating frame plus a message on one terminal line,
// updating and clearing in place. The zero value is not usable; construct
// one with New. All methods are safe to call on a nil *Spinner, so a
// caller can pass one around without a separate "is there a spinner"
// check at every call site.
type Spinner struct {
	mu       sync.Mutex
	message  string
	w        io.Writer
	enabled  bool
	stopOnce sync.Once
	// stop and done are set once at construction and never reassigned;
	// run reads them via its own parameters rather than through the
	// struct, and Stop closes/waits on them directly, so neither needs
	// mu once the goroutine has started (a previous version reassigned
	// s.stop to nil in Stop, which raced with run's repeated `case
	// <-s.stop` re-evaluating the field and could leave run selecting
	// on a nil channel forever; see the race-detector failure this
	// replaced).
	stop chan struct{}
	done chan struct{}

	interval time.Duration
}

// New starts a spinner writing to w with the given initial message. If w
// isn't a terminal (redirected to a file, a pipe, or /dev/null, as in
// tests and CI), New returns a disabled Spinner whose methods are no-ops,
// so no control characters ever reach non-interactive output.
func New(w *os.File, message string) *Spinner {
	return newSpinner(w, message, isTerminal(w), defaultInterval)
}

func newSpinner(w io.Writer, message string, enabled bool, interval time.Duration) *Spinner {
	s := &Spinner{message: message, w: w, enabled: enabled, interval: interval}
	if !enabled {
		return s
	}
	s.stop = make(chan struct{})
	s.done = make(chan struct{})
	go s.run(s.stop, s.done)
	return s
}

func (s *Spinner) run(stop, done chan struct{}) {
	defer close(done)
	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()
	i := 0
	lastLen := 0
	for {
		select {
		case <-stop:
			// Best-effort: a failed write here means the terminal is
			// gone, nothing left to report it to or recover for.
			_, _ = fmt.Fprint(s.w, clearLine(lastLen))
			return
		case <-ticker.C:
			s.mu.Lock()
			msg := s.message
			s.mu.Unlock()
			line, n := renderLine(i, msg, lastLen)
			_, _ = fmt.Fprint(s.w, line)
			lastLen = n
			i++
		}
	}
}

// Update changes the message shown next to the spinner.
func (s *Spinner) Update(message string) {
	if s == nil || !s.enabled {
		return
	}
	s.mu.Lock()
	s.message = message
	s.mu.Unlock()
}

// Stop stops the animation and clears the line. Safe to call more than
// once, and safe to call concurrently; only the first call closes the
// stop channel, and every call waits for the animation goroutine to exit.
func (s *Spinner) Stop() {
	if s == nil || !s.enabled {
		return
	}
	s.stopOnce.Do(func() {
		close(s.stop)
	})
	<-s.done
}

// renderLine builds the carriage-return-prefixed line for frame i showing
// message, padded with spaces to fully overwrite a previous line of
// prevLen visible bytes, and returns the new line's own visible length.
func renderLine(i int, message string, prevLen int) (line string, length int) {
	base := fmt.Sprintf("%c %s", frameChars[i%len(frameChars)], message)
	length = len(base)
	if pad := prevLen - length; pad > 0 {
		base += strings.Repeat(" ", pad)
	}
	return "\r" + base, length
}

// clearLine returns the sequence that erases a previously written line of
// prevLen visible bytes and returns the cursor to the start of the line.
func clearLine(prevLen int) string {
	if prevLen == 0 {
		return ""
	}
	return "\r" + strings.Repeat(" ", prevLen) + "\r"
}

func isTerminal(f *os.File) bool {
	if f == nil {
		return false
	}
	fi, err := f.Stat()
	if err != nil {
		return false
	}
	return fi.Mode()&os.ModeCharDevice != 0
}
