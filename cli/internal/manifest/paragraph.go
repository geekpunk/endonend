package manifest

import (
	"bytes"
	"encoding/json"
	"errors"
)

// MarshalJSON emits a plain string when the paragraph has no segments, or a
// segment array otherwise, matching 0003's "either a plain string, or an
// array of segments" shape.
func (p RawParagraph) MarshalJSON() ([]byte, error) {
	if p.Segments != nil {
		return json.Marshal(p.Segments)
	}
	return json.Marshal(p.PlainText)
}

func (p *RawParagraph) UnmarshalJSON(data []byte) error {
	trimmed := bytes.TrimSpace(data)
	if len(trimmed) == 0 {
		return errors.New("empty credits paragraph")
	}
	if trimmed[0] == '"' {
		var s string
		if err := json.Unmarshal(data, &s); err != nil {
			return err
		}
		p.PlainText = s
		p.Segments = nil
		return nil
	}
	var segs []CreditSegment
	if err := json.Unmarshal(data, &segs); err != nil {
		return err
	}
	p.Segments = segs
	p.PlainText = ""
	return nil
}
