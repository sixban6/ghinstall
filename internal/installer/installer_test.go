package installer

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/sixban6/ghinstall/internal/config"
	"github.com/sixban6/ghinstall/internal/downloader"
	"github.com/sixban6/ghinstall/internal/extractor"
	"github.com/sixban6/ghinstall/internal/release"
)

type mockFinder struct {
	release *release.Release
	err     error
}

func (m *mockFinder) LatestStable(ctx context.Context, owner, repo string) (*release.Release, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.release, nil
}

type mockDownloader struct {
	content string
	err     error
}

func (m *mockDownloader) Download(ctx context.Context, url string) (io.ReadCloser, error) {
	if m.err != nil {
		return nil, m.err
	}
	return io.NopCloser(strings.NewReader(m.content)), nil
}

type mockExtractor struct {
	extractedTo string
	err         error
}

func (m *mockExtractor) Extract(src io.Reader, dst string) error {
	if m.err != nil {
		return m.err
	}
	m.extractedTo = dst
	return nil
}

func TestNew(t *testing.T) {
	tests := []struct {
		name        string
		finder      release.Finder
		downloader  downloader.Client
		extractor   extractor.Extractor
		expectNil   bool
	}{
		{
			name:        "with all components",
			finder:      &mockFinder{},
			downloader:  &mockDownloader{},
			extractor:   &mockExtractor{},
			expectNil:   false,
		},
		{
			name:        "with nil components",
			finder:      nil,
			downloader:  nil,
			extractor:   nil,
			expectNil:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			installer := New(tt.finder, tt.downloader, tt.extractor)
			if installer == nil && !tt.expectNil {
				t.Error("New() returned nil, expected non-nil")
			}
			if installer != nil && tt.expectNil {
				t.Error("New() returned non-nil, expected nil")
			}
		})
	}
}

func TestInstaller_Install(t *testing.T) {
	mockRel := &release.Release{
		TagName: "v1.0.0",
		Assets: []release.Asset{
			{
				Name: "app.tar.gz",
				URL:  "https://github.com/owner/repo/releases/download/v1.0.0/app.tar.gz",
				Size: 1024,
			},
		},
	}

	tests := []struct {
		name           string
		config         *config.Config
		finder         release.Finder
		downloader     downloader.Client
		extractor      extractor.Extractor
		wantErr        bool
		expectedExtDir string
	}{
		{
			name: "successful install",
			config: &config.Config{
				Github: []config.Repo{
					{
						URL:       "https://github.com/owner/repo",
						OutputDir: "/tmp/test",
					},
				},
			},
			finder:         &mockFinder{release: mockRel},
			downloader:     &mockDownloader{content: "test content"},
			extractor:      &mockExtractor{},
			wantErr:        false,
			expectedExtDir: "/tmp/test",
		},
		{
			name: "finder error",
			config: &config.Config{
				Github: []config.Repo{
					{
						URL:       "https://github.com/owner/repo",
						OutputDir: "/tmp/test",
					},
				},
			},
			finder:     &mockFinder{err: errors.New("finder error")},
			downloader: &mockDownloader{content: "test content"},
			extractor:  &mockExtractor{},
			wantErr:    true,
		},
		{
			name: "no suitable asset",
			config: &config.Config{
				Github: []config.Repo{
					{
						URL:       "https://github.com/owner/repo",
						OutputDir: "/tmp/test",
					},
				},
			},
			finder: &mockFinder{
				release: &release.Release{
					TagName: "v1.0.0",
					Assets:  []release.Asset{},
				},
			},
			downloader: &mockDownloader{content: "test content"},
			extractor:  &mockExtractor{},
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			installer := New(tt.finder, tt.downloader, tt.extractor)
			filter := release.DefaultFilter()
			err := installer.Install(context.Background(), tt.config, filter)

			if (err != nil) != tt.wantErr {
				t.Errorf("Installer.Install() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && tt.expectedExtDir != "" {
				if mockExt, ok := tt.extractor.(*mockExtractor); ok {
					if mockExt.extractedTo != tt.expectedExtDir {
						t.Errorf("Expected extraction to %s, got %s", tt.expectedExtDir, mockExt.extractedTo)
					}
				}
			}
		})
	}
}

func TestInstaller_InstallRepo(t *testing.T) {
	mockRel := &release.Release{
		TagName: "v1.0.0",
		Assets: []release.Asset{
			{
				Name: "app.tar.gz",
				URL:  "https://github.com/owner/repo/releases/download/v1.0.0/app.tar.gz",
				Size: 1024,
			},
		},
	}

	cfg := &config.Config{
		MirrorURL: "https://ghfast.top",
	}

	mockExt := &mockExtractor{}
	installer := New(
		&mockFinder{release: mockRel},
		&mockDownloader{content: "test content"},
		mockExt,
	)

	err := installer.InstallRepo(
		context.Background(),
		cfg,
		"https://github.com/owner/repo",
		"/tmp/single",
		release.DefaultFilter(),
	)

	if err != nil {
		t.Errorf("Installer.InstallRepo() error = %v", err)
	}

	if mockExt.extractedTo != "/tmp/single" {
		t.Errorf("Expected extraction to /tmp/single, got %s", mockExt.extractedTo)
	}
}

func TestInstaller_Install_WithMirror(t *testing.T) {
	mockRel := &release.Release{
		TagName: "v1.0.0",
		Assets: []release.Asset{
			{
				Name: "app.tar.gz",
				URL:  "https://github.com/owner/repo/releases/download/v1.0.0/app.tar.gz",
				Size: 1024,
			},
		},
	}

	cfg := &config.Config{
		Github: []config.Repo{
			{
				URL:       "https://github.com/owner/repo",
				OutputDir: "/tmp/test",
			},
		},
		MirrorURL: "https://ghfast.top",
	}

	mockDown := &mockDownloader{content: "test content"}
	installer := New(
		&mockFinder{release: mockRel},
		mockDown,
		&mockExtractor{},
	)

	err := installer.Install(context.Background(), cfg, release.DefaultFilter())
	if err != nil {
		t.Errorf("Installer.Install() with mirror error = %v", err)
	}
}

func TestInstaller_Install_MultipleRepos(t *testing.T) {
	mockRel := &release.Release{
		TagName: "v1.0.0",
		Assets: []release.Asset{
			{
				Name: "app.tar.gz",
				URL:  "https://github.com/owner/repo/releases/download/v1.0.0/app.tar.gz",
				Size: 1024,
			},
		},
	}

	cfg := &config.Config{
		Github: []config.Repo{
			{
				URL:       "https://github.com/owner1/repo1",
				OutputDir: "/tmp/repo1",
			},
			{
				URL:       "https://github.com/owner2/repo2",
				OutputDir: "/tmp/repo2",
			},
		},
	}

	extractorCalls := make([]string, 0)
	mockExt := &mockExtractorWithCallback{
		callback: func(dst string) {
			extractorCalls = append(extractorCalls, dst)
		},
	}

	installer := New(
		&mockFinder{release: mockRel},
		&mockDownloader{content: "test content"},
		mockExt,
	)

	err := installer.Install(context.Background(), cfg, release.DefaultFilter())
	if err != nil {
		t.Errorf("Installer.Install() with multiple repos error = %v", err)
	}

	expectedCalls := []string{"/tmp/repo1", "/tmp/repo2"}
	if len(extractorCalls) != len(expectedCalls) {
		t.Errorf("Expected %d extractor calls, got %d", len(expectedCalls), len(extractorCalls))
		return
	}

	for i, expected := range expectedCalls {
		if extractorCalls[i] != expected {
			t.Errorf("Expected extractor call %d to be %s, got %s", i, expected, extractorCalls[i])
		}
	}
}

type mockExtractorWithCallback struct {
	callback func(string)
	err      error
}

func (m *mockExtractorWithCallback) Extract(src io.Reader, dst string) error {
	if m.err != nil {
		return m.err
	}
	if m.callback != nil {
		m.callback(dst)
	}
	return nil
}

func TestInstaller_Install_ContextCancellation(t *testing.T) {
	slowFinder := &slowMockFinder{
		delay: 100 * time.Millisecond,
	}

	cfg := &config.Config{
		Github: []config.Repo{
			{
				URL:       "https://github.com/owner/repo",
				OutputDir: "/tmp/test",
			},
		},
	}

	installer := New(slowFinder, &mockDownloader{}, &mockExtractor{})

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	err := installer.Install(ctx, cfg, release.DefaultFilter())
	if err == nil {
		t.Error("Expected context cancellation error, got nil")
	}
}

type slowMockFinder struct {
	delay time.Duration
}

func (s *slowMockFinder) LatestStable(ctx context.Context, owner, repo string) (*release.Release, error) {
	select {
	case <-time.After(s.delay):
		return &release.Release{
			TagName: "v1.0.0",
			Assets: []release.Asset{
				{Name: "app.tar.gz", URL: "test-url"},
			},
		}, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}