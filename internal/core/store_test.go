package core

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestStoreCacheInfoRoundTrip(t *testing.T) {
	store, err := NewStore(filepath.Join(t.TempDir(), "issues.db"))
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	defer store.Close()

	now := time.Now().UTC().Truncate(time.Second)
	if err := store.SaveLastSync(now); err != nil {
		t.Fatalf("save last sync: %v", err)
	}

	info, err := store.LoadCacheInfo()
	if err != nil {
		t.Fatalf("load cache info: %v", err)
	}
	if !info.HasSync {
		t.Fatal("expected cache info to have sync timestamp")
	}
	if !info.LastSyncAt.Equal(now) {
		t.Fatalf("expected %s, got %s", now, info.LastSyncAt)
	}
}

func TestNewStoreUsesPrivatePermissions(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "cache")
	dbPath := filepath.Join(dir, "issues.db")

	store, err := NewStore(dbPath)
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	defer store.Close()

	dirInfo, err := os.Stat(dir)
	if err != nil {
		t.Fatalf("stat dir: %v", err)
	}
	if got := dirInfo.Mode().Perm(); got != 0o700 {
		t.Fatalf("expected dir mode 0700, got %o", got)
	}

	fileInfo, err := os.Stat(dbPath)
	if err != nil {
		t.Fatalf("stat db: %v", err)
	}
	if got := fileInfo.Mode().Perm(); got != 0o600 {
		t.Fatalf("expected db mode 0600, got %o", got)
	}
}

func TestNewStoreDoesNotChmodExistingParentDir(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "shared")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir dir: %v", err)
	}
	if err := os.Chmod(dir, 0o755); err != nil {
		t.Fatalf("chmod dir: %v", err)
	}

	store, err := NewStore(filepath.Join(dir, "issues.db"))
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	defer store.Close()

	info, err := os.Stat(dir)
	if err != nil {
		t.Fatalf("stat dir: %v", err)
	}
	if got := info.Mode().Perm(); got != 0o755 {
		t.Fatalf("expected existing dir mode 0755 to be preserved, got %o", got)
	}
}
