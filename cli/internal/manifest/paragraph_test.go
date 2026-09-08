package manifest

import (
	"encoding/json"
	"testing"
)

func TestRawParagraph_MarshalPlainText(t *testing.T) {
	p := RawParagraph{PlainText: "hello"}
	got, err := json.Marshal(p)
	if err != nil {
		t.Fatalf("Marshal returned error: %v", err)
	}
	if string(got) != `"hello"` {
		t.Errorf("Marshal() = %s, want %q", got, `"hello"`)
	}
}

func TestRawParagraph_MarshalSegments(t *testing.T) {
	p := RawParagraph{Segments: []CreditSegment{
		{Text: "Mastered at "},
		{Text: "Cool Studio", URL: "https://coolstudio.example"},
	}}
	got, err := json.Marshal(p)
	if err != nil {
		t.Fatalf("Marshal returned error: %v", err)
	}
	want := `[{"text":"Mastered at "},{"text":"Cool Studio","url":"https://coolstudio.example"}]`
	if string(got) != want {
		t.Errorf("Marshal() = %s, want %s", got, want)
	}
}

func TestRawParagraph_UnmarshalPlainText(t *testing.T) {
	var p RawParagraph
	if err := json.Unmarshal([]byte(`"a plain paragraph"`), &p); err != nil {
		t.Fatalf("Unmarshal returned error: %v", err)
	}
	if p.PlainText != "a plain paragraph" || p.Segments != nil {
		t.Errorf("Unmarshal() = %+v, want PlainText set and Segments nil", p)
	}
}

func TestRawParagraph_UnmarshalSegments(t *testing.T) {
	var p RawParagraph
	raw := `[{"text":"a"},{"text":"b","url":"https://example.com"}]`
	if err := json.Unmarshal([]byte(raw), &p); err != nil {
		t.Fatalf("Unmarshal returned error: %v", err)
	}
	if len(p.Segments) != 2 || p.Segments[1].URL != "https://example.com" {
		t.Errorf("Unmarshal() = %+v, want 2 segments with second URL set", p)
	}
	if p.PlainText != "" {
		t.Errorf("Unmarshal() left stale PlainText %q", p.PlainText)
	}
}

func TestRawParagraph_UnmarshalEmptyErrors(t *testing.T) {
	var p RawParagraph
	if err := json.Unmarshal([]byte(``), &p); err == nil {
		t.Error("Unmarshal(\"\") : want error, got nil")
	}
}

func TestRawParagraph_UnmarshalInvalidJSON(t *testing.T) {
	var p RawParagraph
	if err := json.Unmarshal([]byte(`{not json`), &p); err == nil {
		t.Error("Unmarshal(malformed object): want error, got nil")
	}
}

func TestCredits_RoundTripInAlbumJSON(t *testing.T) {
	c := Credits{Paragraphs: []RawParagraph{
		{PlainText: "Written and performed by Ligatures."},
		{Segments: []CreditSegment{{Text: "Mastered at "}, {Text: "Cool Studio", URL: "https://coolstudio.example"}}},
	}}
	raw, err := json.Marshal(c)
	if err != nil {
		t.Fatalf("Marshal returned error: %v", err)
	}
	var back Credits
	if err := json.Unmarshal(raw, &back); err != nil {
		t.Fatalf("Unmarshal returned error: %v", err)
	}
	if len(back.Paragraphs) != 2 {
		t.Fatalf("got %d paragraphs, want 2", len(back.Paragraphs))
	}
	if back.Paragraphs[0].PlainText != c.Paragraphs[0].PlainText {
		t.Errorf("paragraph 0 = %+v, want %+v", back.Paragraphs[0], c.Paragraphs[0])
	}
	if len(back.Paragraphs[1].Segments) != 2 {
		t.Errorf("paragraph 1 segments = %+v, want 2 entries", back.Paragraphs[1].Segments)
	}
}
