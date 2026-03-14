package sentry

import "testing"

func TestNormalizeStatus(t *testing.T) {
	tests := map[string]string{
		"unresolved":   "open",
		"ignored":      "open",
		"muted":        "open",
		"reprocessing": "open",
		"resolved":     "resolved",
	}

	for input, want := range tests {
		if got := normalizeStatus(input); got != want {
			t.Fatalf("normalizeStatus(%q) = %q, want %q", input, got, want)
		}
	}
}
