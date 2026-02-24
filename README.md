# Kyper CLI

The official command-line tool for developers to push, validate, and manage apps on the [Kyper](https://kyper.shop) marketplace.

## Features

- **Device auth login** — browser-based authentication flow
- **Interactive project setup** — guided wizard with auto-detection of your stack, processes, and dependencies
- **Local validation** — catch `kyper.yml` issues before pushing
- **One-command deploy** — validate, archive, upload, and stream build logs with `kyper push`
- **Build management** — stream logs, retry failed builds, cancel or withdraw versions
- **Scriptable** — every command supports `--json` for CI/automation

## Install

```bash
brew tap kyper-shop/tap
brew install kyper
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

```bash
go build -o kyper ./cmd/kyper
go test ./...
```

## License

Proprietary. All rights reserved.
