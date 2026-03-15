package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/sci-ecommerce/issuesherpa/internal/appconfig"
)

func TestLoadRuntimeConfigEnvOverridesFile(t *testing.T) {
	configRoot := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", configRoot)
	t.Setenv("HOME", configRoot)

	configPath, err := appconfig.DefaultPath()
	if err != nil {
		t.Fatalf("default config path: %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(configPath), 0o700); err != nil {
		t.Fatalf("mkdir config dir: %v", err)
	}
	if err := os.WriteFile(configPath, []byte(`
[sentry]
auth_token = "file-token"
org = "file-org"
projects = ["file-project"]

[gitlab]
token = "file-gitlab"
projects = ["file-group/project"]

[github]
token = "file-github"
repos = ["owner/file-repo"]
`), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	t.Setenv("SENTRY_AUTH_TOKEN", "env-token")
	t.Setenv("SENTRY_PROJECTS", "env-project")
	t.Setenv("GITHUB_REPOS", "owner/env-repo")

	cfg, err := loadRuntimeConfig()
	if err != nil {
		t.Fatalf("load runtime config: %v", err)
	}

	if cfg.SentryToken != "env-token" {
		t.Fatalf("sentry token = %q, want env override", cfg.SentryToken)
	}
	if cfg.SentryOrg != "file-org" {
		t.Fatalf("sentry org = %q, want file value", cfg.SentryOrg)
	}
	if len(cfg.SentryProjects) != 1 || cfg.SentryProjects[0] != "env-project" {
		t.Fatalf("sentry projects = %#v, want env override", cfg.SentryProjects)
	}
	if cfg.GitLabToken != "file-gitlab" {
		t.Fatalf("gitlab token = %q, want file value", cfg.GitLabToken)
	}
	if len(cfg.GitHubRepos) != 1 || cfg.GitHubRepos[0] != "owner/env-repo" {
		t.Fatalf("github repos = %#v, want env override", cfg.GitHubRepos)
	}
}
