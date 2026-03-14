package core

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/sci-ecommerce/issuesherpa/models"
)

func TestLoadCachedWithoutDataReturnsSentinel(t *testing.T) {
	svc, err := newWithFetchers(Config{
		DBPath: filepath.Join(t.TempDir(), "issues.db"),
	}, providerFetchers{})
	if err != nil {
		t.Fatalf("new service: %v", err)
	}
	defer svc.Close()

	_, err = svc.LoadCached(context.Background())
	if !errors.Is(err, ErrNoCachedData) {
		t.Fatalf("expected ErrNoCachedData, got %v", err)
	}
}

func TestSyncPersistsAndQueriesIssues(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "issues.db")
	svc, err := newWithFetchers(Config{
		DBPath:         dbPath,
		SentryToken:    "token",
		SentryOrg:      "acme",
		SentryProjects: []string{"shop-api"},
		GitLabToken:    "token",
		GitLabProjects: []string{"group/project"},
	}, providerFetchers{
		sentry: func(ctx context.Context, token, org string, projects []string) ([]models.Issue, error) {
			return []models.Issue{
				{
					ID:        "sentry:1",
					ShortID:   "sentry:ISSUE-1",
					Title:     "Checkout panic",
					Status:    "open",
					Project:   models.Project{Slug: "shop-api", Name: "shop-api"},
					Reporter:  "alice",
					Source:    "sentry",
					FirstSeen: "2026-03-11T10:00:00Z",
					LastSeen:  "2026-03-11T10:00:00Z",
				},
			}, nil
		},
		gitlab: func(ctx context.Context, token string, projects []string) ([]models.Issue, error) {
			return []models.Issue{
				{
					ID:        "gitlab:2",
					ShortID:   "group/project#22",
					Title:     "UI regression",
					Status:    "resolved",
					Project:   models.Project{Slug: "group/project", Name: "group/project"},
					Reporter:  "bob",
					Source:    "gitlab",
					FirstSeen: "2026-03-12T10:00:00Z",
					LastSeen:  "2026-03-13T10:00:00Z",
				},
			}, nil
		},
	})
	if err != nil {
		t.Fatalf("new service: %v", err)
	}
	defer svc.Close()

	issues, err := svc.Sync(context.Background())
	if err != nil {
		t.Fatalf("sync: %v", err)
	}
	if len(issues) != 2 {
		t.Fatalf("expected 2 synced issues, got %d", len(issues))
	}
	if issues[0].ID != "gitlab:2" {
		t.Fatalf("expected newest issue first, got %s", issues[0].ID)
	}

	filtered, err := svc.List(context.Background(), IssueFilter{
		Source:   "sentry",
		Status:   "open",
		Search:   "checkout",
		SortBy:   "created",
		SortDesc: true,
	})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(filtered) != 1 || filtered[0].ID != "sentry:1" {
		t.Fatalf("unexpected filtered issues: %#v", filtered)
	}

	issue, err := svc.Get(context.Background(), "group/project#22")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if issue.ID != "gitlab:2" {
		t.Fatalf("expected gitlab issue, got %s", issue.ID)
	}

	leaderboard, err := svc.Leaderboard(context.Background(), IssueFilter{SortBy: "created", SortDesc: true})
	if err != nil {
		t.Fatalf("leaderboard: %v", err)
	}
	if len(leaderboard) != 2 {
		t.Fatalf("expected 2 leaderboard entries, got %d", len(leaderboard))
	}
	if leaderboard[0].Name != "alice" {
		t.Fatalf("expected alphabetical tie-break, got %s", leaderboard[0].Name)
	}
}

func TestSyncReturnsFetchedIssuesWhenCacheWriteFails(t *testing.T) {
	svc := &Service{
		config: Config{
			SentryToken:    "token",
			SentryOrg:      "acme",
			SentryProjects: []string{"shop-api"},
		},
		store: failingStore{},
		fetchers: providerFetchers{
			sentry: func(ctx context.Context, token, org string, projects []string) ([]models.Issue, error) {
				return []models.Issue{
					{
						ID:        "sentry:1",
						ShortID:   "sentry:ISSUE-1",
						Title:     "Checkout panic",
						Status:    "open",
						Project:   models.Project{Slug: "shop-api", Name: "shop-api"},
						Reporter:  "alice",
						Source:    "sentry",
						FirstSeen: "2026-03-11T10:00:00Z",
						LastSeen:  "2026-03-11T10:00:00Z",
					},
				}, nil
			},
		},
	}

	issues, err := svc.Sync(context.Background())
	if err != nil {
		t.Fatalf("sync: %v", err)
	}
	if len(issues) != 1 || issues[0].ID != "sentry:1" {
		t.Fatalf("expected fetched issues despite cache failure, got %#v", issues)
	}
	if len(svc.Warnings()) != 1 || svc.Warnings()[0].Source != "cache" {
		t.Fatalf("expected cache warning, got %#v", svc.Warnings())
	}
}

func TestSyncPreservesPartialProviderFailuresAsWarnings(t *testing.T) {
	svc := &Service{
		config: Config{
			SentryToken:    "token",
			SentryOrg:      "acme",
			SentryProjects: []string{"shop-api"},
			GitHubToken:    "token",
			GitHubRepos:    []string{"owner/repo"},
		},
		store: noopStore{},
		fetchers: providerFetchers{
			sentry: func(ctx context.Context, token, org string, projects []string) ([]models.Issue, error) {
				return []models.Issue{
					{
						ID:        "sentry:1",
						ShortID:   "sentry:ISSUE-1",
						Title:     "Checkout panic",
						Status:    "open",
						Project:   models.Project{Slug: "shop-api", Name: "shop-api"},
						Reporter:  "alice",
						Source:    "sentry",
						FirstSeen: "2026-03-11T10:00:00Z",
						LastSeen:  "2026-03-11T10:00:00Z",
					},
				}, nil
			},
			github: func(ctx context.Context, token string, repos []string) ([]models.Issue, error) {
				return nil, errors.New("auth failed")
			},
		},
	}

	issues, err := svc.Sync(context.Background())
	if err != nil {
		t.Fatalf("sync: %v", err)
	}
	if len(issues) != 1 || issues[0].Source != "sentry" {
		t.Fatalf("expected successful provider data, got %#v", issues)
	}
	warnings := svc.Warnings()
	if len(warnings) != 1 {
		t.Fatalf("expected 1 warning, got %#v", warnings)
	}
	if warnings[0].Source != "github" {
		t.Fatalf("expected github warning, got %#v", warnings[0])
	}
}

type failingStore struct{}

func (failingStore) UpsertIssues([]models.Issue) error {
	return errors.New("write failed")
}

func (failingStore) LoadIssues() ([]models.Issue, error) {
	return nil, nil
}

func (failingStore) SaveLastSync(time.Time) error {
	return nil
}

func (failingStore) LoadCacheInfo() (CacheInfo, error) {
	return CacheInfo{}, nil
}

func (failingStore) Close() error {
	return nil
}

type noopStore struct{}

func (noopStore) UpsertIssues([]models.Issue) error {
	return nil
}

func (noopStore) LoadIssues() ([]models.Issue, error) {
	return nil, nil
}

func (noopStore) SaveLastSync(time.Time) error {
	return nil
}

func (noopStore) LoadCacheInfo() (CacheInfo, error) {
	return CacheInfo{}, nil
}

func (noopStore) Close() error {
	return nil
}
