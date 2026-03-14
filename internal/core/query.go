package core

import (
	"sort"
	"strings"
	"time"

	"github.com/sci-ecommerce/issuesherpa/models"
)

type IssueFilter struct {
	Project  string
	Source   string
	Status   string
	Search   string
	SortBy   string
	SortDesc bool
}

type LeaderboardEntry struct {
	Name  string
	Count int
}

const DefaultSortBy = "created"

func ApplyFilters(issues []models.Issue, filter IssueFilter) []models.Issue {
	filtered := make([]models.Issue, 0, len(issues))
	for _, issue := range issues {
		if filter.Project != "" && !strings.EqualFold(issue.Project.Slug, filter.Project) {
			continue
		}
		if filter.Source != "" && !strings.EqualFold(issue.Source, filter.Source) {
			continue
		}
		if filter.Status != "" && !strings.EqualFold(issue.Status, filter.Status) {
			continue
		}
		if filter.Search != "" && !matchesSearch(issue, filter.Search) {
			continue
		}
		filtered = append(filtered, issue)
	}

	return SortIssues(filtered, filter.SortBy, filter.SortDesc)
}

func FilterByStatus(issues []models.Issue, status string) []models.Issue {
	result := make([]models.Issue, 0, len(issues))
	for _, issue := range issues {
		if strings.EqualFold(issue.Status, status) {
			result = append(result, issue)
		}
	}
	return result
}

func CollectProjects(issues []models.Issue) []string {
	values := make([]string, 0, len(issues))
	for _, issue := range issues {
		if issue.Project.Slug != "" {
			values = append(values, issue.Project.Slug)
		}
	}
	return uniqueSortedValues(values)
}

func CollectSources(issues []models.Issue) []string {
	values := make([]string, 0, len(issues))
	for _, issue := range issues {
		if issue.Source != "" {
			values = append(values, issue.Source)
		}
	}
	return uniqueSortedValues(values)
}

func BuildLeaderboard(issues []models.Issue) []LeaderboardEntry {
	counts := map[string]int{}
	for _, issue := range issues {
		reporter := issue.Reporter
		if reporter == "" {
			reporter = "Unknown"
		}
		counts[reporter]++
	}

	result := make([]LeaderboardEntry, 0, len(counts))
	for name, count := range counts {
		result = append(result, LeaderboardEntry{Name: name, Count: count})
	}

	sort.Slice(result, func(i, j int) bool {
		if result[i].Count != result[j].Count {
			return result[i].Count > result[j].Count
		}
		return result[i].Name < result[j].Name
	})

	return result
}

func FindIssue(issues []models.Issue, rawID string) *models.Issue {
	id := strings.ToUpper(strings.TrimSpace(rawID))
	for _, issue := range issues {
		if strings.ToUpper(issue.ShortID) == id || strings.ToUpper(issue.ID) == id {
			copy := issue
			return &copy
		}
	}
	return nil
}

func NormalizeSortBy(value string) string {
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
		return DefaultSortBy
	}
}

func SortIssues(issues []models.Issue, sortBy string, desc bool) []models.Issue {
	issues = append([]models.Issue(nil), issues...)
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

func matchesSearch(issue models.Issue, query string) bool {
	query = strings.ToLower(strings.TrimSpace(query))
	if query == "" {
		return true
	}

	for _, part := range strings.Fields(query) {
		if !matchesTerm(issue, part) {
			return false
		}
	}
	return true
}

func matchesTerm(issue models.Issue, term string) bool {
	term = strings.ToLower(term)
	candidates := []struct {
		value      string
		allowFuzzy bool
	}{
		{value: issue.ID, allowFuzzy: true},
		{value: issue.ShortID, allowFuzzy: true},
		{value: issue.Title, allowFuzzy: true},
		{value: issue.Project.Slug, allowFuzzy: true},
		{value: issue.Project.Name, allowFuzzy: true},
		{value: issue.Reporter, allowFuzzy: true},
		{value: issue.Source, allowFuzzy: true},
	}

	for _, candidate := range candidates {
		if matchCandidate(candidate.value, term, candidate.allowFuzzy) {
			return true
		}
	}
	return false
}

func matchCandidate(candidate, term string, allowFuzzy bool) bool {
	candidate = strings.ToLower(strings.TrimSpace(candidate))
	if candidate == "" {
		return false
	}
	if strings.Contains(candidate, term) {
		return true
	}
	if !allowFuzzy || len(term) < 4 {
		return false
	}
	return fuzzyCandidateMatch(candidate, term)
}

func fuzzyCandidateMatch(candidate, term string) bool {
	for _, token := range strings.Fields(candidate) {
		if token == "" || len(token) < 4 {
			continue
		}
		limit := fuzzyDistanceLimit(len(term), len(token))
		if limit < 0 {
			continue
		}
		if levenshtein(term, token) <= limit {
			return true
		}
	}
	return false
}

func fuzzyDistanceLimit(searchLen, targetLen int) int {
	diff := searchLen - targetLen
	if diff < 0 {
		diff = -diff
	}
	if diff > 2 {
		return -1
	}

	if searchLen <= 5 {
		return 1
	}
	return 2
}

func levenshtein(a, b string) int {
	if a == b {
		return 0
	}

	runesA := []rune(a)
	runesB := []rune(b)
	aLen := len(runesA)
	bLen := len(runesB)

	if aLen == 0 {
		return bLen
	}
	if bLen == 0 {
		return aLen
	}

	dp := make([]int, bLen+1)
	prev := make([]int, bLen+1)
	for j := 0; j <= bLen; j++ {
		prev[j] = j
	}

	for i := 1; i <= aLen; i++ {
		dp[0] = i
		prevDiag := prev[0]

		for j := 1; j <= bLen; j++ {
			deletion := prev[j] + 1
			insertion := dp[j-1] + 1
			substitution := prevDiag
			if runesA[i-1] != runesB[j-1] {
				substitution++
			}

			dp[j] = min(deletion, min(insertion, substitution))
			prevDiag = prev[j]
		}

		copy(prev, dp)
	}
	return dp[bLen]
}

func compareIssues(sortBy string, a, b models.Issue) int {
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
