package github

import "testing"

func TestNormalizeGitHubIssue(t *testing.T) {
	raw := issueResponse{
		ID:        42,
		Number:    7,
		Title:     "Broken checkout",
		State:     "closed",
		CreatedAt: "2026-03-10T10:00:00Z",
		UpdatedAt: "2026-03-11T10:00:00Z",
		URL:       "https://github.com/owner/repo/issues/7",
	}
	raw.User.Login = "alice"
	raw.Assignee = &struct {
		Login string `json:"login"`
	}{Login: "bob"}

	issue := normalizeGitHubIssue("owner/repo", raw)
	if issue.ID != "github:42" {
		t.Fatalf("unexpected id: %s", issue.ID)
	}
	if issue.ShortID != "owner/repo#7" {
		t.Fatalf("unexpected short id: %s", issue.ShortID)
	}
	if issue.Status != "resolved" {
		t.Fatalf("unexpected status: %s", issue.Status)
	}
	if issue.Reporter != "alice" {
		t.Fatalf("unexpected reporter: %s", issue.Reporter)
	}
	if issue.AssignedTo == nil || issue.AssignedTo.Name != "bob" {
		t.Fatalf("unexpected assignee: %#v", issue.AssignedTo)
	}
}
