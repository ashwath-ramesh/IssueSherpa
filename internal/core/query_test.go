package core

import (
	"testing"

	"github.com/sci-ecommerce/issuesherpa/models"
)

func TestApplyFilters(t *testing.T) {
	issues := []models.Issue{
		{
			ID:        "sentry:1",
			ShortID:   "sentry:ISSUE-1",
			Title:     "Checkout panic",
			Status:    "open",
			Project:   models.Project{Slug: "shop-api", Name: "shop-api"},
			Reporter:  "alice",
			Source:    "sentry",
			FirstSeen: "2026-03-10T10:00:00Z",
			LastSeen:  "2026-03-11T10:00:00Z",
		},
		{
			ID:        "github:2",
			ShortID:   "org/repo#2",
			Title:     "Checkout docs typo",
			Status:    "resolved",
			Project:   models.Project{Slug: "org/repo", Name: "org/repo"},
			Reporter:  "bob",
			Source:    "github",
			FirstSeen: "2026-03-12T10:00:00Z",
			LastSeen:  "2026-03-13T10:00:00Z",
		},
	}

	filtered := ApplyFilters(issues, IssueFilter{
		Search:   "checkout",
		Status:   "open",
		SortBy:   "created",
		SortDesc: true,
	})
	if len(filtered) != 1 {
		t.Fatalf("expected 1 issue, got %d", len(filtered))
	}
	if filtered[0].ID != "sentry:1" {
		t.Fatalf("expected sentry issue, got %s", filtered[0].ID)
	}
}

func TestFindIssueMatchesFullAndShortID(t *testing.T) {
	issues := []models.Issue{
		{ID: "gitlab:10", ShortID: "group/project#10"},
	}

	if got := FindIssue(issues, "gitlab:10"); got == nil || got.ID != "gitlab:10" {
		t.Fatalf("expected full ID match, got %#v", got)
	}
	if got := FindIssue(issues, "GROUP/PROJECT#10"); got == nil || got.ShortID != "group/project#10" {
		t.Fatalf("expected short ID match, got %#v", got)
	}
}

func TestFuzzyDistanceLimit(t *testing.T) {
	cases := []struct {
		searchLen int
		targetLen int
		expected  int
	}{
		{4, 4, 1},
		{5, 5, 1},
		{6, 6, 2},
		{9, 9, 2},
		{10, 10, 2},
		{10, 13, -1},
	}

	for _, tc := range cases {
		got := fuzzyDistanceLimit(tc.searchLen, tc.targetLen)
		if got != tc.expected {
			t.Fatalf("searchLen=%d targetLen=%d expected=%d got=%d", tc.searchLen, tc.targetLen, tc.expected, got)
		}
	}
}

func TestLevenshtein(t *testing.T) {
	tests := []struct {
		a        string
		b        string
		expected int
	}{
		{"kitten", "sitten", 1},
		{"sherpa", "sherif", 2},
		{"", "", 0},
		{"", "abc", 3},
	}

	for _, test := range tests {
		got := levenshtein(test.a, test.b)
		if got != test.expected {
			t.Fatalf("levenshtein(%q, %q) expected %d got %d", test.a, test.b, test.expected, got)
		}
	}
}

func TestMatchCandidate(t *testing.T) {
	if !matchCandidate("Checkout panic", "checkout", false) {
		t.Fatal("expected exact match")
	}

	if !matchCandidate("Checkout panic", "chechout", true) {
		t.Fatal("expected fuzzy match when enabled")
	}

	if matchCandidate("Checkout panic", "chechout", false) {
		t.Fatal("did not expect fuzzy match when disabled")
	}
}

func TestMatchCandidateShortTermHasNoFuzzyFallback(t *testing.T) {
	if !matchCandidate("checkout", "che", true) {
		t.Fatal("expected exact fallback for short terms")
	}

	if matchCandidate("checkout", "chxck", true) {
		t.Fatal("expected short-ish typo terms to require exact match only")
	}
}

func TestFuzzyCandidateMatchAndMatchesTermForIDs(t *testing.T) {
	issue := models.Issue{
		ID:       "sentry-1234",
		ShortID:  "sr-9876",
		Project:  models.Project{Slug: "proj", Name: "Project"},
		Reporter: "alice",
		Source:   "sentry",
	}

	if !matchesTerm(issue, "sentry-1235") {
		t.Fatal("expected fuzzy match against ID")
	}

	if !matchesTerm(issue, "sr-9877") {
		t.Fatal("expected fuzzy match against short ID")
	}

	if !fuzzyCandidateMatch("Checkout labels", "lables") {
		t.Fatal("expected fuzzy token match")
	}

	if fuzzyCandidateMatch("abc", "ab") {
		t.Fatal("expected no fuzzy match for short token")
	}
}
