package main

import "testing"

func TestRunCLIUnknownCommandReturnsError(t *testing.T) {
	if err := runCLI([]string{"foo"}, nil); err == nil {
		t.Fatalf("expected unknown command error")
	}
}

func TestRunCLIMissingValueFlagReturnsError(t *testing.T) {
	if err := runCLI([]string{"list", "--sort"}, nil); err == nil {
		t.Fatalf("expected missing value flag error")
	}
}
