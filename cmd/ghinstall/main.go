package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/sixban6/ghinstall"
)

var appVersion = "dev"

func main() {
	var (
		configFile = flag.String("config", "", "Path to configuration file")
		timeout    = flag.Duration("timeout", 5*time.Minute, "Timeout for installation")
		verbose    = flag.Bool("verbose", false, "Enable verbose logging")
		version    = flag.Bool("version", false, "Show version information")
	)
	flag.Parse()

	if *version {
		fmt.Printf("ghinstall version %s\n", appVersion)
		fmt.Println("A tool for automatically downloading GitHub releases")
		return
	}

	if *configFile == "" {
		if len(flag.Args()) > 0 {
			*configFile = flag.Args()[0]
		} else {
			fmt.Fprintf(os.Stderr, "Usage: %s [flags] <config-file>\n", os.Args[0])
			fmt.Fprintf(os.Stderr, "   or: %s -config <config-file>\n", os.Args[0])
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

	log.Printf("Loading configuration from %s", *configFile)
	cfg, err := ghinstall.LoadConfig(*configFile)
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	log.Printf("Found %d repositories to install", len(cfg.Github))
	
	if cfg.MirrorURL != "" {
		log.Printf("Using GitHub mirror: %s", cfg.MirrorURL)
	}

	log.Printf("Starting installation...")
	
	start := time.Now()
	if err := ghinstall.InstallWithConfig(ctx, cfg); err != nil {
		log.Fatalf("Installation failed: %v", err)
	}
	
	duration := time.Since(start)
	log.Printf("Installation completed successfully in %v", duration)
}