package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/sci-ecommerce/issuesherpa/internal/core"
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

	providers := activeProviders(sentryToken, sentryOrg, sentryProjects, gitlabToken, gitlabProjects, githubToken, githubRepos)
	if !offline && len(providers) == 0 {
		logEvent("error", "config.providers_missing", "required", "sentry|gitlab|github")
		logEvent("error", "config.providers_hint", "sentry", "SENTRY_AUTH_TOKEN,SENTRY_ORG,SENTRY_PROJECTS", "gitlab", "GITLAB_TOKEN,GITLAB_PROJECTS", "github", "GITHUB_TOKEN,GITHUB_REPOS")
		os.Exit(1)
	}

	if !offline {
		logEvent("info", "providers.configured", "providers", strings.Join(providers, ","), "count", strconv.Itoa(len(providers)))
	}

	svc, err := core.New(core.Config{
		SentryToken:    sentryToken,
		SentryOrg:      sentryOrg,
		SentryProjects: sentryProjects,
		GitLabToken:    gitlabToken,
		GitLabProjects: gitlabProjects,
		GitHubToken:    githubToken,
		GitHubRepos:    githubRepos,
	})
	if err != nil {
		logEvent("error", "database.open_failed", "error", sanitizeTerminalText(err.Error()))
		os.Exit(1)
	}
	defer svc.Close()

	var issues []Issue
	cacheInfo, _ := svc.CacheInfo(context.Background())

	if offline {
		issues, err = svc.LoadCached(context.Background())
		if errors.Is(err, core.ErrNoCachedData) {
			logEvent("warn", "cache.not_found", "action", "run_without_offline_first")
			os.Exit(1)
		}
		if err != nil {
			logEvent("error", "cache.load_failed", "error", sanitizeTerminalText(err.Error()))
			os.Exit(1)
		}
		logEvent("info", "cache.load_start")
		logEvent("info", "cache.load_complete", "issues", strconv.Itoa(len(issues)))
		printCacheStatus(cacheInfo)
	} else {
		issues, err = svc.Sync(context.Background())
		if err != nil {
			logEvent("error", "sync.fetch_failed", "error", sanitizeTerminalText(err.Error()))
			os.Exit(1)
		}
		cacheInfo, _ = svc.CacheInfo(context.Background())
		for _, warning := range svc.Warnings() {
			logEvent("warn", "sync.warning", "source", sanitizeTerminalText(warning.Source), "message", sanitizeTerminalText(warning.Message))
		}

		logIssueDownloadSummary(issues, sentryProjects, gitlabProjects, githubRepos)
		logEvent("info", "sync.complete", "issues", strconv.Itoa(len(issues)))
	}

	if len(args) > 0 {
		if err := runCLI(args, issues); err != nil {
			exitCode := 1
			if cliErr, ok := err.(*cliError); ok {
				exitCode = cliErr.ExitCode
				if exitCode <= 0 {
					exitCode = 1
				}
			}
			os.Exit(exitCode)
		}
		return
	}

	p := tea.NewProgram(newModel(issues, cacheInfo, offline, svc), tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		logEvent("error", "tui.run_failed", "error", sanitizeTerminalText(err.Error()))
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
	if value == "" {
		return ""
	}
	if isPlaceholderValue(value) {
		logEvent("warn", "env.placeholder_ignored", "variable", name)
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
		logEvent("warn", "env.placeholder_ignored", "variable", name)
		return nil
	}
	values := parseCSVList(raw)
	out := make([]string, 0, len(values))
	for _, v := range values {
		if isPlaceholderValue(v) {
			logEvent("warn", "env.placeholder_entry_ignored", "variable", name, "value", strings.TrimSpace(v))
			continue
		}
		out = append(out, v)
	}
	return out
}

func isPlaceholderValue(value string) bool {
	v := strings.TrimSpace(strings.ToLower(value))
	if v == "xxx" || v == "yyy" || v == "zzz" || v == "placeholder" || v == "your_token" || v == "changeme" || v == "todo" {
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
		logEvent("warn", "sync.empty")
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

	logEvent("info", "download_check.start")
	for _, source := range []string{"sentry", "github", "gitlab"} {
		stat := sourceStats[source]
		logEvent("info", "download_check.source", "source", source, "total", strconv.Itoa(stat.total))
		if len(configured[source]) == 0 {
			continue
		}
		for _, project := range configured[source] {
			logEvent("info", "download_check.project", "source", source, "project", project, "issues", strconv.Itoa(stat.bySrc[project]))
		}
	}
}

func printCacheStatus(info core.CacheInfo) {
	if !info.HasSync {
		return
	}
	age := time.Since(info.LastSyncAt).Round(time.Minute)
	logEvent("info", "cache.status", "last_sync", info.LastSyncAt.UTC().Format(time.RFC3339), "age", age.String())
	if info.Stale {
		logEvent("warn", "cache.stale", "threshold", "24h")
	}
}

func logEvent(level string, event string, fields ...string) {
	parts := make([]string, 0, len(fields)/2)
	for i := 0; i+1 < len(fields); i += 2 {
		parts = append(parts, fmt.Sprintf("%s=%q", fields[i], sanitizeTerminalText(fields[i+1])))
	}
	if len(parts) > 0 {
		fmt.Fprintf(os.Stderr, "[%s] [%s] %s %s\n", time.Now().UTC().Format(time.RFC3339), strings.ToUpper(level), event, strings.Join(parts, " "))
		return
	}
	fmt.Fprintf(os.Stderr, "[%s] [%s] %s\n", time.Now().UTC().Format(time.RFC3339), strings.ToUpper(level), event)
}
