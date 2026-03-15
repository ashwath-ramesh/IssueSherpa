package apppaths

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

const appName = "issuesherpa"

func ConfigDir() (string, error) {
	if runtime.GOOS == "windows" {
		base := strings.TrimSpace(os.Getenv("APPDATA"))
		if base == "" {
			home, err := homeDir()
			if err != nil {
				return "", err
			}
			base = filepath.Join(home, "AppData", "Roaming")
		}
		return filepath.Join(base, appName), nil
	}

	base := strings.TrimSpace(os.Getenv("XDG_CONFIG_HOME"))
	if base != "" {
		if !filepath.IsAbs(base) {
			return "", fmt.Errorf("XDG_CONFIG_HOME must be absolute, got %q", base)
		}
	} else {
		home, err := homeDir()
		if err != nil {
			return "", err
		}
		base = filepath.Join(home, ".config")
	}
	return filepath.Join(base, appName), nil
}

func ConfigPath() (string, error) {
	dir, err := ConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "config.toml"), nil
}

func ResolveConfigPath() (string, error) {
	primary, err := ConfigPath()
	if err != nil {
		return "", err
	}
	if fileExists(primary) {
		return primary, nil
	}
	if legacy, ok, err := legacyConfigPath(); err == nil && ok && fileExists(legacy) {
		return legacy, nil
	}
	return primary, nil
}

func DataDir() (string, error) {
	if runtime.GOOS == "windows" {
		base := strings.TrimSpace(os.Getenv("APPDATA"))
		if base == "" {
			home, err := homeDir()
			if err != nil {
				return "", err
			}
			base = filepath.Join(home, "AppData", "Roaming")
		}
		return filepath.Join(base, appName), nil
	}

	base := strings.TrimSpace(os.Getenv("XDG_DATA_HOME"))
	if base != "" {
		if !filepath.IsAbs(base) {
			return "", fmt.Errorf("XDG_DATA_HOME must be absolute, got %q", base)
		}
	} else {
		home, err := homeDir()
		if err != nil {
			return "", err
		}
		base = filepath.Join(home, ".local", "share")
	}
	return filepath.Join(base, appName), nil
}

func ResolveDBPath() (string, error) {
	dir, err := DataDir()
	if err != nil {
		return "", err
	}
	primary := filepath.Join(dir, "issues.db")
	if fileExists(primary) {
		return primary, nil
	}
	if legacy, ok, err := legacyDBPath(); err == nil && ok && fileExists(legacy) {
		return legacy, nil
	}
	return primary, nil
}

func legacyConfigPath() (string, bool, error) {
	if runtime.GOOS != "darwin" || strings.TrimSpace(os.Getenv("XDG_CONFIG_HOME")) != "" {
		return "", false, nil
	}
	home, err := homeDir()
	if err != nil {
		return "", false, err
	}
	return filepath.Join(home, "Library", "Application Support", appName, "config.toml"), true, nil
}

func legacyDBPath() (string, bool, error) {
	if runtime.GOOS != "darwin" || strings.TrimSpace(os.Getenv("XDG_DATA_HOME")) != "" {
		return "", false, nil
	}
	home, err := homeDir()
	if err != nil {
		return "", false, err
	}
	return filepath.Join(home, "Library", "Application Support", appName, "issues.db"), true, nil
}

func homeDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(home) == "" {
		return "", errors.New("home directory is empty")
	}
	return home, nil
}

func fileExists(path string) bool {
	if strings.TrimSpace(path) == "" {
		return false
	}
	_, err := os.Stat(path)
	return err == nil
}
