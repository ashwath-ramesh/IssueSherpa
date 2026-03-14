package gitlab

import "testing"

func TestNormalizeGitLabIssue(t *testing.T) {
	raw := issueResponse{
		ID:        77,
		Iid:       9,
		Title:     "Broken checkout",
		State:     "closed",
		CreatedAt: "2026-03-10T10:00:00Z",
		UpdatedAt: "2026-03-11T10:00:00Z",
		WebURL:    "https://gitlab.com/group/project/-/issues/9",
		Author: struct {
			Name     string `json:"name"`
			Email    string `json:"email"`
			Username string `json:"username"`
		}{
			Name:     "Alice",
			Username: "alice",
		},
		Assignee: &struct {
			Name  string `json:"name"`
			Email string `json:"email"`
		}{
			Name:  "Bob",
			Email: "bob@example.com",
		},
	}

	issue := normalizeGitLabIssue("group/project", raw)
	if issue.ID != "gitlab:77" {
		t.Fatalf("unexpected id: %s", issue.ID)
	}
	if issue.ShortID != "group/project#9" {
		t.Fatalf("unexpected short id: %s", issue.ShortID)
	}
	if issue.Status != "resolved" {
		t.Fatalf("unexpected status: %s", issue.Status)
	}
	if issue.Reporter != "Alice" {
		t.Fatalf("unexpected reporter: %s", issue.Reporter)
	}
	if issue.AssignedTo == nil || issue.AssignedTo.Name != "Bob" {
		t.Fatalf("unexpected assignee: %#v", issue.AssignedTo)
	}
}
