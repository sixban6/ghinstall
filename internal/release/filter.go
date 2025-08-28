package release

import (
	"fmt"
	"runtime"
	"sort"
	"strings"
)

// AssetFilter defines a function that selects one asset from available assets
type AssetFilter func(assets []Asset) (*Asset, error)

// DefaultFilter returns the original FindAsset behavior
func DefaultFilter() AssetFilter {
	return func(assets []Asset) (*Asset, error) {
		patterns := []string{".tar.gz", ".zip", ".tgz"}
		
		for _, asset := range assets {
			name := strings.ToLower(asset.Name)
			for _, pattern := range patterns {
				if strings.Contains(name, strings.ToLower(pattern)) {
					return &asset, nil
				}
			}
		}
		
		return nil, fmt.Errorf("no suitable asset found")
	}
}

// ByNamePattern creates a filter that matches asset names by patterns
func ByNamePattern(patterns ...string) AssetFilter {
	return func(assets []Asset) (*Asset, error) {
		if len(patterns) == 0 {
			return nil, fmt.Errorf("no patterns specified")
		}
		
		for _, asset := range assets {
			name := strings.ToLower(asset.Name)
			for _, pattern := range patterns {
				if strings.Contains(name, strings.ToLower(pattern)) {
					return &asset, nil
				}
			}
		}
		
		return nil, fmt.Errorf("no asset matching patterns %v", patterns)
	}
}

// ByOS creates a filter that matches assets by operating system
func ByOS(os string) AssetFilter {
	return func(assets []Asset) (*Asset, error) {
		targetOS := strings.ToLower(os)
		if targetOS == "" {
			targetOS = strings.ToLower(runtime.GOOS)
		}
		
		// OS mapping
		osAliases := map[string][]string{
			"linux":   {"linux"},
			"darwin":  {"darwin", "macos", "osx"},
			"windows": {"windows", "win"},
		}
		
		aliases := osAliases[targetOS]
		if aliases == nil {
			aliases = []string{targetOS}
		}
		
		for _, asset := range assets {
			name := strings.ToLower(asset.Name)
			for _, alias := range aliases {
				if strings.Contains(name, alias) {
					return &asset, nil
				}
			}
		}
		
		return nil, fmt.Errorf("no asset found for OS: %s", os)
	}
}

// ByArch creates a filter that matches assets by architecture
func ByArch(arch string) AssetFilter {
	return func(assets []Asset) (*Asset, error) {
		targetArch := strings.ToLower(arch)
		if targetArch == "" {
			targetArch = strings.ToLower(runtime.GOARCH)
		}
		
		// Architecture mapping
		archAliases := map[string][]string{
			"amd64": {"amd64", "x86_64", "x64"},
			"386":   {"386", "i386", "x86"},
			"arm64": {"arm64", "aarch64"},
			"arm":   {"arm", "armv7"},
		}
		
		aliases := archAliases[targetArch]
		if aliases == nil {
			aliases = []string{targetArch}
		}
		
		for _, asset := range assets {
			name := strings.ToLower(asset.Name)
			for _, alias := range aliases {
				if strings.Contains(name, alias) {
					return &asset, nil
				}
			}
		}
		
		return nil, fmt.Errorf("no asset found for architecture: %s", arch)
	}
}

// Combined creates a filter that applies multiple filters in sequence
func Combined(filters ...AssetFilter) AssetFilter {
	return func(assets []Asset) (*Asset, error) {
		if len(filters) == 0 {
			return nil, fmt.Errorf("no filters specified")
		}
		
		currentAssets := assets
		
		for i, filter := range filters {
			if len(currentAssets) == 0 {
				return nil, fmt.Errorf("no assets left after filter %d", i)
			}
			
			// For the last filter, select one asset
			if i == len(filters)-1 {
				return filter(currentAssets)
			}
			
			// For intermediate filters, we need to modify the logic
			// This is a simplified approach - in practice, you might want
			// to collect all matching assets and pass them to the next filter
			selected, err := filter(currentAssets)
			if err != nil {
				return nil, fmt.Errorf("filter %d failed: %w", i, err)
			}
			currentAssets = []Asset{*selected}
		}
		
		if len(currentAssets) == 0 {
			return nil, fmt.Errorf("no assets found after applying all filters")
		}
		
		return &currentAssets[0], nil
	}
}

// BySize creates a filter that selects the largest or smallest asset
func BySize(largest bool) AssetFilter {
	return func(assets []Asset) (*Asset, error) {
		if len(assets) == 0 {
			return nil, fmt.Errorf("no assets available")
		}
		
		sorted := make([]Asset, len(assets))
		copy(sorted, assets)
		
		sort.Slice(sorted, func(i, j int) bool {
			if largest {
				return sorted[i].Size > sorted[j].Size
			}
			return sorted[i].Size < sorted[j].Size
		})
		
		return &sorted[0], nil
	}
}

// Custom creates a filter from a user-defined function
func Custom(fn func([]Asset) (*Asset, error)) AssetFilter {
	return AssetFilter(fn)
}