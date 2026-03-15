package apppaths

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestConfigPathUsesXDGConfigHome(t *testing.T) {
	root := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", root)

	got, err := ConfigPath()
	if err != nil {
		t.Fatalf("config path: %v", err)
	}

	want := filepath.Join(root, "issuesherpa", "config.toml")
	if got != want {
		t.Fatalf("config path = %q, want %q", got, want)
	}
}

func TestConfigPathDefaultsToHomeDotConfig(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", "")

	got, err := ConfigPath()
	if err != nil {
		t.Fatalf("config path: %v", err)
	}

	want := filepath.Join(home, ".config", "issuesherpa", "config.toml")
	if got != want {
		t.Fatalf("config path = %q, want %q", got, want)
	}
}

func TestConfigPathRejectsRelativeXDGConfigHome(t *testing.T) {
	t.Setenv("XDG_CONFIG_HOME", "relative-config")

	if _, err := ConfigPath(); err == nil {
		t.Fatal("expected relative XDG_CONFIG_HOME to fail")
	}
}

func TestResolveDBPathUsesXDGDataHome(t *testing.T) {
	root := t.TempDir()
	t.Setenv("XDG_DATA_HOME", root)

	got, err := ResolveDBPath()
	if err != nil {
		t.Fatalf("db path: %v", err)
	}

	want := filepath.Join(root, "issuesherpa", "issues.db")
	if got != want {
		t.Fatalf("db path = %q, want %q", got, want)
	}
}

func TestResolveDBPathDefaultsToHomeDotLocalShare(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_DATA_HOME", "")

	got, err := ResolveDBPath()
	if err != nil {
		t.Fatalf("db path: %v", err)
	}

	want := filepath.Join(home, ".local", "share", "issuesherpa", "issues.db")
	if got != want {
		t.Fatalf("db path = %q, want %q", got, want)
	}
}

func TestResolveDBPathRejectsRelativeXDGDataHome(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", "relative-data")

	if _, err := ResolveDBPath(); err == nil {
		t.Fatal("expected relative XDG_DATA_HOME to fail")
	}
}

func TestResolveConfigPathFallsBackToLegacyDarwin(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("legacy fallback only applies on darwin")
	}

	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", "")

	legacy := filepath.Join(home, "Library", "Application Support", "issuesherpa", "config.toml")
	if err := os.MkdirAll(filepath.Dir(legacy), 0o700); err != nil {
		t.Fatalf("mkdir legacy config dir: %v", err)
	}
	if err := os.WriteFile(legacy, []byte("[github]\nrepos=[]\n"), 0o600); err != nil {
		t.Fatalf("write legacy config: %v", err)
	}

	got, err := ResolveConfigPath()
	if err != nil {
		t.Fatalf("resolve config path: %v", err)
	}
	if got != legacy {
		t.Fatalf("config path = %q, want legacy %q", got, legacy)
	}
}

func TestResolveDBPathFallsBackToLegacyDarwin(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("legacy fallback only applies on darwin")
	}

	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_DATA_HOME", "")

	legacy := filepath.Join(home, "Library", "Application Support", "issuesherpa", "issues.db")
	if err := os.MkdirAll(filepath.Dir(legacy), 0o700); err != nil {
		t.Fatalf("mkdir legacy data dir: %v", err)
	}
	if err := os.WriteFile(legacy, []byte("sqlite"), 0o600); err != nil {
		t.Fatalf("write legacy db: %v", err)
	}

	got, err := ResolveDBPath()
	if err != nil {
		t.Fatalf("resolve db path: %v", err)
	}
	if got != legacy {
		t.Fatalf("db path = %q, want legacy %q", got, legacy)
	}
}
