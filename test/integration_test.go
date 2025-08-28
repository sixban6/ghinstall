package test

import (
	"context"
	"log"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/sixban6/ghinstall"
)

func TestInstall_EndToEnd(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	tmpDir := t.TempDir()
	configPath := createTestConfig(t, tmpDir)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	err := ghinstall.Install(ctx, configPath)
	if err != nil {
		t.Errorf("Install() failed: %v", err)
		return
	}

	outputDir := filepath.Join(tmpDir, "output")
	if _, err := os.Stat(outputDir); os.IsNotExist(err) {
		t.Errorf("Output directory %s was not created", outputDir)
		return
	}

	files, err := os.ReadDir(outputDir)
	if err != nil {
		t.Errorf("Failed to read output directory: %v", err)
		return
	}

	if len(files) == 0 {
		t.Error("No files were extracted to the output directory")
	}

	t.Logf("Successfully extracted %d files/directories", len(files))
}

func TestLoadConfig(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := createTestConfig(t, tmpDir)

	cfg, err := ghinstall.LoadConfig(configPath)
	if err != nil {
		t.Errorf("LoadConfig() failed: %v", err)
		return
	}

	if len(cfg.Github) != 1 {
		t.Errorf("Expected 1 GitHub repository, got %d", len(cfg.Github))
	}

	expectedURL := "https://github.com/cli/cli"
	if cfg.Github[0].URL != expectedURL {
		t.Errorf("Expected URL %s, got %s", expectedURL, cfg.Github[0].URL)
	}
}

func TestInstallWithConfig(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	tmpDir := t.TempDir()
	outputDir := filepath.Join(tmpDir, "output")

	cfg := &ghinstall.Config{
		Github: []ghinstall.Repo{
			{
				URL:       "https://github.com/sixban6/singgen",
				OutputDir: outputDir,
			},
		},
		MirrorURL: "https://ghfast.top",
	}

	err := ghinstall.InstallWithConfig(context.Background(), cfg)
	if err != nil {
		log.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	err = ghinstall.InstallWithConfig(ctx, cfg)
	if err != nil {
		t.Errorf("InstallWithConfig() failed: %v", err)
		return
	}

	if _, err := os.Stat(outputDir); os.IsNotExist(err) {
		t.Errorf("Output directory %s was not created", outputDir)
	}
}

func TestParseRepoURL(t *testing.T) {
	tests := []struct {
		url       string
		wantOwner string
		wantRepo  string
		wantErr   bool
	}{
		{
			url:       "https://github.com/golang/example",
			wantOwner: "golang",
			wantRepo:  "example",
			wantErr:   false,
		},
		{
			url:       "https://github.com/owner/repo/",
			wantOwner: "owner",
			wantRepo:  "repo",
			wantErr:   false,
		},
		{
			url:     "https://gitlab.com/owner/repo",
			wantErr: true,
		},
		{
			url:     "invalid-url",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.url, func(t *testing.T) {
			owner, repo, err := ghinstall.ParseRepoURL(tt.url)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseRepoURL() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if owner != tt.wantOwner {
					t.Errorf("ParseRepoURL() owner = %v, want %v", owner, tt.wantOwner)
				}
				if repo != tt.wantRepo {
					t.Errorf("ParseRepoURL() repo = %v, want %v", repo, tt.wantRepo)
				}
			}
		})
	}
}

func TestInstall_InvalidConfig(t *testing.T) {
	tmpDir := t.TempDir()
	invalidConfigPath := filepath.Join(tmpDir, "invalid.yaml")

	invalidConfig := `invalid yaml content:
  - this is not valid`

	if err := os.WriteFile(invalidConfigPath, []byte(invalidConfig), 0644); err != nil {
		t.Fatalf("Failed to create invalid config file: %v", err)
	}

	err := ghinstall.Install(context.Background(), invalidConfigPath)
	if err == nil {
		t.Error("Install() should fail with invalid config")
	}
}

func TestInstall_NonexistentConfig(t *testing.T) {
	err := ghinstall.Install(context.Background(), "/nonexistent/config.yaml")
	if err == nil {
		t.Error("Install() should fail with nonexistent config file")
	}
}

func TestInstall_ContextCancellation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	tmpDir := t.TempDir()
	configPath := createTestConfig(t, tmpDir)

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	err := ghinstall.Install(ctx, configPath)
	if err == nil {
		t.Error("Install() should fail with context cancellation")
	}
}

func createTestConfig(t *testing.T, tmpDir string) string {
	outputDir := filepath.Join(tmpDir, "output")
	configContent := `github:
  - url: "https://github.com/cli/cli"
    output_dir: "` + outputDir + `"
mirror_url: ""
`

	configPath := filepath.Join(tmpDir, "config.yaml")
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("Failed to create test config file: %v", err)
	}

	return configPath
}

func BenchmarkInstall(b *testing.B) {
	if testing.Short() {
		b.Skip("Skipping benchmark in short mode")
	}

	tmpDir := b.TempDir()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		outputDir := filepath.Join(tmpDir, "bench-output", string(rune(i)))
		cfg := &ghinstall.Config{
			Github: []ghinstall.Repo{
				{
					URL:       "https://github.com/cli/cli",
					OutputDir: outputDir,
				},
			},
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		err := ghinstall.InstallWithConfig(ctx, cfg)
		cancel()

		if err != nil {
			b.Errorf("Benchmark iteration %d failed: %v", i, err)
			return
		}
	}
}
