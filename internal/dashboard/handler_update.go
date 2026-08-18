package dashboard

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// BuildVersion may be overridden during a release build with -ldflags -X.
// A dev suffix intentionally compares lower than the corresponding stable
// tag, so an official release remains visible to development deployments.
var BuildVersion = "v2.0.0-dev"

const (
	GitHubLatestReleaseURL = "https://api.github.com/repos/JayLee-sre/ZHIYU-WAF/releases/latest"
	GitHubReleasesPageURL  = "https://github.com/JayLee-sre/ZHIYU-WAF/releases"
	updateCheckCacheTTL    = 10 * time.Minute
)

type githubRelease struct {
	TagName     string `json:"tag_name"`
	Name        string `json:"name"`
	Body        string `json:"body"`
	HTMLURL     string `json:"html_url"`
	PublishedAt string `json:"published_at"`
	Prerelease  bool   `json:"prerelease"`
	Draft       bool   `json:"draft"`
}

type updateCheckResponse struct {
	Status          string `json:"status"`
	CurrentVersion  string `json:"current_version"`
	LatestVersion   string `json:"latest_version,omitempty"`
	ReleaseName     string `json:"release_name,omitempty"`
	ReleaseNotes    string `json:"release_notes,omitempty"`
	PublishedAt     string `json:"published_at,omitempty"`
	ReleaseURL      string `json:"release_url"`
	UpdateAvailable bool   `json:"update_available"`
	Message         string `json:"message"`
	CheckedAt       string `json:"checked_at"`
	Cached          bool   `json:"cached"`
}

func (s *Server) handleCheckGitHubUpdate(w http.ResponseWriter, r *http.Request) {
	result := s.checkGitHubUpdate(r.Context())
	writeJSON(w, http.StatusOK, result)
}

func (s *Server) checkGitHubUpdate(ctx context.Context) updateCheckResponse {
	now := time.Now().UTC()
	s.updateCacheMu.Lock()
	if !s.updateCacheExpires.IsZero() && now.Before(s.updateCacheExpires) {
		cached := s.updateCache
		cached.Cached = true
		s.updateCacheMu.Unlock()
		return cached
	}
	s.updateCacheMu.Unlock()

	result := updateCheckResponse{
		CurrentVersion: s.buildVersion,
		ReleaseURL:     GitHubReleasesPageURL,
		CheckedAt:      now.Format(time.RFC3339),
		Cached:         false,
	}

	client := s.githubHTTPClient
	if client == nil {
		client = &http.Client{Timeout: 5 * time.Second}
	}
	endpoint := s.githubReleaseURL
	if endpoint == "" {
		endpoint = GitHubLatestReleaseURL
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		result.Status = "unavailable"
		result.Message = "无法创建 GitHub 更新检查请求。"
		return result
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2026-03-10")
	req.Header.Set("User-Agent", "ZHIYU-WAF-V2-Update-Check")

	resp, err := client.Do(req)
	if err != nil {
		result.Status = "unavailable"
		result.Message = "GitHub Releases 暂时不可访问，请稍后重试。"
		return result
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		result.Status = "no_release"
		result.Message = "GitHub 尚未发布正式版本；当前部署将继续使用现有构建。"
		s.cacheUpdateResult(result, now)
		return result
	}
	if resp.StatusCode != http.StatusOK {
		result.Status = "unavailable"
		result.Message = fmt.Sprintf("GitHub Releases 查询暂不可用（HTTP %d），请稍后重试。", resp.StatusCode)
		return result
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		result.Status = "unavailable"
		result.Message = "无法读取 GitHub Releases 响应。"
		return result
	}
	var release githubRelease
	if err := json.Unmarshal(body, &release); err != nil {
		result.Status = "unavailable"
		result.Message = "GitHub Releases 返回了无法识别的数据。"
		return result
	}
	if release.TagName == "" || release.Draft || release.Prerelease {
		result.Status = "no_release"
		result.Message = "GitHub 暂无可用于稳定更新的正式版本。"
		s.cacheUpdateResult(result, now)
		return result
	}

	current, currentOK := parseSemanticVersion(s.buildVersion)
	latest, latestOK := parseSemanticVersion(release.TagName)
	if !currentOK || !latestOK {
		result.Status = "unavailable"
		result.LatestVersion = release.TagName
		result.ReleaseName = release.Name
		result.ReleaseURL = nonEmpty(release.HTMLURL, GitHubReleasesPageURL)
		result.Message = "版本标签不是有效的语义版本，无法安全比较。"
		return result
	}

	result.LatestVersion = release.TagName
	result.ReleaseName = release.Name
	result.ReleaseNotes = truncateReleaseNotes(release.Body)
	result.PublishedAt = release.PublishedAt
	result.ReleaseURL = nonEmpty(release.HTMLURL, GitHubReleasesPageURL)
	result.UpdateAvailable = compareSemanticVersion(latest, current) > 0
	if result.UpdateAvailable {
		result.Status = "update_available"
		result.Message = "发现新的 GitHub 正式发布版本；请查看发行说明并按受控流程升级。"
	} else {
		result.Status = "up_to_date"
		result.Message = "当前构建已是最新的 GitHub 正式版本。"
	}
	s.cacheUpdateResult(result, now)
	return result
}

func (s *Server) cacheUpdateResult(result updateCheckResponse, now time.Time) {
	s.updateCacheMu.Lock()
	defer s.updateCacheMu.Unlock()
	s.updateCache = result
	s.updateCacheExpires = now.Add(updateCheckCacheTTL)
}

func nonEmpty(value, fallback string) string {
	if value != "" {
		return value
	}
	return fallback
}

func truncateReleaseNotes(value string) string {
	const limit = 4000
	value = strings.TrimSpace(value)
	if len(value) <= limit {
		return value
	}
	return value[:limit] + "\n\n…"
}

type semanticVersion struct {
	major      int
	minor      int
	patch      int
	prerelease string
}

func parseSemanticVersion(value string) (semanticVersion, bool) {
	value = strings.TrimSpace(value)
	value = strings.TrimPrefix(value, "v")
	value = strings.TrimPrefix(value, "V")
	if before, _, ok := strings.Cut(value, "+"); ok {
		value = before
	}
	core, prerelease, hasPrerelease := strings.Cut(value, "-")
	if hasPrerelease && prerelease == "" {
		return semanticVersion{}, false
	}
	parts := strings.Split(core, ".")
	if len(parts) != 3 {
		return semanticVersion{}, false
	}
	values := [3]int{}
	for i, part := range parts {
		if part == "" || (len(part) > 1 && part[0] == '0') {
			return semanticVersion{}, false
		}
		n, err := strconv.Atoi(part)
		if err != nil || n < 0 {
			return semanticVersion{}, false
		}
		values[i] = n
	}
	return semanticVersion{major: values[0], minor: values[1], patch: values[2], prerelease: prerelease}, true
}

func compareSemanticVersion(left, right semanticVersion) int {
	for _, pair := range [][2]int{{left.major, right.major}, {left.minor, right.minor}, {left.patch, right.patch}} {
		if pair[0] > pair[1] {
			return 1
		}
		if pair[0] < pair[1] {
			return -1
		}
	}
	return comparePrerelease(left.prerelease, right.prerelease)
}

func comparePrerelease(left, right string) int {
	if left == right {
		return 0
	}
	if left == "" {
		return 1
	}
	if right == "" {
		return -1
	}
	leftParts, rightParts := strings.Split(left, "."), strings.Split(right, ".")
	limit := len(leftParts)
	if len(rightParts) < limit {
		limit = len(rightParts)
	}
	for i := 0; i < limit; i++ {
		lNum, lErr := strconv.Atoi(leftParts[i])
		rNum, rErr := strconv.Atoi(rightParts[i])
		if lErr == nil && rErr == nil {
			if lNum > rNum {
				return 1
			}
			if lNum < rNum {
				return -1
			}
			continue
		}
		if lErr == nil && rErr != nil {
			return -1
		}
		if lErr != nil && rErr == nil {
			return 1
		}
		if leftParts[i] > rightParts[i] {
			return 1
		}
		if leftParts[i] < rightParts[i] {
			return -1
		}
	}
	if len(leftParts) > len(rightParts) {
		return 1
	}
	return -1
}
