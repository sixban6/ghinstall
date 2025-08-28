package release

import (
	"context"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
	"time"
)

func TestGitHubClient_LatestStable(t *testing.T) {
	tests := []struct {
		name           string
		serverResponse string
		serverStatus   int
		wantTagName    string
		wantErr        bool
	}{
		{
			name: "successful response with stable releases",
			serverResponse: `[
				{
					"tag_name": "v2.0.0",
					"name": "Release 2.0.0",
					"draft": false,
					"prerelease": false,
					"published_at": "2023-12-01T00:00:00Z",
					"assets": [
						{
							"name": "app-v2.0.0.tar.gz",
							"browser_download_url": "https://github.com/owner/repo/releases/download/v2.0.0/app-v2.0.0.tar.gz",
							"content_type": "application/gzip",
							"size": 1024
						}
					]
				},
				{
					"tag_name": "v1.0.0",
					"name": "Release 1.0.0",
					"draft": false,
					"prerelease": false,
					"published_at": "2023-11-01T00:00:00Z",
					"assets": [
						{
							"name": "app-v1.0.0.tar.gz",
							"browser_download_url": "https://github.com/owner/repo/releases/download/v1.0.0/app-v1.0.0.tar.gz",
							"content_type": "application/gzip",
							"size": 512
						}
					]
				}
			]`,
			serverStatus: http.StatusOK,
			wantTagName:  "v2.0.0",
			wantErr:      false,
		},
		{
			name: "filter out prerelease",
			serverResponse: `[
				{
					"tag_name": "v2.0.0-beta1",
					"name": "Release 2.0.0 Beta 1",
					"draft": false,
					"prerelease": true,
					"published_at": "2023-12-01T00:00:00Z",
					"assets": []
				},
				{
					"tag_name": "v1.0.0",
					"name": "Release 1.0.0",
					"draft": false,
					"prerelease": false,
					"published_at": "2023-11-01T00:00:00Z",
					"assets": []
				}
			]`,
			serverStatus: http.StatusOK,
			wantTagName:  "v1.0.0",
			wantErr:      false,
		},
		{
			name: "filter out draft",
			serverResponse: `[
				{
					"tag_name": "v2.0.0",
					"name": "Release 2.0.0",
					"draft": true,
					"prerelease": false,
					"published_at": "2023-12-01T00:00:00Z",
					"assets": []
				},
				{
					"tag_name": "v1.0.0",
					"name": "Release 1.0.0",
					"draft": false,
					"prerelease": false,
					"published_at": "2023-11-01T00:00:00Z",
					"assets": []
				}
			]`,
			serverStatus: http.StatusOK,
			wantTagName:  "v1.0.0",
			wantErr:      false,
		},
		{
			name:           "empty releases",
			serverResponse: `[]`,
			serverStatus:   http.StatusOK,
			wantTagName:    "",
			wantErr:        true,
		},
		{
			name: "no stable releases",
			serverResponse: `[
				{
					"tag_name": "v1.0.0-alpha",
					"name": "Alpha Release",
					"draft": false,
					"prerelease": true,
					"published_at": "2023-11-01T00:00:00Z",
					"assets": []
				}
			]`,
			serverStatus: http.StatusOK,
			wantTagName:  "",
			wantErr:      true,
		},
		{
			name:           "server error",
			serverResponse: `{"message": "Not Found"}`,
			serverStatus:   http.StatusNotFound,
			wantTagName:    "",
			wantErr:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tt.serverStatus)
				w.Write([]byte(tt.serverResponse))
			}))
			defer server.Close()

			client := &GitHubClient{
				httpClient: &http.Client{Timeout: 5 * time.Second},
				baseURL:    server.URL,
			}

			got, err := client.LatestStable(context.Background(), "owner", "repo")
			if (err != nil) != tt.wantErr {
				t.Errorf("GitHubClient.LatestStable() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && got.TagName != tt.wantTagName {
				t.Errorf("GitHubClient.LatestStable() = %v, want TagName %v", got.TagName, tt.wantTagName)
			}
		})
	}
}

func TestFindLatestRelease(t *testing.T) {
	releases := []Release{
		{
			TagName:     "v1.0.0",
			PublishedAt: time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
		},
		{
			TagName:     "v2.0.0",
			PublishedAt: time.Date(2023, 2, 1, 0, 0, 0, 0, time.UTC),
		},
		{
			TagName:     "v1.5.0",
			PublishedAt: time.Date(2023, 1, 15, 0, 0, 0, 0, time.UTC),
		},
	}

	latest := findLatestRelease(releases)
	if latest.TagName != "v2.0.0" {
		t.Errorf("findLatestRelease() = %v, want v2.0.0", latest.TagName)
	}
}

func TestNormalizeTag(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"v1.0.0", "v1.0.0"},
		{"1.0.0", "v1.0.0"},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := normalizeTag(tt.input)
			if got != tt.want {
				t.Errorf("normalizeTag(%v) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestRelease_FindAsset(t *testing.T) {
	release := &Release{
		Assets: []Asset{
			{Name: "app-linux-amd64.tar.gz", URL: "url1"},
			{Name: "app-windows-amd64.zip", URL: "url2"},
			{Name: "app-darwin-amd64.tar.gz", URL: "url3"},
			{Name: "source-code.zip", URL: "url4"},
		},
	}

	tests := []struct {
		name     string
		patterns []string
		wantName string
		wantURL  string
	}{
		{
			name:     "find tar.gz",
			patterns: []string{".tar.gz"},
			wantName: "app-linux-amd64.tar.gz",
			wantURL:  "url1",
		},
		{
			name:     "find zip",
			patterns: []string{".zip"},
			wantName: "app-windows-amd64.zip",
			wantURL:  "url2",
		},
		{
			name:     "default patterns",
			patterns: nil,
			wantName: "app-linux-amd64.tar.gz",
			wantURL:  "url1",
		},
		{
			name:     "specific pattern",
			patterns: []string{"linux"},
			wantName: "app-linux-amd64.tar.gz",
			wantURL:  "url1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := release.FindAsset(tt.patterns...)
			if got == nil {
				t.Errorf("Release.FindAsset() = nil, want asset")
				return
			}
			if got.Name != tt.wantName {
				t.Errorf("Release.FindAsset() Name = %v, want %v", got.Name, tt.wantName)
			}
			if got.URL != tt.wantURL {
				t.Errorf("Release.FindAsset() URL = %v, want %v", got.URL, tt.wantURL)
			}
		})
	}
}

func TestRelease_FindAsset_NotFound(t *testing.T) {
	release := &Release{
		Assets: []Asset{
			{Name: "README.md", URL: "url1"},
		},
	}

	got := release.FindAsset(".tar.gz", ".zip")
	if got != nil {
		t.Errorf("Release.FindAsset() = %v, want nil", got)
	}
}

func TestFilterStableReleases(t *testing.T) {
	releases := []Release{
		{TagName: "v1.0.0", Draft: false, Prerelease: false},
		{TagName: "v2.0.0-beta", Draft: false, Prerelease: true},
		{TagName: "v1.5.0", Draft: true, Prerelease: false},
		{TagName: "v0.9.0", Draft: false, Prerelease: false},
	}

	stable := filterStableReleases(releases)
	want := []Release{
		{TagName: "v1.0.0", Draft: false, Prerelease: false},
		{TagName: "v0.9.0", Draft: false, Prerelease: false},
	}

	if !reflect.DeepEqual(stable, want) {
		t.Errorf("filterStableReleases() = %v, want %v", stable, want)
	}
}