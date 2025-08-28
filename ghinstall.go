package ghinstall

import (
	"context"

	"github.com/sixban6/ghinstall/internal/config"
	"github.com/sixban6/ghinstall/internal/installer"
)

// Install provides a one-click entry point: loads configuration from the specified
// file path and completes the full installation process.
func Install(ctx context.Context, cfgPath string) error {
	cfg, err := config.Load(cfgPath)
	if err != nil {
		return err
	}
	return installer.New(nil, nil, nil).Install(ctx, cfg)
}

// InstallWithConfig installs using a pre-loaded configuration.
func InstallWithConfig(ctx context.Context, cfg *Config) error {
	return installer.New(nil, nil, nil).Install(ctx, cfg)
}

// Config exports the internal config structure for library usage.
type Config = config.Config

// Repo exports the internal repo structure for library usage.
type Repo = config.Repo

// LoadConfig loads and validates a configuration file.
func LoadConfig(cfgPath string) (*Config, error) {
	return config.Load(cfgPath)
}

// ParseRepoURL parses a GitHub repository URL into owner and repository name.
func ParseRepoURL(repoURL string) (owner, repo string, err error) {
	return config.ParseRepoURL(repoURL)
}