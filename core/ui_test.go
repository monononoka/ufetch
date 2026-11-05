package core

import (
	"strings"
	"testing"
)

func TestFormatProgress(t *testing.T) {
	tests := []struct {
		name     string
		done     int64
		total    int64
		contains string
	}{
		{name: "halfway", done: 50, total: 100, contains: "[############------------]  50.0%"},
		{name: "clamped at complete", done: 120, total: 100, contains: "[########################] 100.0%"},
		{name: "unknown size", done: 1024, total: -1, contains: "[size unknown] 1.0 KB downloaded"},
		{name: "zero size", done: 0, total: 0, contains: "[size unknown] 0 B downloaded"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := FormatProgress(tt.done, tt.total); !strings.Contains(got, tt.contains) {
				t.Fatalf("FormatProgress() = %q, want it to contain %q", got, tt.contains)
			}
		})
	}
}
