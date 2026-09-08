package canonical

import "testing"

func TestMarshal_SortsKeysAndDropsWhitespace(t *testing.T) {
	type payload struct {
		Zebra string `json:"zebra"`
		Alpha string `json:"alpha"`
	}
	got, err := Marshal(payload{Zebra: "z", Alpha: "a"})
	if err != nil {
		t.Fatalf("Marshal returned error: %v", err)
	}
	want := `{"alpha":"a","zebra":"z"}`
	if string(got) != want {
		t.Errorf("Marshal() = %q, want %q", got, want)
	}
}

func TestMarshal_NestedMapsAreSorted(t *testing.T) {
	got, err := Marshal(map[string]any{
		"b": 1,
		"a": map[string]any{"y": 2, "x": 1},
	})
	if err != nil {
		t.Fatalf("Marshal returned error: %v", err)
	}
	want := `{"a":{"x":1,"y":2},"b":1}`
	if string(got) != want {
		t.Errorf("Marshal() = %q, want %q", got, want)
	}
}

func TestMarshal_InvalidValueErrors(t *testing.T) {
	// channels cannot be marshaled to JSON.
	if _, err := Marshal(make(chan int)); err == nil {
		t.Error("Marshal() with an unencodable value: want error, got nil")
	}
}

func TestToMap_RoundTripsFields(t *testing.T) {
	type payload struct {
		Name string `json:"name"`
		N    int    `json:"n"`
	}
	m, err := ToMap(payload{Name: "x", N: 3})
	if err != nil {
		t.Fatalf("ToMap returned error: %v", err)
	}
	if m["name"] != "x" {
		t.Errorf("ToMap()[\"name\"] = %v, want %q", m["name"], "x")
	}
	if m["n"].(float64) != 3 {
		t.Errorf("ToMap()[\"n\"] = %v, want 3", m["n"])
	}
}

func TestToMap_Nil(t *testing.T) {
	m, err := ToMap(nil)
	if err != nil {
		t.Fatalf("ToMap(nil) returned error: %v", err)
	}
	if m != nil {
		t.Errorf("ToMap(nil) = %v, want nil map", m)
	}
}
