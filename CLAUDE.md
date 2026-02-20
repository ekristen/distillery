# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Distillery (`dist`) is a Go CLI tool for installing pre-compiled binaries from multiple sources: GitHub, GitLab, Homebrew, HashiCorp, Helm, Kubernetes, and HTTP. It handles asset discovery, checksum verification, signature verification (cosign/GPG), and local inventory management.

## Build, Test, and Lint

```bash
# Build
go build -o dist main.go

# Test (matches CI)
go test -timeout 60s ./...

# Run a single test
go test -timeout 60s -run TestFunctionName ./pkg/path/to/package/...

# Lint (matches CI)
golangci-lint run

# Docs (requires Docker)
make docs-serve
```

## Architecture

### Command Registration Pattern

Commands use a plugin-style `init()` registration pattern. Each command package registers itself via `common.RegisterCommand()` in its `init()` function, and `main.go` imports them as blank imports (`_ "github.com/ekristen/distillery/pkg/commands/install"`). New commands must follow this pattern.

### Provider/Source System

The core abstraction is `provider.ISource` (`pkg/provider/interface.go`) which all source implementations satisfy. The `Provider` base struct (`pkg/provider/provider.go`) contains shared logic orchestrated by `CommonRun()`: Download → Verify → Extract → Install → Cleanup.

Source implementations live in `pkg/source/` (e.g., `github.go`, `gitlab.go`, `hashicorp.go`). Each source has a corresponding `*_asset.go` file implementing the `asset.IAsset` interface for that source's download artifacts.

### Asset Scoring

`pkg/score/` ranks release assets by relevance to the target OS/architecture using weighted scoring. Assets scoring below 40 are rejected. The `pkg/osconfig/` package provides OS/arch detection and platform-specific aliases.

### Inventory and Installation

`pkg/inventory/` tracks installed binaries on disk at `$HOME/.distillery/opt/source/owner/repo/version/`. Symlinks in `$HOME/.distillery/bin/` provide PATH access. Multiple versions of the same binary are supported via `binary@version` naming.

### Configuration

`pkg/config/` supports YAML/TOML config files. Aliases (`pkg/config/aliases.go`) map shorthand names to full source paths. Distfiles (`pkg/distfile/`) define batch installation sets.

## Key Conventions

- **CLI framework**: `urfave/cli/v3`
- **Logging**: `rs/zerolog` with structured fields; use `p.Logger` in provider code
- **Testing**: `stretchr/testify` for assertions; table-driven tests preferred
- **Error wrapping**: `fmt.Errorf("context: %w", err)`
- **Lint rules**: max function length 100 lines (relaxed in tests), max line length 140 chars, cyclomatic complexity limit 15, duplication threshold 100 lines
- **Auth tokens**: `DISTILLERY_GITHUB_TOKEN` / `DISTILLERY_GITLAB_TOKEN` env vars
