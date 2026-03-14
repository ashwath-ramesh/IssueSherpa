package main

import (
	"reflect"
	"testing"

	"github.com/sci-ecommerce/issuesherpa/internal/core"
)

func TestIssueFilterFromArgsBuildsSearchForSearchCmd(t *testing.T) {
	args := []string{"--project", "frontend", "error", "timeout"}
	filter, positional, err := issueFilterFromArgs(args, "search")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if filter.Project != "frontend" {
		t.Fatalf("project = %q, want %q", filter.Project, "frontend")
	}
	if filter.Search != "error timeout" {
		t.Fatalf("search = %q, want %q", filter.Search, "error timeout")
	}
	if !reflect.DeepEqual(positional, []string{"error", "timeout"}) {
		t.Fatalf("positional = %v", positional)
	}
}

func TestIssueFilterFromArgsMutuallyExclusiveStatusFlags(t *testing.T) {
	_, _, err := issueFilterFromArgs([]string{"--open", "--resolved"}, "list")
	if err == nil {
		t.Fatalf("expected error when both --open and --resolved are set")
	}
	if err.Error() != "--open and --resolved cannot be used together" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestIssueFilterFromArgsRejectsMissingValueFlags(t *testing.T) {
	_, _, err := issueFilterFromArgs([]string{"--project"}, "list")
	if err == nil {
		t.Fatalf("expected error for missing value")
	}
	if err.Error() != "--project requires a value" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestIssueFilterFromArgsRejectsMissingValueWhenNextIsFlag(t *testing.T) {
	_, _, err := issueFilterFromArgs([]string{"--project", "--source", "github"}, "list")
	if err == nil {
		t.Fatalf("expected error when next token is another flag")
	}
	if err.Error() != "--project requires a value" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestParseCSVListSkipsEmptyValues(t *testing.T) {
	got := parseCSVList(" alpha , , beta,,gamma ")
	want := []string{"alpha", "beta", "gamma"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("parseCSVList = %v, want %v", got, want)
	}
}

func TestBuildIssueFilterNormalizesSourceAndSort(t *testing.T) {
	args := []string{
		"--source", "GITHUB",
		"--sort", "Last_Seen",
		"--project", "repo-1",
		"--desc",
	}
	filter := buildIssueFilter(args)
	want := core.IssueFilter{
		Project:  "repo-1",
		Source:   "github",
		SortBy:   "updated",
		SortDesc: true,
	}
	if filter != want {
		t.Fatalf("buildIssueFilter = %#v, want %#v", filter, want)
	}
}

func TestExtractPositionalArgsSkipsFlagValues(t *testing.T) {
	args := []string{"alpha", "--project", "repo", "beta", "--search", "keyword", "gamma"}
	got := extractPositionalArgs(args)
	want := []string{"alpha", "beta", "gamma"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("extractPositionalArgs = %v, want %v", got, want)
	}
}

func TestIssueFilterFromArgsStatusFlags(t *testing.T) {
	openFilter, _, err := issueFilterFromArgs([]string{"--open"}, "list")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if openFilter.Status != "open" {
		t.Fatalf("open filter status = %q, want open", openFilter.Status)
	}

	resolvedFilter, _, err := issueFilterFromArgs([]string{"--resolved"}, "list")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resolvedFilter.Status != "resolved" {
		t.Fatalf("resolved filter status = %q, want resolved", resolvedFilter.Status)
	}
}
