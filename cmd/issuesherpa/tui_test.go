package main

import (
	"strings"
	"testing"

	"github.com/sci-ecommerce/issuesherpa/internal/core"
	"github.com/mattn/go-runewidth"
)

func TestTruncateTextHonorsDisplayWidth(t *testing.T) {
	got := truncateText("世世世", 4)
	if runewidth.StringWidth(got) > 4 {
		t.Fatalf("truncateText width = %d, want <= 4", runewidth.StringWidth(got))
	}

	got = truncateText("AB世", 3)
	if runewidth.StringWidth(got) > 3 {
		t.Fatalf("truncateText width = %d, want <= 3", runewidth.StringWidth(got))
	}
}

func TestWrapTextHonorsDisplayWidth(t *testing.T) {
	lines := wrapText("世世世世", 4)
	if len(lines) == 0 {
		t.Fatalf("expected wrapped lines")
	}
	for _, line := range lines {
		if runewidth.StringWidth(line) > 4 {
			t.Fatalf("wrapped line width = %d, want <= 4: %q", runewidth.StringWidth(line), line)
		}
	}
}

func TestDisplayFieldPadAndTruncate(t *testing.T) {
	got := displayField("世世世", 3)
	if runewidth.StringWidth(got) != 3 {
		t.Fatalf("displayField width = %d, want 3: %q", runewidth.StringWidth(got), got)
	}
}

func TestRenderIssueLineFitsWidthForWideAndNarrow(t *testing.T) {
	title := "this title has 世 and 🔥 mixed chars"
	title = "a " + title
	line := renderIssueLine(20, "open", "github", "abc-123", "my/project", title)
	if runewidth.StringWidth(line) > 20 {
		t.Fatalf("rendered line width = %d, want <= 20: %q", runewidth.StringWidth(line), line)
	}

	line = renderIssueLine(56, "resolved", "github", "abcdefgh-123", "long-project-name", title)
	if runewidth.StringWidth(line) > 56 {
		t.Fatalf("rendered line width = %d, want <= 56: %q", runewidth.StringWidth(line), line)
	}

	line = renderIssueLine(90, "resolved", "github", "abcdefgh-123", "long-project-name", title)
	if runewidth.StringWidth(line) > 90 {
		t.Fatalf("rendered line width = %d, want <= 90: %q", runewidth.StringWidth(line), line)
	}
}

func TestRefreshIssuesCapturesOfflineState(t *testing.T) {
	m := &model{offline: true, service: &core.Service{}}
	cmd := m.refreshIssues("ticket-1")
	m.offline = false

	msg, ok := cmd().(syncResultMsg)
	if !ok {
		t.Fatalf("expected syncResultMsg, got %T", msg)
	}
	if msg.err == nil || !strings.Contains(msg.err.Error(), "cannot refresh while --offline is enabled") {
		t.Fatalf("expected offline refresh guard error, got %v", msg.err)
	}
	if msg.preferredID != "ticket-1" {
		t.Fatalf("expected preferredID ticket-1, got %q", msg.preferredID)
	}
}

func TestRefreshIssuesCapturesServiceState(t *testing.T) {
	m := &model{offline: false, service: nil}
	cmd := m.refreshIssues("ticket-2")
	m.service = &core.Service{}

	msg, ok := cmd().(syncResultMsg)
	if !ok {
		t.Fatalf("expected syncResultMsg, got %T", msg)
	}
	if msg.err == nil || !strings.Contains(msg.err.Error(), "refresh unavailable") {
		t.Fatalf("expected service refresh unavailable error, got %v", msg.err)
	}
}

func TestDescribeSyncWarnings(t *testing.T) {
	if got := describeSyncWarnings([]core.Warning{{Message: "network"}}); got != "refreshed with warning: network" {
		t.Fatalf("unexpected singular warning message: %q", got)
	}

	if got := describeSyncWarnings([]core.Warning{
		{Message: "a"},
		{Message: "b"},
	}); got != "refreshed with 2 warnings" {
		t.Fatalf("unexpected plural warning message: %q", got)
	}
}

func TestStartRefreshNoopWhileAlreadyRefreshing(t *testing.T) {
	m := &model{refreshing: true}
	_, cmd := m.startRefresh("id")
	if cmd != nil {
		t.Fatalf("expected no-op when already refreshing")
	}
}
