package main

import (
	"context"
	"flag"
	"fmt"
	log "github.com/sixban6/ghinstall/internal/logger"
	"os"
	"time"

	"github.com/sixban6/ghinstall"
)

var appVersion = "dev"

func main() {
	var (
		configFile = flag.String("config", "", "Path to configuration file")
		timeout    = flag.Duration("timeout", 5*time.Minute, "Timeout for installation")
		verbose    = flag.Bool("verbose", true, "Enable verbose logging")
		version    = flag.Bool("version", false, "Show version information")
	)
	flag.Parse()

	if *version {
		log.Info("ghinstall version %s\n", appVersion)
		fmt.Println("A tool for automatically downloading GitHub releases")
		return
	}

	if *configFile == "" {
		if len(flag.Args()) > 0 {
			*configFile = flag.Args()[0]
		} else {
			log.Error("Usage: %s [flags] <config-file>\n", os.Args[0])
			log.Error("   or: %s -config <config-file>\n", os.Args[0])
			flag.PrintDefaults()
			os.Exit(1)
		}
	}

	if !*verbose {
		log.SetOutput(os.Stderr)
		log.SetFlags(0)
	}

	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()

	log.Info("Loading configuration from %s", *configFile)
	cfg, err := ghinstall.LoadConfig(*configFile)
	if err != nil {
		log.Error("Failed to load configuration: %v", err)
	}

	log.Info("Found %d repositories to install", len(cfg.Github))

	if cfg.MirrorURL != "" {
		log.Info("Using GitHub mirror: %s", cfg.MirrorURL)
	}

	log.Info("Starting installation...")

	start := time.Now()
	if err := ghinstall.InstallWithConfig(ctx, cfg); err != nil {
		log.Error("Installation failed: %v", err)
	}

	duration := time.Since(start)
	log.Info("Installation completed successfully in %v", duration)
}
