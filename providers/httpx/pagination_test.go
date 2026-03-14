package httpx

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
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
