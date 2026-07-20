package source

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestSearXNGSearch(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/search" ||
			request.URL.Query().Get("q") != "inference official documentation" ||
			request.URL.Query().Get("format") != "json" ||
			request.URL.Query().Get("safesearch") != "1" {
			t.Fatalf("unexpected request: %s", request.URL.String())
		}
		response.Header().Set("Content-Type", "application/json")
		_, _ = response.Write([]byte(`{"results":[
			{"title":"Official guide","url":"https://example.com/guide","content":"A useful guide","engine":"brave","publishedDate":"2026-07-20T10:00:00Z"},
			{"title":"Unsafe","url":"http://127.0.0.1/private","content":"ignored"}
		]}`))
	}))
	defer server.Close()
	client, err := NewSearXNG(SearXNGConfig{BaseURL: server.URL, Timeout: time.Second})
	if err != nil {
		t.Fatal(err)
	}
	results, err := client.Search(context.Background(), SearchRequest{
		Query: "inference official documentation", Language: "all",
		Category: "general", Page: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 || results[0].Title != "Official guide" ||
		results[0].Rank != 1 || len(results[0].Engines) != 1 ||
		results[0].PublishedAt == nil {
		t.Fatalf("results=%#v", results)
	}
}

func TestSearXNGFailures(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		body       string
		maxBytes   int64
		want       string
	}{
		{name: "format disabled", statusCode: http.StatusForbidden, want: "JSON format is disabled"},
		{name: "rate limited", statusCode: http.StatusTooManyRequests, want: "rate limited"},
		{name: "server failure", statusCode: http.StatusBadGateway, want: "HTTP 502"},
		{name: "malformed", statusCode: http.StatusOK, body: `{`, want: "malformed JSON"},
		{name: "oversized", statusCode: http.StatusOK, body: `{"results":[]}`, maxBytes: 4, want: "size limit"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
				response.WriteHeader(test.statusCode)
				_, _ = response.Write([]byte(test.body))
			}))
			defer server.Close()
			client, err := NewSearXNG(SearXNGConfig{
				BaseURL: server.URL, Timeout: time.Second,
				MaxResponseBytes: test.maxBytes,
			})
			if err != nil {
				t.Fatal(err)
			}
			_, err = client.Search(context.Background(), SearchRequest{Query: "test"})
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("err=%v, want %q", err, test.want)
			}
		})
	}
}

func TestSearXNGTimeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		<-request.Context().Done()
	}))
	defer server.Close()
	client, err := NewSearXNG(SearXNGConfig{
		BaseURL: server.URL, Timeout: 10 * time.Millisecond,
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = client.Search(context.Background(), SearchRequest{Query: "test"})
	if err == nil || !strings.Contains(err.Error(), "request failed") {
		t.Fatalf("err=%v, want timeout", err)
	}
}
