package slug

import "testing"

func TestSlugify(t *testing.T) {
	cases := map[string]string{
		"Demo":                 "demo",
		"PARACIDIC":            "paracidic",
		"Anything, Anything!":  "anything-anything",
		"A Time To Die":        "a-time-to-die",
		"  leading and trail ": "leading-and-trail",
		"already-a-slug":       "already-a-slug",
		"multiple   spaces":    "multiple-spaces",
		"":                     "",
		"100%":                 "100",
	}
	for in, want := range cases {
		if got := Slugify(in); got != want {
			t.Errorf("Slugify(%q) = %q, want %q", in, got, want)
		}
	}
}
