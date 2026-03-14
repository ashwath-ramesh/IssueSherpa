package main

import "testing"

func TestSanitizeTerminalTextStripsControlSequences(t *testing.T) {
	input := "hello\x1b[31m red\x1b[0m \x1b]0;pwnd\aworld\nnext\t世界"
	got := sanitizeTerminalText(input)
	want := "hello red world next 世界"
	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}
