package appconfig

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
)

type Config struct {
	Sentry SentryConfig `toml:"sentry"`
	GitLab GitLabConfig `toml:"gitlab"`
	GitHub GitHubConfig `toml:"github"`
}

type SentryConfig struct {
	AuthToken string   `toml:"auth_token"`
	Org       string   `toml:"org"`
	Projects  []string `toml:"projects"`
}

type GitLabConfig struct {
	Token    string   `toml:"token"`
	Projects []string `toml:"projects"`
}

type GitHubConfig struct {
	Token string   `toml:"token"`
	Repos []string `toml:"repos"`
}

const defaultTemplate = `# IssueSherpa user config
#
# Fill in one or more providers, then run:
#   issuesherpa
#   issuesherpa list

[sentry]
auth_token = ""
org = ""
projects = []

[gitlab]
token = ""
projects = []

[github]
token = ""
repos = []
`

func DefaultPath() (string, error) {
	root, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("resolve config dir: %w", err)
	}
	return filepath.Join(root, "issuesherpa", "config.toml"), nil
}

func LoadDefault() (Config, string, error) {
	path, err := DefaultPath()
	if err != nil {
		return Config{}, "", err
	}
	cfg, err := Load(path)
	return cfg, path, err
}

func Load(path string) (Config, error) {
	var cfg Config
	if strings.TrimSpace(path) == "" {
		return cfg, errors.New("config path is required")
	}
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return cfg, fmt.Errorf("stat config: %w", err)
	}
	if _, err := toml.DecodeFile(path, &cfg); err != nil {
		return cfg, fmt.Errorf("decode config: %w", err)
	}
	cfg.normalize()
	return cfg, nil
}

func InitDefault() (string, bool, error) {
	path, err := DefaultPath()
	if err != nil {
		return "", false, err
	}
	created, err := Init(path)
	return path, created, err
}

func Init(path string) (bool, error) {
	if strings.TrimSpace(path) == "" {
		return false, errors.New("config path is required")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return false, fmt.Errorf("create config dir: %w", err)
	}
	if _, err := os.Stat(path); err == nil {
		return false, nil
	} else if !os.IsNotExist(err) {
		return false, fmt.Errorf("stat config: %w", err)
	}
	if err := os.WriteFile(path, []byte(defaultTemplate), 0o600); err != nil {
		return false, fmt.Errorf("write config: %w", err)
	}
	return true, nil
}

func (c *Config) normalize() {
	c.Sentry.AuthToken = strings.TrimSpace(c.Sentry.AuthToken)
	c.Sentry.Org = strings.TrimSpace(c.Sentry.Org)
	c.Sentry.Projects = normalizeList(c.Sentry.Projects)
	c.GitLab.Token = strings.TrimSpace(c.GitLab.Token)
	c.GitLab.Projects = normalizeList(c.GitLab.Projects)
	c.GitHub.Token = strings.TrimSpace(c.GitHub.Token)
	c.GitHub.Repos = normalizeList(c.GitHub.Repos)
}

func normalizeList(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	out := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		out = append(out, trimmed)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}
