package appconfig

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestLoadMissingConfigReturnsZeroValue(t *testing.T) {
	cfg, err := Load(filepath.Join(t.TempDir(), "missing.toml"))
	if err != nil {
		t.Fatalf("load missing config: %v", err)
	}
	if cfg.Sentry.AuthToken != "" || cfg.GitLab.Token != "" || cfg.GitHub.Token != "" {
		t.Fatalf("expected zero-value config, got %#v", cfg)
	}
}

func TestLoadNormalizesWhitespace(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.toml")
	if err := os.WriteFile(path, []byte(`
[sentry]
auth_token = " token "
org = " org "
projects = [" a ", "", "b "]

[gitlab]
token = " gitlab "
projects = [" one "]

[github]
token = " github "
repos = [" repo "]
`), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if cfg.Sentry.AuthToken != "token" || cfg.Sentry.Org != "org" {
		t.Fatalf("unexpected sentry config: %#v", cfg.Sentry)
	}
	if len(cfg.Sentry.Projects) != 2 || cfg.Sentry.Projects[0] != "a" || cfg.Sentry.Projects[1] != "b" {
		t.Fatalf("unexpected sentry projects: %#v", cfg.Sentry.Projects)
	}
	if cfg.GitLab.Token != "gitlab" || len(cfg.GitLab.Projects) != 1 || cfg.GitLab.Projects[0] != "one" {
		t.Fatalf("unexpected gitlab config: %#v", cfg.GitLab)
	}
	if cfg.GitHub.Token != "github" || len(cfg.GitHub.Repos) != 1 || cfg.GitHub.Repos[0] != "repo" {
		t.Fatalf("unexpected github config: %#v", cfg.GitHub)
	}
}

func TestInitCreatesTemplateAndDoesNotOverwrite(t *testing.T) {
	path := filepath.Join(t.TempDir(), "issuesherpa", "config.toml")

	created, err := Init(path)
	if err != nil {
		t.Fatalf("init config: %v", err)
	}
	if !created {
		t.Fatal("expected config to be created")
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read config: %v", err)
	}
	if !strings.Contains(string(data), "[sentry]") {
		t.Fatalf("expected sentry template block, got %q", string(data))
	}

	if err := os.WriteFile(path, []byte("custom = true\n"), 0o600); err != nil {
		t.Fatalf("overwrite config for test: %v", err)
	}

	created, err = Init(path)
	if err != nil {
		t.Fatalf("re-init config: %v", err)
	}
	if created {
		t.Fatal("expected existing config to be preserved")
	}

	data, err = os.ReadFile(path)
	if err != nil {
		t.Fatalf("read existing config: %v", err)
	}
	if string(data) != "custom = true\n" {
		t.Fatalf("expected config to be preserved, got %q", string(data))
	}
}

func TestDefaultPathUsesXDGConfigHome(t *testing.T) {
	root := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", root)

	path, err := DefaultPath()
	if err != nil {
		t.Fatalf("default path: %v", err)
	}

	want := filepath.Join(root, "issuesherpa", "config.toml")
	if path != want {
		t.Fatalf("path = %q, want %q", path, want)
	}
}

func TestInitDefaultRespectsLegacyConfigOnDarwin(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("legacy config path only applies on darwin")
	}

	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", "")

	legacy := filepath.Join(home, "Library", "Application Support", "issuesherpa", "config.toml")
	if err := os.MkdirAll(filepath.Dir(legacy), 0o700); err != nil {
		t.Fatalf("mkdir legacy dir: %v", err)
	}
	if err := os.WriteFile(legacy, []byte("custom = true\n"), 0o600); err != nil {
		t.Fatalf("write legacy config: %v", err)
	}

	path, created, err := InitDefault()
	if err != nil {
		t.Fatalf("init default: %v", err)
	}
	if created {
		t.Fatal("expected legacy config to prevent new file creation")
	}
	if path != legacy {
		t.Fatalf("path = %q, want legacy %q", path, legacy)
	}
}
