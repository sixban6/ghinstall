package ghinstall

import (
	"context"

	"github.com/sixban6/ghinstall/internal/config"
	"github.com/sixban6/ghinstall/internal/installer"
	"github.com/sixban6/ghinstall/internal/release"
)

// Install provides a one-click entry point: loads configuration from the specified
// file path and completes the full installation process with default asset filter.
func Install(ctx context.Context, cfgPath string) error {
	cfg, err := config.Load(cfgPath)
	if err != nil {
		return err
	}
	return installer.New(nil, nil, nil).Install(ctx, cfg, DefaultAssetFilter())
}

// InstallWithFilter provides installation with a custom asset filter.
func InstallWithFilter(ctx context.Context, cfgPath string, filter AssetFilter) error {
	cfg, err := config.Load(cfgPath)
	if err != nil {
		return err
	}
	return installer.New(nil, nil, nil).Install(ctx, cfg, filter)
}

// InstallWithConfig installs using a pre-loaded configuration with default asset filter.
func InstallWithConfig(ctx context.Context, cfg *Config) error {
	return installer.New(nil, nil, nil).Install(ctx, cfg, DefaultAssetFilter())
}

// InstallWithConfigAndFilter installs using a pre-loaded configuration and custom asset filter.
func InstallWithConfigAndFilter(ctx context.Context, cfg *Config, filter AssetFilter) error {
	return installer.New(nil, nil, nil).Install(ctx, cfg, filter)
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

// AssetFilter exports the asset filter function type for library usage.
type AssetFilter = release.AssetFilter

// Asset exports the asset structure for library usage.
type Asset = release.Asset

// DefaultAssetFilter returns the default asset filter (same as original behavior).
func DefaultAssetFilter() AssetFilter {
	return release.DefaultFilter()
}

// ByNamePattern creates a filter that matches asset names by patterns.
func ByNamePattern(patterns ...string) AssetFilter {
	return release.ByNamePattern(patterns...)
}

// ByOS creates a filter that matches assets by operating system.
func ByOS(os string) AssetFilter {
	return release.ByOS(os)
}

// ByArch creates a filter that matches assets by architecture.
func ByArch(arch string) AssetFilter {
	return release.ByArch(arch)
}

// CombinedFilter creates a filter that applies multiple filters in sequence.
func CombinedFilter(filters ...AssetFilter) AssetFilter {
	return release.Combined(filters...)
}

// BySize creates a filter that selects the largest or smallest asset.
func BySize(largest bool) AssetFilter {
	return release.BySize(largest)
}

// CustomFilter creates a filter from a user-defined function.
func CustomFilter(fn func([]Asset) (*Asset, error)) AssetFilter {
	return release.Custom(fn)
}