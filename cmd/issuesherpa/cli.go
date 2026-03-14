package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/sci-ecommerce/issuesherpa/internal/core"
)

func runCLI(args []string, issues []Issue) error {
	if len(args) == 0 {
		printCLIHelp()
		return nil
	}

	cmd := args[0]
	flags := args[1:]
	filter, positional, err := issueFilterFromArgs(flags, cmd)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Invalid filter options: %v\n", err)
		return err
	}

	switch cmd {
	case "list":
		cliListIssues(core.ApplyFilters(issues, filter), filter.Search)
	case "search":
		if filter.Search == "" {
			if len(positional) > 0 {
				filter.Search = strings.Join(positional, " ")
			}
		}
		if filter.Search == "" {
			fmt.Fprintln(os.Stderr, "usage: issuesherpa search <query>")
			fmt.Fprintln(os.Stderr, "or use --search <query>")
			return fmt.Errorf("search query is required")
		}
		cliListIssues(core.ApplyFilters(issues, filter), filter.Search)
	case "show":
		if len(positional) == 0 {
			fmt.Fprintln(os.Stderr, "usage: issuesherpa show <ISSUE-ID>")
			return fmt.Errorf("issue id is required")
		}
		if err := cliShowIssue(issues, positional[0]); err != nil {
			fmt.Fprintln(os.Stderr, err)
			return err
		}
	case "leaderboard":
		cliLeaderboard(core.ApplyFilters(issues, filter))
	case "json":
		if err := cliJSON(core.ApplyFilters(issues, filter)); err != nil {
			fmt.Fprintln(os.Stderr, err)
			return err
		}
	default:
		fmt.Fprintf(os.Stderr, "unknown command: %s\n", cmd)
		printCLIHelp()
		return fmt.Errorf("unknown command: %s", cmd)
	}

	return nil
}

func printCLIHelp() {
	fmt.Print(`issuesherpa - IssueSherpa: unified Sentry + GitLab + GitHub issues

Usage:
  issuesherpa                              Launch TUI
  issuesherpa list                         List all issues
  issuesherpa list --open                  List open issues only
  issuesherpa list --resolved              List resolved issues only
  issuesherpa search <query>               Search across title/project/reporter/id
  issuesherpa search --source github foo    Search GitHub issues for "foo"
  issuesherpa show <ISSUE-ID>              Show issue details
  issuesherpa leaderboard                  Show reporter leaderboard
  issuesherpa leaderboard --project <slug>  Leaderboard for a project
  issuesherpa json                         Output all issues as JSON
  issuesherpa json --project <slug>        JSON for a project

Filters and sorting:
  --project <slug>                        Filter by project/repo slug
  --source <sentry|gitlab|github>         Filter by source
  --search <query>                        Filter/search issues
  --sort <created|updated|project|reporter|status|title|source|id>
                                          Sort order
  --open                                  Show only open issues
  --resolved                              Show only resolved issues
  --desc                                  Sort descending
  --offline                               Use cached data only (no API calls)
`)
}

func cliListIssues(issues []Issue, search string) {
	if len(issues) == 0 {
		fmt.Println("No issues matched")
		if search != "" {
			fmt.Printf("Search: %q\n", search)
		}
		return
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintf(w, "SOURCE\tID\tSTATUS\tCREATED\tPROJECT\tREPORTER\tTITLE\n")
	fmt.Fprintf(w, "------\t--\t------\t-------\t-------\t--------\t-----\n")

	for _, i := range issues {
		status := "OPEN"
		if i.Status == "resolved" {
			status = "DONE"
		}
		title := truncateText(sanitizeTerminalText(i.Title), 68)
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
			sanitizeTerminalText(i.Source),
			sanitizeTerminalText(i.ShortID),
			status,
			FormatDate(i.FirstSeen),
			sanitizeTerminalText(i.Project.Slug),
			sanitizeTerminalText(i.Reporter),
			title)
	}
	w.Flush()
	fmt.Printf("\nTotal: %d issues\n", len(issues))
}

func cliShowIssue(issues []Issue, rawID string) error {
	issue := core.FindIssue(issues, rawID)
	if issue != nil {
		fmt.Printf("ID:          %s\n", sanitizeTerminalText(issue.ShortID))
		fmt.Printf("Source:      %s\n", sanitizeTerminalText(issue.Source))
		fmt.Printf("Project:     %s\n", sanitizeTerminalText(issue.Project.Slug))
		fmt.Printf("Title:       %s\n", sanitizeTerminalText(issue.Title))
		fmt.Printf("Status:      %s\n", sanitizeTerminalText(issue.Status))
		fmt.Printf("Reporter:    %s\n", sanitizeTerminalText(issue.Reporter))
		fmt.Printf("Events:      %s\n", sanitizeTerminalText(issue.Count))
		fmt.Printf("Users:       %d\n", issue.UserCount)
		fmt.Printf("Created:     %s\n", FormatDate(issue.FirstSeen))
		fmt.Printf("Last Seen:   %s\n", FormatDate(issue.LastSeen))
		if issue.URL != "" {
			fmt.Printf("URL:         %s\n", sanitizeTerminalText(issue.URL))
		}
		if issue.AssignedTo != nil {
			fmt.Printf("Assigned To: %s\n", sanitizeTerminalText(issue.AssignedTo.Name))
		}
		return nil
	}
	return fmt.Errorf("Issue %s not found", rawID)
}

func cliLeaderboard(issues []Issue) {
	entries := core.BuildLeaderboard(issues)
	total := len(issues)

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintf(w, "RANK\tREPORTER\tCOUNT\tPERCENT\tBAR\n")
	fmt.Fprintf(w, "----\t--------\t-----\t-------\t---\n")

	maxCount := 0
	if len(entries) > 0 {
		maxCount = entries[0].Count
	}

	for rank, e := range entries {
		pct := float64(e.Count) / float64(total) * 100
		barLen := 0
		if maxCount > 0 {
			barLen = e.Count * 30 / maxCount
		}
		bar := strings.Repeat("#", barLen)
		fmt.Fprintf(w, "%d\t%s\t%d\t%.1f%%\t%s\n",
			rank+1, sanitizeTerminalText(e.Name), e.Count, pct, bar)
	}
	w.Flush()
	fmt.Printf("\nTotal: %d issues from %d reporters\n", total, len(entries))
}

func cliJSON(issues []Issue) error {
	type cliIssueJSON struct {
		ID        string `json:"id"`
		Source    string `json:"source"`
		ShortID   string `json:"short_id"`
		Title     string `json:"title"`
		Status    string `json:"status"`
		Project   string `json:"project"`
		Reporter  string `json:"reporter"`
		URL       string `json:"url"`
		Events    string `json:"events"`
		Users     int    `json:"users"`
		FirstSeen string `json:"first_seen"`
		LastSeen  string `json:"last_seen"`
	}

	out := make([]cliIssueJSON, 0, len(issues))

	for _, i := range issues {
		out = append(out, cliIssueJSON{
			ID:        i.ID,
			Source:    i.Source,
			ShortID:   i.ShortID,
			Title:     i.Title,
			Status:    i.Status,
			Project:   i.Project.Slug,
			Reporter:  i.Reporter,
			URL:       i.URL,
			Events:    i.Count,
			Users:     i.UserCount,
			FirstSeen: i.FirstSeen,
			LastSeen:  i.LastSeen,
		})
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(out); err != nil {
		return fmt.Errorf("Error writing JSON output: %w", err)
	}

	return nil
}
