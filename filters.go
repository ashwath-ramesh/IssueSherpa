package main

import (
	"sort"
	"strings"
	"time"
)

type IssueFilter struct {
	Project  string
	Source   string
	Status   string
	Search   string
	SortBy   string
	SortDesc bool
}

const (
	defaultSortBy = "created"
)

func splitSearchTerms(value string) []string {
	parts := strings.Fields(value)
	return parts
}

func parseCSVList(value string) []string {
	parts := strings.Split(value, ",")
	var out []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		out = append(out, p)
	}
	return out
}

func buildIssueFilter(args []string) IssueFilter {
	return IssueFilter{
		Project:  extractFlag(args, "--project"),
		Source:   strings.ToLower(extractFlag(args, "--source")),
		SortBy:   normalizeSortBy(extractFlag(args, "--sort")),
		SortDesc: hasFlag(args, "--desc"),
		Search:   extractFlag(args, "--search"),
	}
}

func extractFlag(args []string, flag string) string {
	for i := 0; i < len(args); i++ {
		if args[i] != flag {
			continue
		}
		if i+1 < len(args) {
			return args[i+1]
		}
		return ""
	}
	return ""
}

func issueFilterFromArgs(args []string, cmd string) (IssueFilter, []string) {
	filter := buildIssueFilter(args)

	if hasFlag(args, "--open") {
		filter.Status = "open"
	}
	if hasFlag(args, "--resolved") {
		filter.Status = "resolved"
	}

	positional := extractPositionalArgs(args)
	if cmd == "search" && filter.Search == "" && len(positional) > 0 {
		filter.Search = strings.Join(positional, " ")
	}

	if filter.Status == "" {
		filter.SortBy = normalizeSortBy(filter.SortBy)
	} else {
		filter.SortBy = normalizeSortBy(filter.SortBy)
	}

	return filter, positional
}

func applyFilters(issues []Issue, filter IssueFilter) []Issue {
	filtered := make([]Issue, 0, len(issues))
	for _, i := range issues {
		if filter.Project != "" && !strings.EqualFold(i.Project.Slug, filter.Project) {
			continue
		}
		if filter.Source != "" && !strings.EqualFold(i.Source, filter.Source) {
			continue
		}
		if filter.Status != "" && !strings.EqualFold(i.Status, filter.Status) {
			continue
		}
		if filter.Search != "" && !matchesSearch(i, filter.Search) {
			continue
		}
		filtered = append(filtered, i)
	}

	return sortIssues(filtered, filter.SortBy, filter.SortDesc)
}

func matchesSearch(issue Issue, query string) bool {
	query = strings.ToLower(strings.TrimSpace(query))
	if query == "" {
		return true
	}
	parts := splitSearchTerms(query)

	for _, p := range parts {
		if !matchesTerm(issue, p) {
			return false
		}
	}
	return true
}

func matchesTerm(issue Issue, term string) bool {
	term = strings.ToLower(term)
	candidates := []string{
		issue.ID,
		issue.ShortID,
		issue.Title,
		issue.Project.Slug,
		issue.Project.Name,
		issue.Reporter,
		issue.Source,
	}
	for _, c := range candidates {
		if strings.Contains(strings.ToLower(c), term) {
			return true
		}
	}
	return false
}

func normalizeSortBy(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "created", "created_at", "firstseen", "first_seen", "first":
		return "created"
	case "updated", "updated_at", "last", "lastseen", "last_seen":
		return "updated"
	case "project", "repo", "repository", "path":
		return "project"
	case "reporter", "author":
		return "reporter"
	case "status":
		return "status"
	case "title", "subject":
		return "title"
	case "source":
		return "source"
	case "id":
		return "id"
	default:
		return defaultSortBy
	}
}

func sortIssues(issues []Issue, sortBy string, desc bool) []Issue {
	issues = append([]Issue(nil), issues...)
	if len(issues) <= 1 {
		return issues
	}

	sort.SliceStable(issues, func(i, j int) bool {
		cmp := compareIssues(sortBy, issues[i], issues[j])
		if desc {
			return cmp > 0
		}
		return cmp < 0
	})

	return issues
}

func compareIssues(sortBy string, a, b Issue) int {
	switch sortBy {
	case "updated":
		ta := parseIssueTime(a.LastSeen)
		tb := parseIssueTime(b.LastSeen)
		if ta.Equal(tb) {
			return strings.Compare(a.ShortID, b.ShortID)
		}
		if ta.Before(tb) {
			return -1
		}
		return 1
	case "project":
		if a.Project.Slug == b.Project.Slug {
			return strings.Compare(a.ShortID, b.ShortID)
		}
		return strings.Compare(strings.ToLower(a.Project.Slug), strings.ToLower(b.Project.Slug))
	case "reporter":
		if a.Reporter == b.Reporter {
			return strings.Compare(a.ShortID, b.ShortID)
		}
		return strings.Compare(strings.ToLower(a.Reporter), strings.ToLower(b.Reporter))
	case "status":
		if a.Status == b.Status {
			return strings.Compare(a.ShortID, b.ShortID)
		}
		return strings.Compare(strings.ToLower(a.Status), strings.ToLower(b.Status))
	case "title":
		if a.Title == b.Title {
			return strings.Compare(a.ShortID, b.ShortID)
		}
		return strings.Compare(strings.ToLower(a.Title), strings.ToLower(b.Title))
	case "source":
		if a.Source == b.Source {
			return strings.Compare(a.ShortID, b.ShortID)
		}
		return strings.Compare(strings.ToLower(a.Source), strings.ToLower(b.Source))
	case "id":
		if a.ID == b.ID {
			return strings.Compare(a.ShortID, b.ShortID)
		}
		return strings.Compare(strings.ToLower(a.ID), strings.ToLower(b.ID))
	default:
		ta := parseIssueTime(a.FirstSeen)
		tb := parseIssueTime(b.FirstSeen)
		if ta.Equal(tb) {
			return strings.Compare(a.ShortID, b.ShortID)
		}
		if ta.Before(tb) {
			return -1
		}
		return 1
	}
}

func parseIssueTime(value string) time.Time {
	if value == "" {
		return time.Time{}
	}
	if parsed, err := time.Parse(time.RFC3339, value); err == nil {
		return parsed
	}
	if parsed, err := time.Parse("2006-01-02T15:04:05", value); err == nil {
		return parsed
	}
	if len(value) >= 10 {
		if parsed, err := time.Parse("2006-01-02", value[:10]); err == nil {
			return parsed
		}
	}
	return time.Time{}
}

func uniqueSortedValues(values []string) []string {
	set := map[string]struct{}{}
	for _, value := range values {
		if value != "" {
			set[value] = struct{}{}
		}
	}

	out := make([]string, 0, len(set))
	for value := range set {
		out = append(out, value)
	}
	sort.Strings(out)
	return out
}

func extractPositionalArgs(args []string) []string {
	var positional []string
	for i := 0; i < len(args); i++ {
		a := args[i]
		if !strings.HasPrefix(a, "--") {
			positional = append(positional, a)
			continue
		}
		switch a {
		case "--project", "--source", "--sort", "--search":
			i++
		}
	}
	return positional
}

func filterBySource(issues []Issue, source string) []Issue {
	if source == "" {
		return issues
	}
	result := make([]Issue, 0)
	for _, i := range issues {
		if strings.EqualFold(i.Source, source) {
			result = append(result, i)
		}
	}
	return result
}

func filterByProject(issues []Issue, slug string) []Issue {
	if slug == "" {
		return issues
	}
	result := make([]Issue, 0)
	for _, i := range issues {
		if strings.EqualFold(i.Project.Slug, slug) {
			result = append(result, i)
		}
	}
	return result
}

func filterByStatus(issues []Issue, status string) []Issue {
	result := make([]Issue, 0)
	for _, i := range issues {
		if strings.EqualFold(i.Status, status) {
			result = append(result, i)
		}
	}
	return result
}

func filterBySearch(issues []Issue, query string) []Issue {
	if query == "" {
		return issues
	}
	result := make([]Issue, 0)
	for _, i := range issues {
		if matchesSearch(i, query) {
			result = append(result, i)
		}
	}
	return result
}

func collectProjects(issues []Issue) []string {
	vals := make([]string, 0, len(issues))
	for _, i := range issues {
		if i.Project.Slug != "" {
			vals = append(vals, i.Project.Slug)
		}
	}
	return uniqueSortedValues(vals)
}

func collectSources(issues []Issue) []string {
	vals := make([]string, 0, len(issues))
	for _, i := range issues {
		if i.Source != "" {
			vals = append(vals, i.Source)
		}
	}
	return uniqueSortedValues(vals)
}

func buildLeaderboard(issues []Issue) []struct {
	name  string
	count int
} {
	counts := map[string]int{}
	for _, i := range issues {
		reporter := i.Reporter
		if reporter == "" {
			reporter = "Unknown"
		}
		counts[reporter]++
	}

	result := make([]struct {
		name  string
		count int
	}, 0, len(counts))
	for n, c := range counts {
		result = append(result, struct {
			name  string
			count int
		}{name: n, count: c})
	}

	sort.Slice(result, func(i, j int) bool {
		if result[i].count != result[j].count {
			return result[i].count > result[j].count
		}
		return result[i].name < result[j].name
	})
	return result
}


func splitCSVList(value string) []string {
	return parseCSVList(value)
}
