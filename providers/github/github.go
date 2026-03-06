package github

import (
	"encoding/json"
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
	return &client{token: token, baseURL: "https://api.github.com", http: &http.Client{Timeout: 30 * time.Second}}
}

func (c *client) FetchAllIssues(repos []string) ([]models.Issue, error) {
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
				issues, err := c.fetchIssues(repo, state)
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

	for err := range errCh {
		return nil, err
	}

	return all, nil
}

func (c *client) fetchIssues(repo string, state string) ([]models.Issue, error) {
	path := fmt.Sprintf("/repos/%s/issues?state=%s&per_page=100", url.PathEscape(repo), state)
	rawMessages, err := c.getAllPages(path)
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

func (c *client) getAllPages(path string) ([]json.RawMessage, error) {
	var all []json.RawMessage
	currentURL := c.baseURL + path

	for currentURL != "" {
		req, err := http.NewRequest("GET", currentURL, nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("Authorization", "Bearer "+c.token)
		req.Header.Set("Accept", "application/vnd.github+json")

		resp, err := c.http.Do(req)
		if err != nil {
			return nil, err
		}

		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return nil, fmt.Errorf("github API returned %d: %s", resp.StatusCode, string(body))
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
