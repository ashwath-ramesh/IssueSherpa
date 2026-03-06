package main

import (
	"fmt"
	"os"
	"strings"
	"sync"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/sci-ecommerce/issuesherpa/providers/github"
	"github.com/sci-ecommerce/issuesherpa/providers/gitlab"
	"github.com/sci-ecommerce/issuesherpa/providers/sentry"
)

func main() {
	args := os.Args[1:]

	if len(args) > 0 && (args[0] == "help" || args[0] == "--help" || args[0] == "-h") {
		printCLIHelp()
		return
	}

	offline := hasFlag(args, "--offline")
	args = stripFlags(args, "--offline")

	sentryToken := readEnvValue("SENTRY_AUTH_TOKEN")
	sentryOrg := readEnvValue("SENTRY_ORG")
	sentryProjects := readCSVEnv("SENTRY_PROJECTS")

	gitlabToken := readEnvValue("GITLAB_TOKEN")
	gitlabProjects := readCSVEnv("GITLAB_PROJECTS")

	githubToken := readEnvValue("GITHUB_TOKEN")
	githubRepos := readCSVEnv("GITHUB_REPOS")

	if !offline && len(activeProviders(sentryToken, sentryOrg, sentryProjects, gitlabToken, gitlabProjects, githubToken, githubRepos)) == 0 {
		fmt.Fprintln(os.Stderr, "Missing source configuration. Configure at least one provider:")
		fmt.Fprintln(os.Stderr, "  Sentry: SENTRY_AUTH_TOKEN, SENTRY_ORG, SENTRY_PROJECTS")
		fmt.Fprintln(os.Stderr, "  GitLab: GITLAB_TOKEN, GITLAB_PROJECTS")
		fmt.Fprintln(os.Stderr, "  GitHub: GITHUB_TOKEN, GITHUB_REPOS")
		fmt.Fprintln(os.Stderr, "Set the required environment variables above and rerun")
		os.Exit(1)
	}

	if !offline {
		fmt.Fprintf(os.Stderr, "Enabled providers: %s\n", formatProviders(sentryToken, sentryOrg, sentryProjects, gitlabToken, gitlabProjects, githubToken, githubRepos))
	}

	store, err := NewStore("issues.db")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error opening database: %v\n", err)
		os.Exit(1)
	}
	defer store.Close()

	var issues []Issue

	if offline {
		if !store.HasData() {
			fmt.Fprintln(os.Stderr, "No cached data. Run without --offline first to sync.")
			os.Exit(1)
		}
		fmt.Fprintln(os.Stderr, "Loading from cache...")
		issues, err = store.LoadIssues()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error loading cache: %v\n", err)
			os.Exit(1)
		}
		fmt.Fprintf(os.Stderr, "Loaded %d issues from cache.\n", len(issues))
	} else {
	issues, err = fetchAllIssuesFromProviders(
		sentryToken,
		sentryOrg,
		sentryProjects,
			gitlabToken,
			gitlabProjects,
			githubToken,
			githubRepos,
		)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error fetching issues: %v\n", err)
			os.Exit(1)
		}

		logIssueDownloadSummary(issues, sentryProjects, gitlabProjects, githubRepos)

		issues = sortIssues(issues, "created", true)

		if err := store.UpsertIssues(issues); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to save cache: %v\n", err)
		}

		fmt.Fprintf(os.Stderr, "Ready. %d issues synced.\n", len(issues))
	}

	if len(args) > 0 {
		runCLI(args, issues)
		return
	}

	p := tea.NewProgram(newModel(issues), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error running TUI: %v\n", err)
		os.Exit(1)
	}
}

func hasFlag(args []string, flag string) bool {
	for _, a := range args {
		if a == flag {
			return true
		}
	}
	return false
}

func stripFlags(args []string, flags ...string) []string {
	flagSet := map[string]bool{}
	for _, f := range flags {
		flagSet[f] = true
	}
	var result []string
	for _, a := range args {
		if !flagSet[a] {
			result = append(result, a)
		}
	}
	return result
}

func activeProviders(sentryToken, sentryOrg string, sentryProjects []string, gitlabToken string, gitlabProjects []string, githubToken string, githubRepos []string) []string {
	providers := make([]string, 0, 3)
	if sentryToken != "" && sentryOrg != "" && len(sentryProjects) > 0 {
		providers = append(providers, "sentry")
	}
	if gitlabToken != "" && len(gitlabProjects) > 0 {
		providers = append(providers, "gitlab")
	}
	if githubToken != "" && len(githubRepos) > 0 {
		providers = append(providers, "github")
	}
	return providers
}

func formatProviders(sentryToken, sentryOrg string, sentryProjects []string, gitlabToken string, gitlabProjects []string, githubToken string, githubRepos []string) string {
	providers := activeProviders(sentryToken, sentryOrg, sentryProjects, gitlabToken, gitlabProjects, githubToken, githubRepos)
	if len(providers) == 0 {
		return "(none)"
	}
	return strings.Join(providers, ", ")
}

func readEnvValue(name string) string {
	value := strings.TrimSpace(os.Getenv(name))
	if isPlaceholderValue(value) {
		fmt.Fprintf(os.Stderr, "Ignoring %s: placeholder value detected\n", name)
		return ""
	}
	return value
}

func readCSVEnv(name string) []string {
	raw := strings.TrimSpace(os.Getenv(name))
	if raw == "" {
		return nil
	}
	if isPlaceholderValue(raw) {
		fmt.Fprintf(os.Stderr, "Ignoring %s: placeholder value detected\n", name)
		return nil
	}
	values := splitCSVList(raw)
	out := make([]string, 0, len(values))
	for _, v := range values {
		if isPlaceholderValue(v) {
			fmt.Fprintf(os.Stderr, "Ignoring %s entry %q: placeholder value\n", name, v)
			continue
		}
		out = append(out, v)
	}
	return out
}

func isPlaceholderValue(value string) bool {
	v := strings.TrimSpace(strings.ToLower(value))
	if v == "" {
		return true
	}
	if v == "xxx" || v == "yyy" || v == "zzz" || strings.Contains(v, "placeholder") || strings.Contains(v, "your_token") || strings.Contains(v, "changeme") || strings.Contains(v, "todo") {
		return true
	}
	if len(v) <= 8 && isRepeatedCharValue(v) {
		return true
	}
	return false
}

func isRepeatedCharValue(value string) bool {
	if len(value) < 3 {
		return false
	}
	for i := 1; i < len(value); i++ {
		if value[i] != value[0] {
			return false
		}
	}
	return true
}

func logIssueDownloadSummary(issues []Issue, sentryProjects, gitlabProjects, githubRepos []string) {
	if len(issues) == 0 {
		fmt.Fprintln(os.Stderr, "Warning: no issues were downloaded.")
		return
	}

	type srcStat struct {
		total int
		bySrc map[string]int
	}

	sourceStats := map[string]*srcStat{
		"sentry": {bySrc: map[string]int{}},
		"github": {bySrc: map[string]int{}},
		"gitlab": {bySrc: map[string]int{}},
	}

	for _, i := range issues {
		source := strings.ToLower(strings.TrimSpace(i.Source))
		if source == "" {
			source = "sentry"
		}
		if stat, ok := sourceStats[source]; ok {
			stat.total++
			stat.bySrc[i.Project.Slug]++
		}
	}

	configured := map[string][]string{
		"sentry": sentryProjects,
		"github": githubRepos,
		"gitlab": gitlabProjects,
	}

	fmt.Fprintln(os.Stderr, "\nDownload check (per configured project/repo):")
	for _, source := range []string{"sentry", "github", "gitlab"} {
		stat := sourceStats[source]
		fmt.Fprintf(os.Stderr, "- %s: %d total issues\n", source, stat.total)
		if len(configured[source]) == 0 {
			continue
		}
		for _, project := range configured[source] {
			fmt.Fprintf(os.Stderr, "  %s: %d issues\n", project, stat.bySrc[project])
		}
	}
}

func fetchAllIssuesFromProviders(
	sentryToken, sentryOrg string,
	sentryProjects []string,
	gitlabToken string,
	gitlabProjects []string,
	githubToken string,
	githubRepos []string,
) ([]Issue, error) {
	var (
		mu    sync.Mutex
		wg    sync.WaitGroup
		all   []Issue
		errMu sync.Mutex
		errs  []error
	)

	addIssues := func(source []Issue) {
		mu.Lock()
		defer mu.Unlock()
		all = append(all, source...)
	}
	addErr := func(err error) {
		if err == nil {
			return
		}
		errMu.Lock()
		defer errMu.Unlock()
		errs = append(errs, err)
	}

	if sentryToken != "" && sentryOrg != "" && len(sentryProjects) > 0 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			client := sentry.NewClient(sentryToken, sentryOrg)
			issues, err := client.FetchAllIssues(sentryProjects)
			if err != nil {
				addErr(fmt.Errorf("sentry: %w", err))
				return
			}
			addIssues(issues)
		}()
	}

	if gitlabToken != "" && len(gitlabProjects) > 0 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			client := gitlab.NewClient(gitlabToken)
			issues, err := client.FetchAllIssues(gitlabProjects)
			if err != nil {
				addErr(fmt.Errorf("gitlab: %w", err))
				return
			}
			addIssues(issues)
		}()
	}

	if githubToken != "" && len(githubRepos) > 0 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			client := github.NewClient(githubToken)
			issues, err := client.FetchAllIssues(githubRepos)
			if err != nil {
				addErr(fmt.Errorf("github: %w", err))
				return
			}
			addIssues(issues)
		}()
	}

	wg.Wait()

	if len(all) == 0 && len(errs) > 0 {
		return nil, errs[0]
	}

	for _, err := range errs {
		fmt.Fprintf(os.Stderr, "Warning: %v\n", err)
	}

	return all, nil
}
