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
