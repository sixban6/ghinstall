package release

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"

	"golang.org/x/mod/semver"
)

type Asset struct {
	Name        string `json:"name"`
	URL         string `json:"browser_download_url"`
	ContentType string `json:"content_type"`
	Size        int64  `json:"size"`
}

type Release struct {
	TagName     string    `json:"tag_name"`
	Name        string    `json:"name"`
	Assets      []Asset   `json:"assets"`
	Draft       bool      `json:"draft"`
	Prerelease  bool      `json:"prerelease"`
	PublishedAt time.Time `json:"published_at"`
}

type Finder interface {
	LatestStable(ctx context.Context, owner, repo string) (*Release, error)
}

type GitHubClient struct {
	httpClient *http.Client
	baseURL    string
}

func NewGitHubClient() *GitHubClient {
	return &GitHubClient{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		baseURL: "https://api.github.com",
	}
}

func (c *GitHubClient) LatestStable(ctx context.Context, owner, repo string) (*Release, error) {
	url := fmt.Sprintf("%s/repos/%s/%s/releases", c.baseURL, owner, repo)
	
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.Header.Set("User-Agent", "ghinstall/1.0")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch releases: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API returned status %d", resp.StatusCode)
	}

	var releases []Release
	if err := json.NewDecoder(resp.Body).Decode(&releases); err != nil {
		return nil, fmt.Errorf("failed to decode releases: %w", err)
	}

	if len(releases) == 0 {
		return nil, fmt.Errorf("no releases found for %s/%s", owner, repo)
	}

	stableReleases := filterStableReleases(releases)
	if len(stableReleases) == 0 {
		return nil, fmt.Errorf("no stable releases found for %s/%s", owner, repo)
	}

	latest := findLatestRelease(stableReleases)
	return &latest, nil
}

func filterStableReleases(releases []Release) []Release {
	var stable []Release
	for _, release := range releases {
		if !release.Draft && !release.Prerelease {
			stable = append(stable, release)
		}
	}
	return stable
}

func findLatestRelease(releases []Release) Release {
	if len(releases) == 0 {
		panic("no releases to compare")
	}

	if len(releases) == 1 {
		return releases[0]
	}

	sort.Slice(releases, func(i, j int) bool {
		tagI := normalizeTag(releases[i].TagName)
		tagJ := normalizeTag(releases[j].TagName)
		
		if semver.IsValid(tagI) && semver.IsValid(tagJ) {
			return semver.Compare(tagI, tagJ) > 0
		}
		
		if semver.IsValid(tagI) && !semver.IsValid(tagJ) {
			return true
		}
		
		if !semver.IsValid(tagI) && semver.IsValid(tagJ) {
			return false
		}
		
		return releases[i].PublishedAt.After(releases[j].PublishedAt)
	})

	return releases[0]
}

func normalizeTag(tag string) string {
	if tag == "" {
		return ""
	}
	
	if strings.HasPrefix(tag, "v") {
		return tag
	}
	
	return "v" + tag
}

func (r *Release) FindAsset(patterns ...string) *Asset {
	if len(patterns) == 0 {
		patterns = []string{".tar.gz", ".zip", ".tgz"}
	}

	for _, asset := range r.Assets {
		name := strings.ToLower(asset.Name)
		for _, pattern := range patterns {
			if strings.Contains(name, strings.ToLower(pattern)) {
				return &asset
			}
		}
	}

	return nil
}

func (r *Release) String() string {
	return fmt.Sprintf("Release{TagName: %s, Assets: %d}", r.TagName, len(r.Assets))
}