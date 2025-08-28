package config

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestLoad(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    *Config
		wantErr bool
	}{
		{
			name: "valid config",
			content: `github:
  - url: "https://github.com/sixban6/singgen"
    output_dir: "/root"
mirror_url: "https://ghfast.top"`,
			want: &Config{
				Github: []Repo{
					{
						URL:       "https://github.com/sixban6/singgen",
						OutputDir: "/root",
					},
				},
				MirrorURL: "https://ghfast.top",
			},
			wantErr: false,
		},
		{
			name: "multiple repos",
			content: `github:
  - url: "https://github.com/owner1/repo1"
    output_dir: "/opt/repo1"
  - url: "https://github.com/owner2/repo2"
    output_dir: "/opt/repo2"
mirror_url: "https://ghfast.top"`,
			want: &Config{
				Github: []Repo{
					{
						URL:       "https://github.com/owner1/repo1",
						OutputDir: "/opt/repo1",
					},
					{
						URL:       "https://github.com/owner2/repo2",
						OutputDir: "/opt/repo2",
					},
				},
				MirrorURL: "https://ghfast.top",
			},
			wantErr: false,
		},
		{
			name: "no mirror url",
			content: `github:
  - url: "https://github.com/sixban6/singgen"
    output_dir: "/root"`,
			want: &Config{
				Github: []Repo{
					{
						URL:       "https://github.com/sixban6/singgen",
						OutputDir: "/root",
					},
				},
				MirrorURL: "",
			},
			wantErr: false,
		},
		{
			name: "empty github list",
			content: `github: []
mirror_url: "https://ghfast.top"`,
			want:    nil,
			wantErr: true,
		},
		{
			name: "invalid github url",
			content: `github:
  - url: "https://gitlab.com/owner/repo"
    output_dir: "/root"`,
			want:    nil,
			wantErr: true,
		},
		{
			name: "missing url",
			content: `github:
  - output_dir: "/root"`,
			want:    nil,
			wantErr: true,
		},
		{
			name: "missing output_dir",
			content: `github:
  - url: "https://github.com/sixban6/singgen"`,
			want:    nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpFile := createTempConfigFile(t, tt.content)
			defer os.Remove(tmpFile)

			got, err := Load(tmpFile)
			if (err != nil) != tt.wantErr {
				t.Errorf("Load() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Load() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestConfig_GetDownloadURL(t *testing.T) {
	tests := []struct {
		name      string
		config    *Config
		repoURL   string
		assetURL  string
		want      string
	}{
		{
			name: "with mirror",
			config: &Config{
				MirrorURL: "https://ghfast.top",
			},
			repoURL:  "https://github.com/owner/repo",
			assetURL: "https://github.com/owner/repo/releases/download/v1.0.0/app.tar.gz",
			want:     "https://ghfast.top/https://github.com/owner/repo/releases/download/v1.0.0/app.tar.gz",
		},
		{
			name: "without mirror",
			config: &Config{
				MirrorURL: "",
			},
			repoURL:  "https://github.com/owner/repo",
			assetURL: "https://github.com/owner/repo/releases/download/v1.0.0/app.tar.gz",
			want:     "https://github.com/owner/repo/releases/download/v1.0.0/app.tar.gz",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.config.GetDownloadURL(tt.repoURL, tt.assetURL)
			if got != tt.want {
				t.Errorf("Config.GetDownloadURL() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParseRepoURL(t *testing.T) {
	tests := []struct {
		name      string
		repoURL   string
		wantOwner string
		wantRepo  string
		wantErr   bool
	}{
		{
			name:      "valid url",
			repoURL:   "https://github.com/sixban6/singgen",
			wantOwner: "sixban6",
			wantRepo:  "singgen",
			wantErr:   false,
		},
		{
			name:      "url with trailing slash",
			repoURL:   "https://github.com/owner/repo/",
			wantOwner: "owner",
			wantRepo:  "repo",
			wantErr:   false,
		},
		{
			name:      "invalid url - not github",
			repoURL:   "https://gitlab.com/owner/repo",
			wantOwner: "",
			wantRepo:  "",
			wantErr:   true,
		},
		{
			name:      "invalid url - incomplete",
			repoURL:   "https://github.com/owner",
			wantOwner: "",
			wantRepo:  "",
			wantErr:   true,
		},
		{
			name:      "invalid url - empty",
			repoURL:   "",
			wantOwner: "",
			wantRepo:  "",
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotOwner, gotRepo, err := ParseRepoURL(tt.repoURL)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseRepoURL() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if gotOwner != tt.wantOwner {
				t.Errorf("ParseRepoURL() gotOwner = %v, want %v", gotOwner, tt.wantOwner)
			}
			if gotRepo != tt.wantRepo {
				t.Errorf("ParseRepoURL() gotRepo = %v, want %v", gotRepo, tt.wantRepo)
			}
		})
	}
}

func createTempConfigFile(t *testing.T, content string) string {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "config.yaml")
	
	if err := os.WriteFile(tmpFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create temp config file: %v", err)
	}
	
	return tmpFile
}