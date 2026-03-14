package main

import "testing"

func TestIsPlaceholderValueAllowsTodoInLegitValues(t *testing.T) {
	if isPlaceholderValue("") {
		t.Fatalf("expected empty value to be treated as not a placeholder")
	}
	if isPlaceholderValue("my-todo-app") {
		t.Fatalf("expected legitimate value to pass placeholder check")
	}
	if !isPlaceholderValue("todo") {
		t.Fatalf("expected bare todo to be treated as placeholder")
	}
}
