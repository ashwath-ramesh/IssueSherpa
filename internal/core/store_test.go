package core

import (
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
