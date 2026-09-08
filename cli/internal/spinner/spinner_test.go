package spinner

import (
	"bytes"
	"os"
	"strings"
	"testing"
	"time"
)

func TestFrameChars_Cycle(t *testing.T) {
	if frameChars[10%len(frameChars)] != frameChars[0] {
		t.Error("frame index does not wrap around the frame set")
	}
}

func TestRenderLine_NoPreviousLine(t *testing.T) {
	line, length := renderLine(0, "hello", 0)
	want := "\r" + string(frameChars[0]) + " hello"
	if line != want {
		t.Errorf("renderLine() line = %q, want %q", line, want)
	}
	if length != len(string(frameChars[0])+" hello") {
		t.Errorf("renderLine() length = %d, want %d", length, len(string(frameChars[0])+" hello"))
	}
}

func TestRenderLine_PadsToClearShorterMessage(t *testing.T) {
	line, length := renderLine(1, "hi", 10)
	base := string(frameChars[1]) + " hi"
	wantPad := 10 - len(base)
	if !strings.HasSuffix(line, strings.Repeat(" ", wantPad)) {
		t.Errorf("renderLine() = %q, want it padded with %d trailing spaces", line, wantPad)
	}
	if length != len(base) {
		t.Errorf("renderLine() length = %d, want the unpadded length %d", length, len(base))
	}
}

func TestRenderLine_NoPaddingWhenLonger(t *testing.T) {
	line, _ := renderLine(0, "a longer message", 2)
	if strings.HasSuffix(line, " ") {
		t.Errorf("renderLine() = %q, should not pad when the new message is already longer", line)
	}
}

func TestClearLine_Empty(t *testing.T) {
	if got := clearLine(0); got != "" {
		t.Errorf("clearLine(0) = %q, want empty", got)
	}
}

func TestClearLine_NonEmpty(t *testing.T) {
	got := clearLine(5)
	want := "\r     \r"
	if got != want {
		t.Errorf("clearLine(5) = %q, want %q", got, want)
	}
}

func TestIsTerminal_RegularFileIsFalse(t *testing.T) {
	f, err := os.CreateTemp(t.TempDir(), "not-a-tty")
	if err != nil {
		t.Fatalf("CreateTemp: %v", err)
	}
	defer func() { _ = f.Close() }()
	if isTerminal(f) {
		t.Error("isTerminal(regular file) = true, want false")
	}
}

func TestIsTerminal_NilIsFalse(t *testing.T) {
	if isTerminal(nil) {
		t.Error("isTerminal(nil) = true, want false")
	}
}

func TestNew_DisabledForNonTerminalIsANoOp(t *testing.T) {
	f, err := os.CreateTemp(t.TempDir(), "not-a-tty")
	if err != nil {
		t.Fatalf("CreateTemp: %v", err)
	}
	defer func() { _ = f.Close() }()

	s := New(f, "working...")
	s.Update("still working...")
	s.Stop()
	s.Stop() // must tolerate a second Stop without panicking

	if fi, err := f.Stat(); err != nil || fi.Size() != 0 {
		t.Errorf("a disabled spinner wrote to non-terminal output (size=%d, err=%v)", fi.Size(), err)
	}
}

func TestNilSpinner_MethodsAreNoOps(t *testing.T) {
	var s *Spinner
	s.Update("x") // must not panic
	s.Stop()      // must not panic
}

func TestSpinner_AnimatesAndClearsOnStop(t *testing.T) {
	var buf bytes.Buffer
	s := newSpinner(&buf, "loading", true, time.Millisecond)

	// Give the animation goroutine time to render several frames, then
	// Stop() before inspecting buf: Stop() blocks until the goroutine's
	// last write and exit are complete, so reading buf afterward has no
	// concurrent writer and needs no separate synchronization.
	time.Sleep(20 * time.Millisecond)
	s.Update("almost done")
	time.Sleep(5 * time.Millisecond)
	s.Stop()

	out := buf.String()
	if out == "" {
		t.Fatal("spinner wrote nothing while enabled")
	}
	if !strings.Contains(out, "loading") && !strings.Contains(out, "almost done") {
		t.Errorf("spinner output %q contains neither message", out)
	}
	// Stop() clears the line: the output should end with a run of spaces
	// bracketed by carriage returns.
	if !strings.HasSuffix(out, "\r") {
		t.Errorf("spinner output %q should end with the clearing carriage return", out)
	}
}

func TestSpinner_StopIsIdempotent(t *testing.T) {
	var buf bytes.Buffer
	s := newSpinner(&buf, "loading", true, time.Millisecond)
	s.Stop()
	s.Stop()
}
