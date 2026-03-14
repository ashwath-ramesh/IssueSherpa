package gitlab

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
	Iid       int    `json:"iid"`
	Title     string `json:"title"`
	State     string `json:"state"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
	WebURL    string `json:"web_url"`
	Author    struct {
		Name     string `json:"name"`
		Email    string `json:"email"`
		Username string `json:"username"`
	} `json:"author"`
	Assignee *struct {
		Name  string `json:"name"`
		Email string `json:"email"`
	} `json:"assignee"`
}

func NewClient(token string) *client {
	return &client{
		token:   token,
		baseURL: "https://gitlab.com/api/v4",
		http:    &http.Client{Timeout: 30 * time.Second},
		limiter: httpx.NewRateLimiter(200 * time.Millisecond),
	}
}

func (c *client) FetchAllIssues(ctx context.Context, projects []string) ([]models.Issue, error) {
	var (
		mu    sync.Mutex
		all   []models.Issue
		errCh = make(chan error, len(projects)*2)
		wg    sync.WaitGroup
	)

	for _, project := range projects {
		for _, state := range []string{"opened", "closed"} {
			wg.Add(1)
			go func(project, state string) {
				defer wg.Done()
				issues, err := c.fetchIssues(ctx, project, state)
				if err != nil {
					errCh <- fmt.Errorf("gitlab %s %s: %w", project, state, err)
					return
				}
				mu.Lock()
				all = append(all, issues...)
				mu.Unlock()
			}(project, state)
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

func (c *client) fetchIssues(ctx context.Context, project string, state string) ([]models.Issue, error) {
	path := fmt.Sprintf("/projects/%s/issues?state=%s&per_page=100", url.PathEscape(project), state)
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
		issues = append(issues, normalizeGitLabIssue(project, issue))
	}
	return issues, nil
}

func normalizeGitLabIssue(project string, raw issueResponse) models.Issue {
	reporter := raw.Author.Name
	if reporter == "" {
		reporter = raw.Author.Username
	}
	status := normalizeStatus(raw.State)
	assigned := (*models.AssignedTo)(nil)
	if raw.Assignee != nil {
		assigned = &models.AssignedTo{Name: raw.Assignee.Name, Email: raw.Assignee.Email}
	}

	return models.Issue{
		ID:         fmt.Sprintf("gitlab:%d", raw.ID),
		ShortID:    fmt.Sprintf("%s#%d", project, raw.Iid),
		Title:      raw.Title,
		Status:     status,
		Level:      "",
		Project:    models.Project{ID: "", Name: project, Slug: project},
		Count:      "",
		UserCount:  0,
		FirstSeen:  raw.CreatedAt,
		LastSeen:   raw.UpdatedAt,
		Reporter:   reporter,
		AssignedTo: assigned,
		Source:     "gitlab",
		URL:        raw.WebURL,
	}
}

func normalizeStatus(raw string) string {
	s := strings.ToLower(raw)
	switch s {
	case "opened", "reopened", "pending":
		return "open"
	case "closed", "resolved", "merged":
		return "resolved"
	default:
		return s
	}
}

func (c *client) getAllPages(ctx context.Context, path string) ([]json.RawMessage, error) {
	return httpx.FetchAllPages(ctx, httpx.PaginationConfig{
		Client:   c.http,
		BaseURL:  c.baseURL,
		Headers:  map[string]string{"PRIVATE-TOKEN": c.token},
		NextPage: func(linkHeader string) string { return httpx.NextPageURL(linkHeader) },
		Limiter:  c.limiter,
	}, path)
}
