package github

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/sci-ecommerce/issuesherpa/models"
	"github.com/sci-ecommerce/issuesherpa/providers/httpx"
)

type client struct {
	token   string
	http    *http.Client
	baseURL string
	limiter *httpx.RateLimiter
}

type issueResponse struct {
	ID        int64  `json:"id"`
	Number    int    `json:"number"`
	Title     string `json:"title"`
	State     string `json:"state"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
	URL       string `json:"html_url"`
	User      struct {
		Login string `json:"login"`
	} `json:"user"`
	Assignee *struct {
		Login string `json:"login"`
	} `json:"assignee"`
	PullRequest *struct {
		URL string `json:"url"`
	} `json:"pull_request"`
}

func NewClient(token string) *client {
	return &client{
		token:   token,
		baseURL: "https://api.github.com",
		http:    &http.Client{Timeout: 30 * time.Second},
		limiter: httpx.NewRateLimiter(750 * time.Millisecond),
	}
}

func (c *client) FetchAllIssues(ctx context.Context, repos []string) ([]models.Issue, error) {
	var (
		mu    sync.Mutex
		all   []models.Issue
		errCh = make(chan error, len(repos)*2)
		wg    sync.WaitGroup
	)

	for _, repo := range repos {
		for _, state := range []string{"open", "closed"} {
			wg.Add(1)
			go func(repo, state string) {
				defer wg.Done()
				issues, err := c.fetchIssues(ctx, repo, state)
				if err != nil {
					errCh <- fmt.Errorf("github %s %s: %w", repo, state, err)
					return
				}
				mu.Lock()
				all = append(all, issues...)
				mu.Unlock()
			}(repo, state)
		}
	}

	wg.Wait()
	close(errCh)

	var errs []error
	for err := range errCh {
		errs = append(errs, err)
	}
	if len(errs) > 0 {
		return nil, errors.Join(errs...)
	}

	return all, nil
}

func (c *client) fetchIssues(ctx context.Context, repo string, state string) ([]models.Issue, error) {
	owner, repoName, err := splitRepository(repo)
	if err != nil {
		return nil, err
	}

	path := fmt.Sprintf("/repos/%s/%s/issues?state=%s&per_page=100", url.PathEscape(owner), url.PathEscape(repoName), state)
	rawMessages, err := c.getAllPages(ctx, path)
	if err != nil {
		return nil, err
	}

	issues := make([]models.Issue, 0, len(rawMessages))
	for _, raw := range rawMessages {
		var issue issueResponse
		if err := json.Unmarshal(raw, &issue); err != nil {
			return nil, err
		}
		if issue.PullRequest != nil {
			continue
		}
		issues = append(issues, normalizeGitHubIssue(repo, issue))
	}
	return issues, nil
}

func splitRepository(repo string) (string, string, error) {
	parts := strings.Split(repo, "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", fmt.Errorf("invalid repository format: %q (expected owner/repo)", repo)
	}
	return parts[0], parts[1], nil
}

func normalizeGitHubIssue(repo string, raw issueResponse) models.Issue {
	reporter := ""
	if raw.User.Login != "" {
		reporter = raw.User.Login
	}

	status := strings.ToLower(raw.State)
	if status == "open" {
		status = "open"
	} else if status == "closed" {
		status = "resolved"
	}

	assigned := (*models.AssignedTo)(nil)
	if raw.Assignee != nil {
		assigned = &models.AssignedTo{Name: raw.Assignee.Login}
	}

	return models.Issue{
		ID:         fmt.Sprintf("github:%d", raw.ID),
		ShortID:    fmt.Sprintf("%s#%d", repo, raw.Number),
		Title:      raw.Title,
		Status:     status,
		Level:      "",
		Project:    models.Project{ID: repo, Name: repo, Slug: repo},
		Count:      "",
		UserCount:  0,
		FirstSeen:  raw.CreatedAt,
		LastSeen:   raw.UpdatedAt,
		Reporter:   reporter,
		AssignedTo: assigned,
		Source:     "github",
		URL:        raw.URL,
	}
}

func (c *client) getAllPages(ctx context.Context, path string) ([]json.RawMessage, error) {
	return httpx.FetchAllPages(ctx, httpx.PaginationConfig{
		Client:  c.http,
		BaseURL: c.baseURL,
		Headers: map[string]string{
			"Authorization": "Bearer " + c.token,
			"Accept":        "application/vnd.github+json",
		},
		NextPage: func(linkHeader string) string { return httpx.NextPageURL(linkHeader) },
		Limiter:  c.limiter,
	}, path)
}
