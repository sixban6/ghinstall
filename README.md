# GHInstall

## Installation

```bash
go get github.com/sixban6/ghinstall
```

## Usage

### As a Library

```go
package main

import (
    "context"
    "log"
    
    "github.com/sixban6/ghinstall"
)

func main() {
    // Simple usage with config file
    err := ghinstall.Install(context.Background(), "config.yaml")
    if err != nil {
        log.Fatal(err)
    }
    
    // Or use programmatically
    cfg := &ghinstall.Config{
        Github: []ghinstall.Repo{
            {
                URL:       "https://github.com/sixban6/singgen",
                OutputDir: "/root",
            },
        },
        MirrorURL: "https://ghfast.top",
    }
    
    err = ghinstall.InstallWithConfig(context.Background(), cfg)
    if err != nil {
        log.Fatal(err)
    }
}
```

### Configuration File

Create a `config.yaml` file:

```yaml
github:  
  - url: "https://github.com/sixban6/singgen"
    output_dir: "/root"
mirror_url: "https://ghfast.top"  # Optional GitHub mirror for acceleration
```

### Command Line Tool

Build the CLI tool:

```bash
go build -o ghinstall -ldflags="-s -w" ./cmd/ghinstall
```

Use it:

```bash
./ghinstall-cli config.yaml
```

## Architecture

The project follows clean architecture principles with clear separation of concerns:

```
ghinstall/
├── ghinstall.go              # Public API
├── internal/
│   ├── config/               # Configuration parsing
│   ├── release/              # GitHub API client
│   ├── downloader/           # HTTP download client
│   ├── extractor/            # Archive extraction
│   └── installer/            # Main coordinator
├── test/                     # Integration tests
└── .github/workflows/        # CI/CD
```

## License
[MIT License](LICENSE)