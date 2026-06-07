package main

import "testing"

func TestPtrStr(t *testing.T) {
	s := "hello"
	p := ptrStr(s)
	if p == nil || *p != s {
		t.Errorf("ptrStr(%q) = %v, want %q", s, p, s)
	}
}

func TestParseUintOrZero(t *testing.T) {
	tests := []struct {
		input string
		want  uint64
	}{
		{"123", 123},
		{"0", 0},
		{"", 0},
		{"abc", 0},
		{"9999999999", 9999999999},
	}
	for _, tt := range tests {
		got := parseUintOrZero(tt.input)
		if got != tt.want {
			t.Errorf("parseUintOrZero(%q) = %d, want %d", tt.input, got, tt.want)
		}
	}
}
