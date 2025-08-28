package installer

import (
	"context"
	"fmt"
	"log"

	"github.com/sixban6/ghinstall/internal/config"
	"github.com/sixban6/ghinstall/internal/downloader"
	"github.com/sixban6/ghinstall/internal/extractor"
	"github.com/sixban6/ghinstall/internal/release"
)

type Installer struct {
	finder     release.Finder
	downloader downloader.Client
	extractor  extractor.Extractor
}

func New(f release.Finder, d downloader.Client, e extractor.Extractor) *Installer {
	if f == nil {
		f = release.NewGitHubClient()
	}
	if d == nil {
		d = downloader.NewHTTPClient()
	}
	if e == nil {
		e = extractor.New()
	}

	return &Installer{
		finder:     f,
		downloader: d,
		extractor:  e,
	}
}

func (i *Installer) Install(ctx context.Context, cfg *config.Config) error {
	for _, repo := range cfg.Github {
		if err := i.installRepo(ctx, cfg, repo); err != nil {
			return fmt.Errorf("failed to install %s: %w", repo.URL, err)
		}
	}
	return nil
}

func (i *Installer) installRepo(ctx context.Context, cfg *config.Config, repo config.Repo) error {
	log.Printf("Installing %s to %s", repo.URL, repo.OutputDir)

	owner, repoName, err := config.ParseRepoURL(repo.URL)
	if err != nil {
		return fmt.Errorf("failed to parse repository URL: %w", err)
	}

	log.Printf("Finding latest stable release for %s/%s", owner, repoName)
	rel, err := i.finder.LatestStable(ctx, owner, repoName)
	if err != nil {
		return fmt.Errorf("failed to find latest release: %w", err)
	}

	log.Printf("Found release: %s", rel.TagName)

	asset := rel.FindAsset()
	if asset == nil {
		return fmt.Errorf("no suitable asset found in release %s", rel.TagName)
	}

	log.Printf("Selected asset: %s (%.2f MB)", asset.Name, float64(asset.Size)/(1024*1024))

	downloadURL := cfg.GetDownloadURL(repo.URL, asset.URL)
	if downloadURL != asset.URL {
		log.Printf("Using mirror: %s", downloadURL)
	}

	log.Printf("Downloading %s", downloadURL)
	reader, err := i.downloader.Download(ctx, downloadURL)
	if err != nil {
		return fmt.Errorf("failed to download asset: %w", err)
	}
	defer reader.Close()

	log.Printf("Extracting to %s", repo.OutputDir)
	if err := i.extractor.Extract(reader, repo.OutputDir); err != nil {
		return fmt.Errorf("failed to extract archive: %w", err)
	}

	log.Printf("Successfully installed %s %s to %s", repo.URL, rel.TagName, repo.OutputDir)
	return nil
}

func (i *Installer) InstallRepo(ctx context.Context, cfg *config.Config, repoURL, outputDir string) error {
	repo := config.Repo{
		URL:       repoURL,
		OutputDir: outputDir,
	}
	return i.installRepo(ctx, cfg, repo)
}