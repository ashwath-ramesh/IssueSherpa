package core

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/sci-ecommerce/issuesherpa/internal/apppaths"
	"github.com/sci-ecommerce/issuesherpa/models"
	"github.com/sci-ecommerce/issuesherpa/providers/github"
	"github.com/sci-ecommerce/issuesherpa/providers/gitlab"
	"github.com/sci-ecommerce/issuesherpa/providers/sentry"
)

var (
	ErrNoCachedData  = errors.New("no cached data")
	ErrIssueNotFound = errors.New("issue not found")
)

type Config struct {
	DBPath         string
	SentryToken    string
	SentryOrg      string
	SentryProjects []string
	GitLabToken    string
	GitLabProjects []string
	GitHubToken    string
	GitHubRepos    []string
}

type Service struct {
	config   Config
	store    storeBackend
	fetchers providerFetchers
	warnings []Warning
}

type providerFetchers struct {
	sentry func(ctx context.Context, token, org string, projects []string) ([]models.Issue, error)
	gitlab func(ctx context.Context, token string, projects []string) ([]models.Issue, error)
	github func(ctx context.Context, token string, repos []string) ([]models.Issue, error)
}

type storeBackend interface {
	UpsertIssues([]models.Issue) error
	LoadIssues() ([]models.Issue, error)
	SaveLastSync(time.Time) error
	LoadCacheInfo() (CacheInfo, error)
	Close() error
}

type Warning struct {
	Source  string
	Message string
}

func New(config Config) (*Service, error) {
	return newWithFetchers(config, providerFetchers{
		sentry: func(ctx context.Context, token, org string, projects []string) ([]models.Issue, error) {
			return sentry.NewClient(token, org).FetchAllIssues(ctx, projects)
		},
		gitlab: func(ctx context.Context, token string, projects []string) ([]models.Issue, error) {
			return gitlab.NewClient(token).FetchAllIssues(ctx, projects)
		},
		github: func(ctx context.Context, token string, repos []string) ([]models.Issue, error) {
			return github.NewClient(token).FetchAllIssues(ctx, repos)
		},
	})
}

func newWithFetchers(config Config, fetchers providerFetchers) (*Service, error) {
	dbPath := config.DBPath
	if dbPath == "" {
		dbPath = defaultDBPath()
	}

	store, err := NewStore(dbPath)
	if err != nil {
		return nil, err
	}

	return &Service{
		config:   config,
		store:    store,
		fetchers: fetchers,
	}, nil
}

func (s *Service) Close() error {
	return s.store.Close()
}

func (s *Service) Warnings() []Warning {
	if len(s.warnings) == 0 {
		return nil
	}
	out := make([]Warning, len(s.warnings))
	copy(out, s.warnings)
	return out
}

func (s *Service) Sync(ctx context.Context) ([]models.Issue, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	s.warnings = nil

	issues, warnings, err := s.fetchAllIssuesFromProviders(ctx)
	if err != nil {
		return nil, err
	}

	issues = SortIssues(issues, "created", true)
	if err := s.store.UpsertIssues(issues); err != nil {
		warnings = append(warnings, Warning{
			Source:  "cache",
			Message: fmt.Sprintf("failed to save cache: %v", err),
		})
	} else if err := s.store.SaveLastSync(time.Now().UTC()); err != nil {
		warnings = append(warnings, Warning{
			Source:  "cache",
			Message: fmt.Sprintf("failed to update cache metadata: %v", err),
		})
	}
	s.warnings = warnings

	return issues, nil
}

func (s *Service) LoadCached(ctx context.Context) ([]models.Issue, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	issues, err := s.store.LoadIssues()
	if err != nil {
		return nil, err
	}
	if len(issues) == 0 {
		return nil, ErrNoCachedData
	}
	return issues, nil
}

func (s *Service) List(ctx context.Context, filter IssueFilter) ([]models.Issue, error) {
	issues, err := s.LoadCached(ctx)
	if err != nil {
		return nil, err
	}
	return ApplyFilters(issues, filter), nil
}

func (s *Service) Get(ctx context.Context, id string) (*models.Issue, error) {
	issues, err := s.LoadCached(ctx)
	if err != nil {
		return nil, err
	}

	issue := FindIssue(issues, id)
	if issue == nil {
		return nil, ErrIssueNotFound
	}
	return issue, nil
}

func (s *Service) Leaderboard(ctx context.Context, filter IssueFilter) ([]LeaderboardEntry, error) {
	issues, err := s.List(ctx, filter)
	if err != nil {
		return nil, err
	}
	return BuildLeaderboard(issues), nil
}

func (s *Service) CacheInfo(ctx context.Context) (CacheInfo, error) {
	if err := ctx.Err(); err != nil {
		return CacheInfo{}, err
	}
	return s.store.LoadCacheInfo()
}

func (s *Service) fetchAllIssuesFromProviders(ctx context.Context) ([]models.Issue, []Warning, error) {
	var (
		mu    sync.Mutex
		wg    sync.WaitGroup
		all   []models.Issue
		errMu sync.Mutex
		errs  []Warning
	)

	addIssues := func(source []models.Issue) {
		mu.Lock()
		defer mu.Unlock()
		all = append(all, source...)
	}
	addWarning := func(source string, err error) {
		if err == nil {
			return
		}
		errMu.Lock()
		defer errMu.Unlock()
		errs = append(errs, Warning{
			Source:  source,
			Message: err.Error(),
		})
	}

	if s.fetchers.sentry != nil && s.config.SentryToken != "" && s.config.SentryOrg != "" && len(s.config.SentryProjects) > 0 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			issues, err := s.fetchers.sentry(ctx, s.config.SentryToken, s.config.SentryOrg, s.config.SentryProjects)
			if err != nil {
				addWarning("sentry", err)
				return
			}
			addIssues(issues)
		}()
	}

	if s.fetchers.gitlab != nil && s.config.GitLabToken != "" && len(s.config.GitLabProjects) > 0 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			issues, err := s.fetchers.gitlab(ctx, s.config.GitLabToken, s.config.GitLabProjects)
			if err != nil {
				addWarning("gitlab", err)
				return
			}
			addIssues(issues)
		}()
	}

	if s.fetchers.github != nil && s.config.GitHubToken != "" && len(s.config.GitHubRepos) > 0 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			issues, err := s.fetchers.github(ctx, s.config.GitHubToken, s.config.GitHubRepos)
			if err != nil {
				addWarning("github", err)
				return
			}
			addIssues(issues)
		}()
	}

	wg.Wait()

	if len(all) == 0 && len(errs) > 0 {
		return nil, nil, fmt.Errorf("%s: %s", errs[0].Source, errs[0].Message)
	}
	return all, errs, nil
}

func defaultDBPath() string {
	if override := strings.TrimSpace(os.Getenv("ISSUESHERPA_DB_PATH")); override != "" {
		return override
	}
	path, err := apppaths.ResolveDBPath()
	if err != nil || strings.TrimSpace(path) == "" {
		return "issues.db"
	}
	return path
}
