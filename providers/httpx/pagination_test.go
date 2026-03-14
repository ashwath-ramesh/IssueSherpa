package httpx

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
)

func TestFetchAllPagesRetriesAndPaginates(t *testing.T) {
	attempts := 0
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/page1":
			attempts++
			if attempts == 1 {
				w.Header().Set("Retry-After", "0")
				w.WriteHeader(http.StatusTooManyRequests)
				_, _ = w.Write([]byte(`rate limited`))
				return
			}
			w.Header().Set("Link", `<`+server.URL+`/page2>; rel="next"`)
			_, _ = w.Write([]byte(`[{"id":"1"}]`))
		case "/page2":
			_, _ = w.Write([]byte(`[{"id":"2"}]`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	items, err := FetchAllPages(context.Background(), PaginationConfig{
		Client:       server.Client(),
		BaseURL:      server.URL,
		NextPage:     func(link string) string { return NextPageURL(link) },
		Limiter:      NewRateLimiter(0),
		MaxPages:     5,
		MaxRetries:   2,
		RetryBackoff: 1,
	}, "/page1")
	if err != nil {
		t.Fatalf("fetch all pages: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(items))
	}

	var first map[string]string
	if err := json.Unmarshal(items[0], &first); err != nil {
		t.Fatalf("unmarshal first item: %v", err)
	}
	if first["id"] != "1" {
		t.Fatalf("expected first id 1, got %#v", first)
	}
}

func TestFetchAllPagesEnforcesMaxPages(t *testing.T) {
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Link", `<`+server.URL+`/loop>; rel="next"`)
		_, _ = w.Write([]byte(`[{"id":"1"}]`))
	}))
	defer server.Close()

	_, err := FetchAllPages(context.Background(), PaginationConfig{
		Client:     server.Client(),
		BaseURL:    server.URL,
		NextPage:   func(link string) string { return NextPageURL(link) },
		Limiter:    NewRateLimiter(0),
		MaxPages:   2,
		MaxRetries: 1,
	}, "/loop")
	if err == nil {
		t.Fatal("expected pagination limit error")
	}
}

func TestFetchAllPagesRejectsCrossOriginNextLink(t *testing.T) {
	var leaked atomic.Int32

	attacker := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		leaked.Add(1)
		t.Fatalf("unexpected attacker request with auth header %q", r.Header.Get("Authorization"))
	}))
	defer attacker.Close()

	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Link", `<`+attacker.URL+`/steal>; rel="next"`)
		_, _ = w.Write([]byte(`[{"id":"1"}]`))
	}))
	defer server.Close()

	_, err := FetchAllPages(context.Background(), PaginationConfig{
		Client:  server.Client(),
		BaseURL: server.URL,
		Headers: map[string]string{
			"Authorization": "Bearer secret-token",
		},
		NextPage:   func(link string) string { return NextPageURL(link) },
		Limiter:    NewRateLimiter(0),
		MaxPages:   5,
		MaxRetries: 1,
	}, "/page1")
	if err == nil {
		t.Fatal("expected cross-origin pagination error")
	}
	if !strings.Contains(err.Error(), "different origin") {
		t.Fatalf("expected origin error, got %v", err)
	}
	if leaked.Load() != 0 {
		t.Fatalf("expected no attacker requests, got %d", leaked.Load())
	}
}

func TestFetchAllPagesPreservesBasePathPrefixOnFirstRequest(t *testing.T) {
	var gotPath string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		_, _ = w.Write([]byte(`[{"id":"1"}]`))
	}))
	defer server.Close()

	_, err := FetchAllPages(context.Background(), PaginationConfig{
		Client:     server.Client(),
		BaseURL:    server.URL + "/api/v4",
		NextPage:   func(link string) string { return NextPageURL(link) },
		Limiter:    NewRateLimiter(0),
		MaxPages:   5,
		MaxRetries: 1,
	}, "/projects/foo/issues")
	if err != nil {
		t.Fatalf("fetch all pages: %v", err)
	}
	if gotPath != "/api/v4/projects/foo/issues" {
		t.Fatalf("expected prefixed api path, got %q", gotPath)
	}
}
