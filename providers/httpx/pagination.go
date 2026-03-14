package httpx

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	DefaultMaxPages     = 100
	DefaultMaxRetries   = 3
	DefaultRetryBackoff = 500 * time.Millisecond
)

type RateLimiter struct {
	minInterval time.Duration
	mu          sync.Mutex
	next        time.Time
}

func NewRateLimiter(minInterval time.Duration) *RateLimiter {
	return &RateLimiter{minInterval: minInterval}
}

func (l *RateLimiter) Wait(ctx context.Context) error {
	if l == nil || l.minInterval <= 0 {
		return ctx.Err()
	}

	l.mu.Lock()
	wait := time.Until(l.next)
	now := time.Now()
	if wait <= 0 {
		l.next = now.Add(l.minInterval)
		l.mu.Unlock()
		return ctx.Err()
	}
	l.next = l.next.Add(l.minInterval)
	l.mu.Unlock()

	timer := time.NewTimer(wait)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

type PaginationConfig struct {
	Client        *http.Client
	BaseURL       string
	Headers       map[string]string
	NextPage      func(linkHeader string) string
	SuccessStatus func(code int) bool
	Limiter       *RateLimiter
	MaxPages      int
	MaxRetries    int
	RetryBackoff  time.Duration
}

func FetchAllPages(ctx context.Context, cfg PaginationConfig, path string) ([]json.RawMessage, error) {
	if cfg.Client == nil {
		return nil, fmt.Errorf("http client is required")
	}
	if cfg.NextPage == nil {
		return nil, fmt.Errorf("next page parser is required")
	}
	if cfg.SuccessStatus == nil {
		cfg.SuccessStatus = func(code int) bool {
			return code >= 200 && code < 300
		}
	}
	if cfg.MaxPages <= 0 {
		cfg.MaxPages = DefaultMaxPages
	}
	if cfg.MaxRetries <= 0 {
		cfg.MaxRetries = DefaultMaxRetries
	}
	if cfg.RetryBackoff <= 0 {
		cfg.RetryBackoff = DefaultRetryBackoff
	}

	var all []json.RawMessage
	currentURL := cfg.BaseURL + path
	pageCount := 0

	for currentURL != "" {
		pageCount++
		if pageCount > cfg.MaxPages {
			return nil, fmt.Errorf("pagination limit reached after %d pages", cfg.MaxPages)
		}

		body, headers, err := fetchPage(ctx, cfg, currentURL)
		if err != nil {
			return nil, err
		}

		var page []json.RawMessage
		if err := json.Unmarshal(body, &page); err != nil {
			return nil, err
		}
		all = append(all, page...)
		currentURL = cfg.NextPage(headers.Get("Link"))
	}

	return all, nil
}

func NextPageURL(linkHeader string, qualifiers ...string) string {
	if linkHeader == "" {
		return ""
	}
	for _, part := range strings.Split(linkHeader, ",") {
		part = strings.TrimSpace(part)
		if !strings.Contains(part, `rel="next"`) {
			continue
		}
		matched := true
		for _, qualifier := range qualifiers {
			if !strings.Contains(part, qualifier) {
				matched = false
				break
			}
		}
		if !matched {
			continue
		}
		start := strings.Index(part, "<")
		end := strings.Index(part, ">")
		if start >= 0 && end > start {
			return part[start+1 : end]
		}
	}
	return ""
}

func fetchPage(ctx context.Context, cfg PaginationConfig, url string) ([]byte, http.Header, error) {
	var lastErr error
	for attempt := 0; attempt <= cfg.MaxRetries; attempt++ {
		if err := cfg.Limiter.Wait(ctx); err != nil && err != context.Canceled && err != context.DeadlineExceeded {
			return nil, nil, err
		} else if err != nil {
			return nil, nil, err
		}

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			return nil, nil, err
		}
		for key, value := range cfg.Headers {
			req.Header.Set(key, value)
		}

		resp, err := cfg.Client.Do(req)
		if err != nil {
			lastErr = err
			if attempt == cfg.MaxRetries || ctx.Err() != nil {
				break
			}
			if err := sleepWithContext(ctx, retryDelay(cfg.RetryBackoff, attempt, "")); err != nil {
				return nil, nil, err
			}
			continue
		}

		body, readErr := io.ReadAll(resp.Body)
		resp.Body.Close()
		if readErr != nil {
			lastErr = fmt.Errorf("read response body: %w", readErr)
			if attempt == cfg.MaxRetries || ctx.Err() != nil {
				break
			}
			if err := sleepWithContext(ctx, retryDelay(cfg.RetryBackoff, attempt, "")); err != nil {
				return nil, nil, err
			}
			continue
		}

		if cfg.SuccessStatus(resp.StatusCode) {
			return body, resp.Header, nil
		}

		lastErr = fmt.Errorf("api returned %d: %s", resp.StatusCode, string(body))
		if !shouldRetry(resp.StatusCode) || attempt == cfg.MaxRetries {
			break
		}
		if err := sleepWithContext(ctx, retryDelay(cfg.RetryBackoff, attempt, resp.Header.Get("Retry-After"))); err != nil {
			return nil, nil, err
		}
	}

	return nil, nil, lastErr
}

func shouldRetry(statusCode int) bool {
	return statusCode == http.StatusTooManyRequests || statusCode == http.StatusRequestTimeout || statusCode >= 500
}

func retryDelay(base time.Duration, attempt int, retryAfter string) time.Duration {
	if retryAfter != "" {
		if seconds, err := strconv.Atoi(strings.TrimSpace(retryAfter)); err == nil && seconds > 0 {
			return time.Duration(seconds) * time.Second
		}
		if when, err := http.ParseTime(retryAfter); err == nil {
			if delay := time.Until(when); delay > 0 {
				return delay
			}
		}
	}

	delay := base
	for i := 0; i < attempt; i++ {
		delay *= 2
	}
	return delay
}

func sleepWithContext(ctx context.Context, delay time.Duration) error {
	timer := time.NewTimer(delay)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
