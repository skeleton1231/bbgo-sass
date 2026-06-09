package main

import (
	"encoding/json"
	"testing"
)

// --- computeInstanceID (delegates to instanceid.Compute) ---

func TestComputeInstanceID_Grid2WithParams(t *testing.T) {
	config := rawJSON(`{"gridNumber":10,"upperPrice":"70000","lowerPrice":"50000"}`)
	id := computeInstanceID("grid2", "BTCUSDT", config)
	want := "grid2-BTCUSDT-size-10-70000-50000"
	if id != want {
		t.Errorf("got %q, want %q", id, want)
	}
}

func TestComputeInstanceID_Grid2NoParams(t *testing.T) {
	id := computeInstanceID("grid2", "BTCUSDT", nil)
	if id != "grid2-BTCUSDT-size-0--" {
		t.Errorf("got %q, want %q", id, "grid2-BTCUSDT-size-0--")
	}
}

func TestComputeInstanceID_Emacross(t *testing.T) {
	config := rawJSON(`{"interval":"1h","fastWindow":10,"slowWindow":30}`)
	id := computeInstanceID("emacross", "ETHUSDT", config)
	want := "emacross:ETHUSDT:1h:10-30"
	if id != want {
		t.Errorf("got %q, want %q", id, want)
	}
}

func TestComputeInstanceID_Supertrend(t *testing.T) {
	id := computeInstanceID("supertrend", "BTCUSDT", rawJSON(`{}`))
	if id != "supertrend:BTCUSDT" {
		t.Errorf("got %q, want %q", id, "supertrend:BTCUSDT")
	}
}

func TestComputeInstanceID_Bollmaker(t *testing.T) {
	id := computeInstanceID("bollmaker", "ETHUSDT", rawJSON(`{}`))
	if id != "bollmaker:ETHUSDT" {
		t.Errorf("got %q, want %q", id, "bollmaker:ETHUSDT")
	}
}

func TestComputeInstanceID_DCA2(t *testing.T) {
	id := computeInstanceID("dca2", "BTCUSDT", rawJSON(`{}`))
	if id != "dca2-BTCUSDT" {
		t.Errorf("got %q, want %q", id, "dca2-BTCUSDT")
	}
}

func TestComputeInstanceID_PivotShort_WithInterval(t *testing.T) {
	id := computeInstanceID("pivotshort", "BTCUSDT", rawJSON(`{"interval":"1m"}`))
	if id != "pivotshort:BTCUSDT:1m" {
		t.Errorf("got %q, want %q", id, "pivotshort:BTCUSDT:1m")
	}
}

func TestComputeInstanceID_PivotShort_NoInterval(t *testing.T) {
	id := computeInstanceID("pivotshort", "BTCUSDT", rawJSON(`{}`))
	if id != "pivotshort:BTCUSDT" {
		t.Errorf("got %q, want %q", id, "pivotshort:BTCUSDT")
	}
}

func TestComputeInstanceID_UnknownStrategy(t *testing.T) {
	id := computeInstanceID("mystategy", "BTCUSDT", rawJSON(`{}`))
	if id != "mystategy:BTCUSDT" {
		t.Errorf("got %q, want %q", id, "mystategy:BTCUSDT")
	}
}

func TestComputeInstanceID_NullConfig(t *testing.T) {
	id := computeInstanceID("grid2", "ETHUSDT", json.RawMessage("null"))
	if id != "grid2-ETHUSDT-size-0--" {
		t.Errorf("got %q, want %q", id, "grid2-ETHUSDT-size-0--")
	}
}

// --- sanitizeForDocker ---

func TestSanitizeForDocker(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"BTCUSDT", "btcusdt"},
		{"grid2-BTCUSDT", "grid2-btcusdt"},
		{"Hello World", "hello-world"},
		{"UPPER_case-MIXED", "upper-case-mixed"},
		{"a--b---c", "a-b-c"},
		{"-leading", "leading"},
		{"trailing-", "trailing"},
		{"-both-", "both"},
		{"special!@#$chars", "special-chars"},
		{"", ""},
		{"123", "123"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := sanitizeForDocker(tt.input)
			if got != tt.want {
				t.Errorf("sanitizeForDocker(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

// --- instanceSlug ---

func TestInstanceSlug(t *testing.T) {
	slug := instanceSlug("user1234567890", "live", "grid2-BTCUSDT")
	want := "bbgo-user1234-live-grid2-btcusdt"
	if slug != want {
		t.Errorf("got %q, want %q", slug, want)
	}
}

func TestInstanceSlug_Truncation(t *testing.T) {
	longID := "very-long-strategy-name-BTCUSDT-with-lots-of-params-that-exceeds-the-limit"
	slug := instanceSlug("user1234567890", "live", longID)
	if len(slug) > 128 {
		t.Errorf("slug length %d exceeds 128: %q", len(slug), slug)
	}
}
