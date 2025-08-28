package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Github    []Repo `yaml:"github"`
	MirrorURL string `yaml:"mirror_url"`
}

type Repo struct {
	URL       string `yaml:"url"`
	OutputDir string `yaml:"output_dir"`
}

func Load(cfgPath string) (*Config, error) {
	data, err := os.ReadFile(cfgPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file %q: %w", cfgPath, err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file %q: %w", cfgPath, err)
	}

	if err := cfg.validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	cfg.normalize()
	return &cfg, nil
}

func (c *Config) validate() error {
	if len(c.Github) == 0 {
		return fmt.Errorf("no GitHub repositories configured")
	}

	for i, repo := range c.Github {
		if repo.URL == "" {
			return fmt.Errorf("repository at index %d: URL is required", i)
		}
		if repo.OutputDir == "" {
			return fmt.Errorf("repository at index %d: output_dir is required", i)
		}
		if !strings.HasPrefix(repo.URL, "https://github.com/") {
			return fmt.Errorf("repository at index %d: URL must be a GitHub repository URL", i)
		}
	}

	return nil
}

func (c *Config) normalize() {
	for i := range c.Github {
		c.Github[i].OutputDir = filepath.Clean(c.Github[i].OutputDir)
		c.Github[i].URL = strings.TrimSuffix(c.Github[i].URL, "/")
	}

	if c.MirrorURL != "" {
		c.MirrorURL = strings.TrimSuffix(c.MirrorURL, "/")
	}
}

func (c *Config) GetDownloadURL(repoURL, assetURL string) string {
	if c.MirrorURL == "" {
		return assetURL
	}
	return c.MirrorURL + "/" + assetURL
}

func ParseRepoURL(repoURL string) (owner, repo string, err error) {
	if !strings.HasPrefix(repoURL, "https://github.com/") {
		return "", "", fmt.Errorf("invalid GitHub URL: %s", repoURL)
	}

	path := strings.TrimPrefix(repoURL, "https://github.com/")
	parts := strings.Split(path, "/")
	if len(parts) < 2 {
		return "", "", fmt.Errorf("invalid GitHub URL format: %s", repoURL)
	}

	return parts[0], parts[1], nil
}