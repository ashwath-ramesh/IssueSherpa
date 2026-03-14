package sentry

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
	org     string
	baseURL string
	http    *http.Client
}

type issueResponse struct {
	ID        string          `json:"id"`
	ShortID   string          `json:"shortId"`
	Title     string          `json:"title"`
	Status    string          `json:"status"`
	Level     string          `json:"level"`
	Project   projectResponse `json:"project"`
	Count     string          `json:"count"`
	UserCount int             `json:"userCount"`
	FirstSeen string          `json:"firstSeen"`
	LastSeen  string          `json:"lastSeen"`
	Metadata  metadata        `json:"metadata"`
	Assigned  *assignedTo     `json:"assignedTo"`
	Permalink string          `json:"permalink"`
}

type projectResponse struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Slug string `json:"slug"`
}

type metadata struct {
	Name         string `json:"name"`
	ContactEmail string `json:"contact_email"`
}

type assignedTo struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

func NewClient(token, org string) *client {
	return &client{token: token, org: org, baseURL: "https://sentry.io/api/0", http: &http.Client{Timeout: 30 * time.Second}}
}

func (c *client) FetchAllIssues(ctx context.Context, projects []string) ([]models.Issue, error) {
	var (
		mu    sync.Mutex
		all   []models.Issue
		errCh = make(chan error, len(projects)*2)
		wg    sync.WaitGroup
	)

	for _, project := range projects {
		for _, status := range []string{"unresolved", "resolved"} {
			wg.Add(1)
			go func(project, status string) {
				defer wg.Done()
				issues, err := c.fetchIssues(ctx, project, status)
				if err != nil {
					errCh <- fmt.Errorf("sentry %s %s: %w", project, status, err)
					return
				}
				mu.Lock()
				all = append(all, issues...)
				mu.Unlock()
			}(project, status)
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

func (c *client) fetchIssues(ctx context.Context, project string, status string) ([]models.Issue, error) {
	query := fmt.Sprintf("is:%s", status)
	path := fmt.Sprintf("/projects/%s/%s/issues/?query=%s&per_page=100", url.PathEscape(c.org), url.PathEscape(project), url.QueryEscape(query))

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

		normalized := normalizeSentryIssue(issue)
		issues = append(issues, normalized)
	}

	return issues, nil
}

func normalizeSentryIssue(raw issueResponse) models.Issue {
	reporter := raw.Metadata.Name
	if reporter == "" {
		reporter = raw.Metadata.ContactEmail
	}
	if reporter == "" {
		reporter = "System/Automated"
	}

	status := normalizeStatus(raw.Status)

	assigned := (*models.AssignedTo)(nil)
	if raw.Assigned != nil {
		assigned = &models.AssignedTo{Name: raw.Assigned.Name, Email: raw.Assigned.Email}
	}

	return models.Issue{
		ID:         "sentry:" + raw.ID,
		ShortID:    "sentry:" + raw.ShortID,
		Title:      raw.Title,
		Status:     status,
		Level:      raw.Level,
		Project:    models.Project{ID: raw.Project.ID, Name: raw.Project.Name, Slug: raw.Project.Slug},
		Count:      raw.Count,
		UserCount:  raw.UserCount,
		FirstSeen:  raw.FirstSeen,
		LastSeen:   raw.LastSeen,
		Reporter:   reporter,
		AssignedTo: assigned,
		Source:     "sentry",
		URL:        raw.Permalink,
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
		req.Header.Set("Authorization", "Bearer "+c.token)

		resp, err := c.http.Do(req)
		if err != nil {
			return nil, err
		}

		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			return nil, fmt.Errorf("read response body: %w", err)
		}
		if resp.StatusCode != 200 {
			return nil, fmt.Errorf("sentry API returned %d: %s", resp.StatusCode, string(body))
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
		if strings.Contains(part, `rel="next"`) && strings.Contains(part, `results="true"`) {
			start := strings.Index(part, "<")
			end := strings.Index(part, ">")
			if start >= 0 && end > start {
				return part[start+1 : end]
			}
		}
	}
	return ""
}

func normalizeStatus(raw string) string {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "unresolved", "ignored", "muted", "reprocessing":
		return "open"
	case "resolved":
		return "resolved"
	default:
		return strings.ToLower(strings.TrimSpace(raw))
	}
}
