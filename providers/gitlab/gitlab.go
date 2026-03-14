package gitlab

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/sci-ecommerce/issuesherpa/models"
)

type client struct {
	token   string
	http    *http.Client
	baseURL string
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
	return &client{token: token, baseURL: "https://gitlab.com/api/v4", http: &http.Client{Timeout: 30 * time.Second}}
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
	var all []json.RawMessage
	currentURL := c.baseURL + path

	for currentURL != "" {
		req, err := http.NewRequestWithContext(ctx, "GET", currentURL, nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("PRIVATE-TOKEN", c.token)

		resp, err := c.http.Do(req)
		if err != nil {
			return nil, err
		}

		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			return nil, fmt.Errorf("read response body: %w", err)
		}
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return nil, fmt.Errorf("gitlab API returned %d: %s", resp.StatusCode, string(body))
		}

		var page []json.RawMessage
		if err := json.Unmarshal(body, &page); err != nil {
			return nil, err
		}
		all = append(all, page...)
		currentURL = getNextPageURL(resp.Header.Get("Link"))
	}

	return all, nil
}

func getNextPageURL(linkHeader string) string {
	if linkHeader == "" {
		return ""
	}
	for _, part := range strings.Split(linkHeader, ",") {
		part = strings.TrimSpace(part)
		if strings.Contains(part, `rel="next"`) {
			start := strings.Index(part, "<")
			end := strings.Index(part, ">")
			if start >= 0 && end > start {
				return part[start+1 : end]
			}
		}
	}
	return ""
}
