package main

import (
	"github.com/sci-ecommerce/issuesherpa/internal/appconfig"
	"github.com/sci-ecommerce/issuesherpa/internal/core"
)

func loadRuntimeConfig() (core.Config, error) {
	fileConfig, _, err := appconfig.LoadDefault()
	if err != nil {
		return core.Config{}, err
	}

	cfg := core.Config{
		SentryToken:    fileConfig.Sentry.AuthToken,
		SentryOrg:      fileConfig.Sentry.Org,
		SentryProjects: append([]string(nil), fileConfig.Sentry.Projects...),
		GitLabToken:    fileConfig.GitLab.Token,
		GitLabProjects: append([]string(nil), fileConfig.GitLab.Projects...),
		GitHubToken:    fileConfig.GitHub.Token,
		GitHubRepos:    append([]string(nil), fileConfig.GitHub.Repos...),
	}

	if value := readEnvValue("SENTRY_AUTH_TOKEN"); value != "" {
		cfg.SentryToken = value
	}
	if value := readEnvValue("SENTRY_ORG"); value != "" {
		cfg.SentryOrg = value
	}
	if value := readCSVEnv("SENTRY_PROJECTS"); len(value) > 0 {
		cfg.SentryProjects = value
	}

	if value := readEnvValue("GITLAB_TOKEN"); value != "" {
		cfg.GitLabToken = value
	}
	if value := readCSVEnv("GITLAB_PROJECTS"); len(value) > 0 {
		cfg.GitLabProjects = value
	}

	if value := readEnvValue("GITHUB_TOKEN"); value != "" {
		cfg.GitHubToken = value
	}
	if value := readCSVEnv("GITHUB_REPOS"); len(value) > 0 {
		cfg.GitHubRepos = value
	}

	return cfg, nil
}
