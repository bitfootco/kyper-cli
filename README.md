# Kyper CLI

[![CI](https://github.com/bitfootco/kyper-cli/actions/workflows/ci.yml/badge.svg)](https://github.com/bitfootco/kyper-cli/actions/workflows/ci.yml)

The official command-line tool for developers to push, validate, and manage apps on the [Kyper](https://kyper.shop) marketplace.

## Features

- **Device auth login** — browser-based authentication flow
- **Interactive project setup** — guided wizard with auto-detection of your stack, processes, and dependencies
- **Local validation** — catch `kyper.yml` issues before pushing
- **One-command deploy** — validate, archive, upload, and stream build logs with `kyper push`
- **Build management** — stream logs, retry failed builds, cancel or withdraw versions
- **Scriptable** — every command supports `--json` for CI/automation

## Install

### Homebrew (macOS / Linux)

```bash
brew tap bitfootco/tap
brew install kyper
```

### Binary download

Grab a prebuilt binary from the [Releases](https://github.com/bitfootco/kyper-cli/releases) page. Archives are available for macOS and Linux on both amd64 and arm64.

```bash
# Example: macOS ARM
tar -xzf kyper_darwin_arm64.tar.gz
sudo mv kyper /usr/local/bin/
```

## Quick Start

```bash
# Authenticate
kyper login

# Set up your project (interactive wizard)
kyper init

# Validate your config
kyper validate

# Deploy
kyper push
```

## Commands

| Command | Description |
|---|---|
| `kyper login` | Authenticate via browser |
| `kyper init` | Interactive project setup wizard |
| `kyper validate` | Validate `kyper.yml` locally |
| `kyper push` | Validate, archive, upload, and tail build log |
| `kyper status` | Show app and latest version status |
| `kyper logs` | Stream build logs for the latest version |
| `kyper retry` | Retry a failed build |
| `kyper cancel` | Cancel a pending/building version |
| `kyper withdraw` | Withdraw a version from review |
| `kyper whoami` | Show authenticated user |
| `kyper version` | Print CLI version |

## Configuration

Project configuration lives in `kyper.yml` at the root of your repo:

```yaml
name: my-app
version: 1.0.0
description: What this app does
category: productivity

docker:
  dockerfile: ./Dockerfile

processes:
  web: bin/rails server

deps:
  - postgres
  - "redis:7"

pricing:
  subscription: 9.99
```

Run `kyper init` to generate this interactively, or `kyper validate` to check an existing file.

## Tech Stack

Built with Go, [Cobra](https://github.com/spf13/cobra), [Huh](https://github.com/charmbracelet/huh), and [Lip Gloss](https://github.com/charmbracelet/lipgloss).

## Development

Requires **Go 1.23+**.

```bash
# Build
go build -o kyper ./cmd/kyper

# Test (CI runs with -race)
go test -race ./...

# Lint (must match CI)
golangci-lint run
```

### Releasing

Releases are automated via [GoReleaser](https://goreleaser.com). Push a semver tag to trigger a build:

```bash
git tag v0.1.0
git push origin v0.1.0
```

This runs the test suite, builds binaries for all platforms, publishes a GitHub release, and updates the Homebrew tap.

## License

Proprietary. All rights reserved.
