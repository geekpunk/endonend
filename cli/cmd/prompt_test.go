package main

import (
	"bufio"
	"strings"
	"testing"
)

func newTestPrompter(input string) *prompter {
	return newPrompter(bufio.NewReader(strings.NewReader(input)))
}

func TestAsk_ReturnsAnswerWhenGiven(t *testing.T) {
	p := newTestPrompter("hello\n")
	if got := p.ask("Question", "default"); got != "hello" {
		t.Errorf("ask() = %q, want %q", got, "hello")
	}
}

func TestAsk_ReturnsDefaultWhenEmpty(t *testing.T) {
	p := newTestPrompter("\n")
	if got := p.ask("Question", "default"); got != "default" {
		t.Errorf("ask() = %q, want %q", got, "default")
	}
}

func TestAsk_TrimsWhitespace(t *testing.T) {
	p := newTestPrompter("  spaced  \n")
	if got := p.ask("Question", ""); got != "spaced" {
		t.Errorf("ask() = %q, want %q", got, "spaced")
	}
}

func TestAskRequired_KeepsAskingUntilNonEmpty(t *testing.T) {
	p := newTestPrompter("\n\nfinally\n")
	if got := p.askRequired("Question"); got != "finally" {
		t.Errorf("askRequired() = %q, want %q", got, "finally")
	}
}

func TestAskYesNo_DefaultsAndExplicitAnswers(t *testing.T) {
	cases := []struct {
		input      string
		defYes     bool
		want       bool
		reasonWhat string
	}{
		{"\n", true, true, "empty answer keeps default true"},
		{"\n", false, false, "empty answer keeps default false"},
		{"y\n", false, true, "explicit y overrides default false"},
		{"yes\n", false, true, "explicit yes overrides default false"},
		{"n\n", true, false, "explicit n overrides default true"},
		{"Y\n", false, true, "uppercase Y is accepted"},
	}
	for _, c := range cases {
		p := newTestPrompter(c.input)
		if got := p.askYesNo("Question", c.defYes); got != c.want {
			t.Errorf("askYesNo(%q, %v) = %v, want %v (%s)", c.input, c.defYes, got, c.want, c.reasonWhat)
		}
	}
}

func TestAskInt_ParsesOrKeepsAsking(t *testing.T) {
	p := newTestPrompter("not-a-number\n42\n")
	if got := p.askInt("Question", 1); got != 42 {
		t.Errorf("askInt() = %d, want 42", got)
	}
}

func TestAskInt_UsesDefaultOnEmpty(t *testing.T) {
	p := newTestPrompter("\n")
	if got := p.askInt("Question", 21600); got != 21600 {
		t.Errorf("askInt() = %d, want 21600", got)
	}
}

func TestAskFloat_ParsesOrKeepsAsking(t *testing.T) {
	p := newTestPrompter("nope\n85.5\n")
	if got := p.askFloat("Question", 0); got != 85.5 {
		t.Errorf("askFloat() = %v, want 85.5", got)
	}
}
