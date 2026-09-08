package main

import "testing"

func TestRequireEnv(t *testing.T) {
	t.Setenv("ENDONEND_TEST_VAR", "value")
	got, err := requireEnv("ENDONEND_TEST_VAR")
	if err != nil {
		t.Fatalf("requireEnv: %v", err)
	}
	if got != "value" {
		t.Errorf("requireEnv = %q, want %q", got, "value")
	}
}

func TestRequireEnv_MissingReturnsError(t *testing.T) {
	t.Setenv("ENDONEND_TEST_VAR_UNSET", "")
	if _, err := requireEnv("ENDONEND_TEST_VAR_UNSET"); err == nil {
		t.Error("requireEnv for an unset variable: want error, got nil")
	}
}

func TestEnvOr(t *testing.T) {
	t.Setenv("ENDONEND_TEST_VAR", "value")
	if got := envOr("ENDONEND_TEST_VAR", "fallback"); got != "value" {
		t.Errorf("envOr with the var set = %q, want %q", got, "value")
	}
	if got := envOr("ENDONEND_TEST_VAR_UNSET", "fallback"); got != "fallback" {
		t.Errorf("envOr with the var unset = %q, want %q", got, "fallback")
	}
}
