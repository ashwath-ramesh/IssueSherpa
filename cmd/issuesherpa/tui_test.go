package main

import (
	"testing"

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
