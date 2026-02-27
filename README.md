# Kyper CLI

[![CI](https://github.com/bitfootco/kyper-cli/actions/workflows/ci.yml/badge.svg)](https://github.com/bitfootco/kyper-cli/actions/workflows/ci.yml)

The official command-line tool for developers to push, validate, and manage apps on the [Kyper](https://kyper.shop) marketplace.

> Full documentation is also available at [kyper.shop/docs/developers/cli](https://kyper.shop/docs/developers/cli)

## Features

- **Device auth login** — browser-based authentication flow (no passwords in the terminal)
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

Verify the installation:

```bash
kyper version
```

## Quick Start

```bash
# 1. Authenticate (opens browser)
kyper login

# 2. Set up your project (interactive wizard)
kyper init

# 3. Validate your config
kyper validate

# 4. Deploy
kyper push
```

That's it. `kyper push` validates your `kyper.yml`, archives your source, uploads it to Kyper, and streams the build log in real-time. When the build completes, your app enters review.

## Commands

### Global Flags

Every command accepts these flags:

| Flag | Description |
|------|-------------|
| `--json` | Output raw JSON instead of styled text. Useful for scripting and CI pipelines. |
| `--host <url>` | Override the API host URL (default: `https://kyper.shop`) |
| `--version` | Print CLI version |

---

### Authentication

#### `kyper login`

Authenticate via the device authorization flow. Opens your browser to a verification page where you confirm the login, then the CLI receives an API token.

```bash
kyper login

# Open this URL in your browser to authenticate:
#   https://kyper.shop/device?code=A1B2C3D4
#
# ✓ Logged in as dev@example.com (developer)
```

The token is saved to `~/.kyper/config.yml` (permissions `0600`). Subsequent commands use this token automatically.

#### `kyper whoami`

Show the currently authenticated user.

```bash
kyper whoami
# dev@example.com (developer)
```

```bash
kyper whoami --json
# {"email":"dev@example.com","role":"developer"}
```

---

### Project Setup

#### `kyper init`

Interactive wizard that generates `kyper.yml` and `.kyperignore` for your project.

The wizard:
1. **Auto-detects** your stack (Rails, Django, Laravel, Go, Next.js, Express, and more)
2. **Auto-detects** processes (from `Procfile`, `Dockerfile`, framework conventions)
3. **Auto-detects** dependencies (PostgreSQL, Redis, MySQL, S3, etc. from config files and lockfiles)
4. **Suggests** deploy hooks based on your stack (e.g., `bundle exec rails db:migrate` for Rails)
5. **Suggests** health check paths (e.g., `/up` for Rails, `/health/` for Django)
6. Walks you through **pricing**, **category**, and **resource tier** selection
7. **Previews** the generated YAML and asks for confirmation

```bash
kyper init

# Auto-detected:
#   Stack: Rails 8.1 (from Gemfile.lock)
#   Process: web → bin/rails server (from Procfile)
#   Dep: postgres (from config/database.yml)
#   Dep: redis (from Gemfile)
#
# ...interactive prompts...
#
# ✓ Created kyper.yml
# ✓ Created .kyperignore with sensible defaults
```

If `kyper.yml` already exists, the wizard asks before overwriting.

> **Note:** `kyper init` requires interactive mode. The `--json` flag is not supported.

#### `kyper validate`

Validate `kyper.yml` locally without uploading anything. Catches errors (missing required fields, invalid values) and warnings (e.g., database dependency without a deploy hook).

```bash
kyper validate

# Validating kyper.yml
#
# ✓ All checks passed
```

With errors:

```bash
kyper validate

# Validating kyper.yml
#
#   FAIL  processes must include a 'web' key
#   WARN  deps includes 'postgres' but no on_deploy hook is set
#
# 1 error(s), 1 warning(s)
```

```bash
kyper validate --json
# {"valid":false,"errors":["processes must include a 'web' key"],"warnings":["deps includes 'postgres' but no on_deploy hook is set"]}
```

---

### Publishing

#### `kyper tag`

Bump the version in `kyper.yml`. Interactive by default — shows a selector for patch, minor, or major. Use `--bump` for non-interactive usage.

```bash
# Interactive
kyper tag
# Select version bump:
#   patch  1.2.3 → 1.2.4
#   minor  1.2.3 → 1.3.0
#   major  1.2.3 → 2.0.0

# Non-interactive
kyper tag --bump patch
# ✓ Version bumped 1.2.3 → 1.2.4
```

| Flag | Description |
|------|-------------|
| `--bump <type>` | Version bump type: `patch`, `minor`, or `major`. Required with `--json`. |

```bash
kyper tag --bump minor --json
# {"previous_version":"1.2.3","new_version":"1.3.0"}
```

#### `kyper push`

The main deployment command. Runs the full push workflow:

1. **Validates** `kyper.yml` locally
2. **Archives** your source code (respects `.kyperignore`)
3. **Syncs** app metadata (creates or updates the app listing)
4. **Uploads** the version (source zip + kyper.yml)
5. **Streams** the build log in real-time
6. On **build failure**, prompts to retry interactively

```bash
kyper push

# ✓ App synced
# ✓ Version 1.3.0 uploaded
#
# [build] Step 1/12 : FROM ruby:3.4-slim
# [build] Step 2/12 : WORKDIR /app
# ...
# [build] Successfully built a1b2c3d4
# [scan]  Trivy scan passed (0 critical, 2 low)
# [review] Build complete — submitted for review
```

In `--json` mode, the build log is suppressed and only the final result is printed:

```bash
kyper push --json
# {"id":42,"app":"invoice-hero","version":"1.3.0","status":"in_review"}
```

#### `kyper logs`

Stream build logs for the latest version. Useful if you disconnected during a `kyper push` or want to re-read the output.

```bash
kyper logs

# [build] Step 1/12 : FROM ruby:3.4-slim
# ...
```

#### `kyper status`

Show the current app status and latest version info.

```bash
kyper status

# App: Invoice Hero
# Slug: invoice-hero
# Status: active
#
# Latest Version
#   Version: 1.3.0
#   Status:  in_review
```

```bash
kyper status --json
# {"status":"active","latest_version":{"id":42,"version":"1.3.0","status":"in_review","review_notes":""}}
```

#### `kyper retry`

Retry a failed build. Only works when the latest version is in `build_failed` status.

```bash
kyper retry
# Build retry queued — streaming log...
# [build] Step 1/12 : FROM ruby:3.4-slim
# ...
```

#### `kyper cancel`

Cancel a version that is `pending` or `building`.

```bash
kyper cancel
# ✓ Build cancelled
```

#### `kyper withdraw`

Withdraw a version from review. Works on versions in `pending`, `build_failed`, `in_review`, or `rejected` status. Cannot withdraw published or currently building versions.

```bash
kyper withdraw
# ✓ Version 1.3.0 withdrawn
```

---

### Utility

#### `kyper version`

Print the CLI version, commit hash, and build date.

```bash
kyper version
# kyper 0.3.1 (abc1234, 2026-02-20)
```

---

## kyper.yml Reference

Project configuration lives in `kyper.yml` at the root of your repo. Run `kyper init` to generate it interactively.

```yaml
name: Invoice Hero
version: 1.0.0
description: A simple invoicing app for freelancers
tagline: Create and send invoices in seconds
category: productivity

docker:
  dockerfile: ./Dockerfile

processes:
  web: bin/rails server -p $PORT
  worker: bundle exec sidekiq

deps:
  - postgres: "16"
    storage_gb: 30
  - redis: "7"

env:
  - OPENAI_API_KEY
  - STRIPE_SECRET_KEY

hooks:
  on_deploy: bundle exec rails db:migrate
  on_update: bundle exec rails db:migrate

healthcheck:
  path: /up
  interval: 30
  timeout: 10

pricing:
  one_time: 49
  subscription: 12

resources:
  min_memory_mb: 1024
  min_cpu: 1
```

For the full field-by-field reference, see the [kyper.yml Reference](https://kyper.shop/docs/developers/kyper-yml) in the web docs.

### Key fields

| Field | Required | Description |
|-------|----------|-------------|
| `name` | Yes | Display name (slug auto-derived) |
| `version` | Yes | Semver string (e.g., `1.0.0`) |
| `description` | Yes | What your app does |
| `category` | Yes | One of: `developer_tools`, `productivity`, `finance`, `health`, `media`, `education`, `business_operations`, `data_analytics`, `gaming` |
| `docker.dockerfile` | Yes | Path to Dockerfile (relative to project root) |
| `processes.web` | Yes | Command to start the web server |
| `deps` | No | Infrastructure dependencies (`postgres`, `mysql`, `redis`, `elasticsearch`, `opensearch`, `s3`) |
| `env` | No | Required environment variable names (consumers must set these before deploy) |
| `hooks.on_deploy` | No | Run after first deployment (e.g., migrations) |
| `hooks.on_update` | No | Run after updates (e.g., migrations) |
| `pricing.one_time` | No* | One-time purchase price in USD |
| `pricing.subscription` | No* | Monthly subscription price in USD |

\* At least one pricing option is required.

---

## CI / Automation

Every command supports `--json` mode for machine-readable output. Combine with `--bump` on `kyper tag` for fully automated version bumping and deployment:

```bash
# CI pipeline example
kyper tag --bump patch --json
kyper push --json
```

Exit codes: `0` on success, `1` on any error.

## Configuration

Auth credentials are stored in `~/.kyper/config.yml`:

```yaml
api_token: kpr_a1b2c3d4e5f6...
```

This file is created by `kyper login` with `0600` permissions (owner read/write only).

## Tech Stack

Built with Go, [Cobra](https://github.com/spf13/cobra), [Huh](https://github.com/charmbracelet/huh) (interactive forms), [Lip Gloss](https://github.com/charmbracelet/lipgloss) (styling), and [Glamour](https://github.com/charmbracelet/glamour) (markdown rendering).

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
