package main

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

// GitHub's API limits this address: the release feed answers instead, and
// when both refuse, that is said as a limit, not as a network problem.
func TestReleasesFallBackToTheFeed(t *testing.T) {
	feedOK := true
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api":
			w.WriteHeader(http.StatusForbidden)
			_, _ = w.Write([]byte(`{"message":"API rate limit exceeded"}`))
		case "/feed":
			if !feedOK {
				w.WriteHeader(http.StatusTooManyRequests)
				return
			}
			_, _ = w.Write([]byte(`<feed><entry><link rel="alternate" type="text/html" href="https://github.com/wensuper/OmniLLM/releases/tag/v0.2.0-alpha.63"/></entry>` +
				`<entry><link rel="alternate" type="text/html" href="https://github.com/wensuper/OmniLLM/releases/tag/v0.2.0-alpha.62"/></entry></feed>`))
		}
	}))
	defer srv.Close()
	oldAPI, oldFeed := releasesURL, releasesAtomURL
	releasesURL, releasesAtomURL = srv.URL+"/api", srv.URL+"/feed"
	defer func() { releasesURL, releasesAtomURL = oldAPI, oldFeed }()
	tag, err := latestRelease(context.Background(), srv.Client())
	if err != nil || tag != "v0.2.0-alpha.63" {
		t.Fatalf("%q %v", tag, err)
	}
	feedOK = false
	if _, err := latestRelease(context.Background(), srv.Client()); !errors.Is(err, errRateLimited) {
		t.Fatalf("both refused: %v", err)
	}
}
