package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"
)

func runCLI(args []string, issues []Issue) {
	if len(args) == 0 {
		printCLIHelp()
		return
	}

	cmd := args[0]
	flags := args[1:]
	filter, positional := issueFilterFromArgs(flags, cmd)

	switch cmd {
	case "list":
		cliListIssues(applyFilters(issues, filter), filter.Search)
	case "search":
		if filter.Search == "" {
			if len(positional) > 0 {
				filter.Search = strings.Join(positional, " ")
			}
		}
		if filter.Search == "" {
			fmt.Fprintln(os.Stderr, "usage: issuesherpa search <query>")
			fmt.Fprintln(os.Stderr, "or use --search <query>")
			os.Exit(1)
		}
		cliListIssues(applyFilters(issues, filter), filter.Search)
	case "show":
		if len(args) < 2 {
			fmt.Fprintln(os.Stderr, "usage: issuesherpa show <ISSUE-ID>")
			os.Exit(1)
		}
		cliShowIssue(issues, args[1])
	case "leaderboard":
		cliLeaderboard(applyFilters(issues, filter))
	case "json":
		cliJSON(applyFilters(issues, filter))
	default:
		printCLIHelp()
	}
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
		title := i.Title
		if len(title) > 68 {
			title = title[:65] + "..."
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
			i.Source,
			i.ShortID,
			status,
			FormatDate(i.FirstSeen),
			i.Project.Slug,
			i.Reporter,
			title)
	}
	w.Flush()
	fmt.Printf("\nTotal: %d issues\n", len(issues))
}

func cliShowIssue(issues []Issue, rawID string) {
	id := strings.ToUpper(rawID)
	for _, i := range issues {
		if strings.ToUpper(i.ShortID) == id || strings.ToUpper(i.ID) == id {
			fmt.Printf("ID:          %s\n", i.ShortID)
			fmt.Printf("Source:      %s\n", i.Source)
			fmt.Printf("Project:     %s\n", i.Project.Slug)
			fmt.Printf("Title:       %s\n", i.Title)
			fmt.Printf("Status:      %s\n", i.Status)
			fmt.Printf("Reporter:    %s\n", i.Reporter)
			fmt.Printf("Events:      %s\n", i.Count)
			fmt.Printf("Users:       %d\n", i.UserCount)
			fmt.Printf("Created:     %s\n", FormatDate(i.FirstSeen))
			fmt.Printf("Last Seen:   %s\n", FormatDate(i.LastSeen))
			if i.URL != "" {
				fmt.Printf("URL:         %s\n", i.URL)
			}
			if i.AssignedTo != nil {
				fmt.Printf("Assigned To: %s\n", i.AssignedTo.Name)
			}
			return
		}
	}
	fmt.Fprintf(os.Stderr, "Issue %s not found\n", rawID)
	os.Exit(1)
}

func cliLeaderboard(issues []Issue) {
	entries := buildLeaderboard(issues)
	total := len(issues)

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintf(w, "RANK\tREPORTER\tCOUNT\tPERCENT\tBAR\n")
	fmt.Fprintf(w, "----\t--------\t-----\t-------\t---\n")

	maxCount := 0
	if len(entries) > 0 {
		maxCount = entries[0].count
	}

	for rank, e := range entries {
		pct := float64(e.count) / float64(total) * 100
		barLen := 0
		if maxCount > 0 {
			barLen = e.count * 30 / maxCount
		}
		bar := strings.Repeat("#", barLen)
		fmt.Fprintf(w, "%d\t%s\t%d\t%.1f%%\t%s\n",
			rank+1, e.name, e.count, pct, bar)
	}
	w.Flush()
	fmt.Printf("\nTotal: %d issues from %d reporters\n", total, len(entries))
}

func cliJSON(issues []Issue) {
	out := make([]struct {
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
	} , 0, len(issues))

	for _, i := range issues {
		out = append(out, struct {
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
		}{
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
	_ = enc.Encode(out)
}
