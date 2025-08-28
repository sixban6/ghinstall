# GHInstall

## Installation

```bash
go get github.com/sixban6/ghinstall
```

## Usage

### As a Library

```go
  // 使用默认过滤器（保持向后兼容）
  err := ghinstall.InstallWithConfig(ctx, cfg)

  // 使用自定义过滤器
  err := ghinstall.InstallWithConfigAndFilter(ctx, cfg, filterFunc)

  使用示例:

  // 1. 按操作系统和架构过滤
  filter := ghinstall.CombinedFilter(
      ghinstall.ByOS("linux"),
      ghinstall.ByArch("amd64"),
  )
  err := ghinstall.InstallWithConfigAndFilter(ctx, cfg, filter)

  // 2. 按文件名模式过滤
  filter := ghinstall.ByNamePattern("sing-box", "linux", "amd64")
  err := ghinstall.InstallWithConfigAndFilter(ctx, cfg, filter)

  // 3. 自定义过滤逻辑
  filter := ghinstall.CustomFilter(func(assets []ghinstall.Asset) (*ghinstall.Asset, error) {
      for _, asset := range assets {
          if strings.Contains(asset.Name, "linux-amd64") &&
             strings.HasSuffix(asset.Name, ".tar.gz") {
              return &asset, nil
          }
      }
      return nil, fmt.Errorf("找不到 linux-amd64 tar.gz 文件")
  })

  预定义过滤器:
  - DefaultAssetFilter() - 原始行为
  - ByNamePattern(patterns...) - 按文件名模式
  - ByOS(os) - 按操作系统
  - ByArch(arch) - 按架构
  - BySize(largest) - 按文件大小
  - CombinedFilter(filters...) - 组合多个过滤器
  - CustomFilter(func) - 完全自定义
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