package dashboard

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestCheckGitHubUpdateFindsNewRelease(t *testing.T) {
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Accept"); got != "application/vnd.github+json" {
			t.Fatalf("unexpected Accept header: %q", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"tag_name":"v2.1.0","name":"V2.1.0","body":"security improvements","html_url":"https://github.com/JayLee-sre/ZHIYU-WAF/releases/tag/v2.1.0","published_at":"2026-08-18T00:00:00Z"}`))
	}))
	defer api.Close()

	srv := &Server{buildVersion: "v2.0.0", githubReleaseURL: api.URL, githubHTTPClient: api.Client()}
	result := srv.checkGitHubUpdate(context.Background())
	if result.Status != "update_available" || !result.UpdateAvailable {
		t.Fatalf("expected available update, got %#v", result)
	}
	if result.LatestVersion != "v2.1.0" {
		t.Fatalf("expected latest version v2.1.0, got %q", result.LatestVersion)
	}
}

func TestCheckGitHubUpdateHandlesMissingRelease(t *testing.T) {
	api := httptest.NewServer(http.NotFoundHandler())
	defer api.Close()

	srv := &Server{buildVersion: "v2.0.0-dev", githubReleaseURL: api.URL, githubHTTPClient: api.Client()}
	result := srv.checkGitHubUpdate(context.Background())
	if result.Status != "no_release" || result.UpdateAvailable {
		t.Fatalf("expected no_release, got %#v", result)
	}
	if result.ReleaseURL != GitHubReleasesPageURL {
		t.Fatalf("expected releases page URL, got %q", result.ReleaseURL)
	}
}

func TestCheckGitHubUpdateCachesSuccessfulResponse(t *testing.T) {
	requests := 0
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		_, _ = w.Write([]byte(`{"tag_name":"v2.0.0","html_url":"https://example.test/release"}`))
	}))
	defer api.Close()

	srv := &Server{buildVersion: "v2.0.0", githubReleaseURL: api.URL, githubHTTPClient: api.Client()}
	first := srv.checkGitHubUpdate(context.Background())
	second := srv.checkGitHubUpdate(context.Background())
	if first.Cached || !second.Cached || requests != 1 {
		t.Fatalf("expected one upstream request and cached second result; first=%#v second=%#v requests=%d", first, second, requests)
	}
}

func TestCompareSemanticVersion(t *testing.T) {
	tests := []struct {
		left  string
		right string
		want  int
	}{
		{"v2.1.0", "v2.0.9", 1},
		{"v2.0.0", "v2.0.0-dev", 1},
		{"v2.0.0-rc.2", "v2.0.0-rc.10", -1},
		{"v2.0.0-alpha", "v2.0.0-1", 1},
	}
	for _, test := range tests {
		left, ok := parseSemanticVersion(test.left)
		if !ok {
			t.Fatalf("failed to parse left version %q", test.left)
		}
		right, ok := parseSemanticVersion(test.right)
		if !ok {
			t.Fatalf("failed to parse right version %q", test.right)
		}
		got := compareSemanticVersion(left, right)
		if (got > 0) != (test.want > 0) || (got < 0) != (test.want < 0) {
			t.Errorf("compare(%q, %q) = %d, want sign %d", test.left, test.right, got, test.want)
		}
	}
}

func TestCheckGitHubUpdateDoesNotCacheFailures(t *testing.T) {
	srv := &Server{
		buildVersion:     "v2.0.0",
		githubReleaseURL: "http://127.0.0.1:1/unavailable",
		githubHTTPClient: &http.Client{Timeout: 20 * time.Millisecond},
	}
	result := srv.checkGitHubUpdate(context.Background())
	if result.Status != "unavailable" || !srv.updateCacheExpires.IsZero() {
		t.Fatalf("expected uncached unavailable result, got %#v", result)
	}
}
