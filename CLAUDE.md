
CLAUDE.md — Kyper CLI

This is the Kyper CLI — a Go binary for developers to push, validate, and manage apps on the Kyper marketplace. It replaces a previous Ruby gem implementation.

Related Repos

- Platform (Rails): ~/Code/kyper — the main Kyper marketplace app (API, web UI, admin, deployment infrastructure)
- This CLI repo is a pure API client. All business logic lives in the platform.

Tech Stack

┌────────────────────┬────────────────────────┐
│       Layer        │         Choice         │
├────────────────────┼────────────────────────┤
│ Language           │ Go 1.23+               │
├────────────────────┼────────────────────────┤
│ CLI framework      │ Cobra                  │
├────────────────────┼────────────────────────┤
│ TUI / forms        │ Huh (Bubble Tea-based) │
├────────────────────┼────────────────────────┤
│ Styling            │ Lip Gloss              │
├────────────────────┼────────────────────────┤
│ Markdown rendering │ Glamour                │
├────────────────────┼────────────────────────┤
│ YAML               │ gopkg.in/yaml.v3       │
├────────────────────┼────────────────────────┤
│ HTTP               │ net/http (stdlib)      │
├────────────────────┼────────────────────────┤
│ Release            │ GoReleaser             │
├────────────────────┼────────────────────────┤
│ Distribution       │ Homebrew tap           │
└────────────────────┴────────────────────────┘

Project Structure

cmd/kyper/main.go              # Entrypoint — calls cmd.Execute()
internal/
  cmd/                          # One file per Cobra command
    root.go                     # Root command, persistent --json and --host flags
    login.go                    # Device auth flow
    init.go                     # Interactive project setup wizard
    validate.go                 # Local kyper.yml validation
    push.go                     # Validate + zip + upload + tail build log
    logs.go                     # Cursor-based build log streaming
    retry.go                    # Retry failed build
    status.go                   # App + version status display
    cancel.go                   # Cancel pending/building version
    withdraw.go                 # Withdraw version from review
    whoami.go                   # Show authenticated user
    version_cmd.go              # CLI version display
    helpers.go                  # Shared: requireAuth(), loadKyperYML(), tailLog()
  api/                          # HTTP client layer
    client.go                   # All API methods
    errors.go                   # APIError type, IsNotFound/IsUnauthorized helpers
    transport.go                # Custom RoundTripper: auth header, user-agent, retry
  config/                       # Local state management
    config.go                   # ~/.kyper/config.yml read/write (0600 perms)
    kyperfile.go                # kyper.yml Go structs + YAML parsing
  kyperfile/                    # Validation logic (pure functions, no I/O)
    validate.go                 # All rules + constants (CATEGORIES, KNOWN_DEPS, etc.)
  detect/                       # Auto-detection for init wizard
    stack.go                    # Framework detection (Rails, Django, Express, Go, etc.)
    processes.go                # Process detection from Procfile/Dockerfile/package.json
    deps.go                     # Dep detection from 5 sources
    depversions.go              # Version suggestions from lockfiles
  archive/                      # Zip creation
    archive.go                  # Builds upload zip, respects .kyperignore
  ui/                           # Shared presentation utilities
    styles.go                   # Lip Gloss style constants (success, error, warning, etc.)
    spinner.go                  # RunWithSpinner(label, fn) wrapper
    output.go                   # PrintSuccess/Error/Warning/Table/JSON helpers
  version/
    version.go                  # Version/Commit/Date vars (set by ldflags)
.goreleaser.yml
.github/workflows/
  ci.yml                        # go test + go vet + golangci-lint
  release.yml                   # GoReleaser on tag push

Conventions

- Standard Go project layout — cmd/ for entrypoints, internal/ for private packages
- Cobra commands — one file per command in internal/cmd/, registered in root.go
- No global state — pass config/client through command context or closures
- Errors return, don't panic — use fmt.Errorf("context: %w", err) wrapping
- --json flag on every command — when set, output raw JSON, skip interactive elements. Enables scripting.
- Test with httptest — mock API responses with httptest.NewServer, never call real API in tests
- Lip Gloss styles — all colors/formatting defined in ui/styles.go, never hardcoded in commands

Design Goals

Build a CLI experience on par with fly and terraform:
- Beautiful colored output with consistent styling
- Interactive prompts with arrow-key selection (Huh forms)
- Real-time log streaming with proper formatting
- Spinners for async operations
- Clear error messages with actionable suggestions
- Fast startup (no interpreter overhead)

---
Commands (10 total, all top-level — no subcommands)

kyper login       — Authenticate via browser (device auth flow)
kyper init        — Interactive project setup wizard
kyper validate    — Validate kyper.yml locally
kyper push        — Validate + archive + upload + tail build log
kyper status      — Show app and latest version status
kyper logs        — Stream build logs for latest version
kyper retry       — Retry a failed build
kyper cancel      — Cancel a pending/building version
kyper withdraw    — Withdraw a version from review
kyper whoami      — Show authenticated user
kyper version     — Print CLI version

---
API Surface

The CLI is a pure HTTP client. All endpoints are under https://kyper.shop/api/v1/.
Base URL is configurable via KYPER_HOST env var.

Authentication

All endpoints except device auth require Authorization: Bearer <api_token> header.
Token is stored in ~/.kyper/config.yml.

Endpoints

Device Auth (unauthenticated — used by kyper login)

┌────────┬────────────────────────────────┬────────────────────────────────────────────────────────────────────────────────────┐
│ Method │              Path              │                                      Response                                      │
├────────┼────────────────────────────────┼────────────────────────────────────────────────────────────────────────────────────┤
│ POST   │ /api/v1/device/authorize       │ {code: "UUID", verification_uri: "https://kyper.shop/device?code=UUID"}            │
├────────┼────────────────────────────────┼────────────────────────────────────────────────────────────────────────────────────┤
│ GET    │ /api/v1/device/token?code=UUID │ {api_token: "..."} when authorized, {pending: true} when waiting, 404 when expired │
└────────┴────────────────────────────────┴────────────────────────────────────────────────────────────────────────────────────┘

Device grants expire after 5 minutes. CLI polls every 2 seconds.

User

┌────────┬────────────┬───────────────────┐
│ Method │    Path    │     Response      │
├────────┼────────────┼───────────────────┤
│ GET    │ /api/v1/me │ {id, email, role} │
└────────┴────────────┴───────────────────┘

Apps

┌────────┬───────────────────────────┬──────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────┬────────────────────────────────────────────────────────────────────┐
│ Method │           Path            │                                                           Body                                                           │                              Response                              │
├────────┼───────────────────────────┼──────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────┼────────────────────────────────────────────────────────────────────┤
│ GET    │ /api/v1/apps              │ —                                                                                                                        │ [{slug, title, tagline, status, pricing_type, ...}]                │
├────────┼───────────────────────────┼──────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────┼────────────────────────────────────────────────────────────────────┤
│ POST   │ /api/v1/apps              │ {app: {title, tagline, description, category, pricing_type, one_time_price_cents, subscription_price_cents, tech_stack}} │ {slug, ...} (201) or {errors: [...]} (422)                         │
├────────┼───────────────────────────┼──────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────┼────────────────────────────────────────────────────────────────────┤
│ GET    │ /api/v1/apps/:slug        │ —                                                                                                                        │ Single app object                                                  │
├────────┼───────────────────────────┼──────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────┼────────────────────────────────────────────────────────────────────┤
│ PATCH  │ /api/v1/apps/:slug        │ {app: {tagline, description, category, ...}}                                                                             │ Updated app (note: title is immutable)                             │
├────────┼───────────────────────────┼──────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────┼────────────────────────────────────────────────────────────────────┤
│ GET    │ /api/v1/apps/:slug/status │ —                                                                                                                        │ {app, status, latest_version: {id, version, status, review_notes}} │
└────────┴───────────────────────────┴──────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────┴────────────────────────────────────────────────────────────────────┘

Versions (Build Pipeline)

┌────────┬─────────────────────────────────────────┬───────────────────────────────────────────────────┬────────────────────────────────────────────┐
│ Method │                  Path                   │                       Body                        │                  Response                  │
├────────┼─────────────────────────────────────────┼───────────────────────────────────────────────────┼────────────────────────────────────────────┤
│ POST   │ /api/v1/apps/:slug/versions             │ multipart: kyper_yml (string) + source_zip (file) │ {id, app, version, status, message} (201)  │
├────────┼─────────────────────────────────────────┼───────────────────────────────────────────────────┼────────────────────────────────────────────┤
│ GET    │ /api/v1/versions/:id/build_log?cursor=N │ —                                                 │ {status, log, cursor, complete}            │
├────────┼─────────────────────────────────────────┼───────────────────────────────────────────────────┼────────────────────────────────────────────┤
│ POST   │ /api/v1/versions/:id/retry              │ —                                                 │ {id, message} (only if build_failed)       │
├────────┼─────────────────────────────────────────┼───────────────────────────────────────────────────┼────────────────────────────────────────────┤
│ POST   │ /api/v1/versions/:id/cancel             │ —                                                 │ {id, message} (only if pending/building)   │
├────────┼─────────────────────────────────────────┼───────────────────────────────────────────────────┼────────────────────────────────────────────┤
│ DELETE │ /api/v1/versions/:id                    │ —                                                 │ {message} (only if not published/building) │
└────────┴─────────────────────────────────────────┴───────────────────────────────────────────────────┴────────────────────────────────────────────┘

Error Response Formats

{"error": "Single error message"}
{"errors": ["Validation error 1", "Validation error 2"]}

Status codes: 200/201 success, 400 bad input, 401 unauthorized, 404 not found, 409 conflict, 422 validation errors.

---
Config Files

~/.kyper/config.yml (user state, 0600 permissions)

api_token: <urlsafe_base64_token>

kyper.yml (project config, committed to repo)

Full schema:

name: my-app                    # Required. Lowercase, alphanum + hyphens, ≤100 chars. Becomes the slug.
version: 1.0.0                  # Required. Semver MAJOR.MINOR.PATCH.
description: What this app does # Required. Max 500 chars.
tagline: Short pitch            # Optional. Max 160 chars. Falls back to first 160 of description.
category: productivity          # Required. One of: developer_tools, productivity, finance, health, media, education, business_operations, data_analytics, gaming

docker:
  dockerfile: ./Dockerfile      # Required. Path to Dockerfile. docker.image is REJECTED (Kyper builds from source).

processes:
  web: bin/rails server          # Required. Must include `web`. Value is the start command.
  worker: bundle exec sidekiq    # Optional additional processes.

deps:                            # Optional. Array of known dependencies.
  - postgres                     # String format (latest allowed version)
  - "redis:7"                    # Version-pinned format
  - postgres: "16"              # Hash format with version
    storage_gb: 50               # Optional storage override (default 20GB)

pricing:
  one_time: 29.99               # USD, ≥$1 if set. At least one pricing option required.
  subscription: 9.99            # USD/month, ≥$1 if set.

resources:
  min_memory_mb: 512            # Default 512. Affects tier selection.
  min_cpu: 1                    # Default 1.

env:                             # Optional. Array of required env var key names.
  - DATABASE_URL
  - API_KEY

hooks:
  on_deploy: bundle exec rails db:migrate    # Optional. Runs after first deployment.
  on_update: bundle exec rails db:migrate    # Optional. Runs on version update.

healthcheck:
  path: /up                      # Optional. Default /up. Must start with /.
  interval: 30                   # Optional. Seconds, 10-300. Default 30.
  timeout: 10                    # Optional. Positive integer. Default 10.

Known deps: postgres, mysql, redis, elasticsearch, opensearch
Allowed dep versions: postgres (14, 15, 16), mysql (8), redis (6, 7), elasticsearch (8), opensearch (2)

Auto-injected env vars (consumers cannot override): DATABASE_URL, REDIS_URL, SECRET_KEY_BASE, PORT, KYPER_DEPLOYMENT_ID, ELASTICSEARCH_URL, OPENSEARCH_URL

---
Commands — Detailed Behavior

kyper login

1. Call POST /api/v1/device/authorize (unauthenticated) → get {code, verification_uri}
2. Print verification URL to terminal
3. Open browser: open (macOS), xdg-open (Linux)
4. Poll GET /api/v1/device/token?code=X every 2 seconds, 5-minute timeout
5. On success: save token to ~/.kyper/config.yml, call GET /api/v1/me to verify, print identity
6. Spinner during code request + polling phases

kyper init

Interactive wizard using Huh forms. Flow:
1. App basics — name (default: CWD basename), category (9-option select), tagline (max 160), description
2. Auto-detect — scan project files, display detected stack/processes/deps with source labels
3. Processes — confirm detected or enter manually. web is required.
4. Dependencies — multi-select from detected + manual add. Suggest versions from lockfiles.
5. Hooks — if DB dep present, prompt for deploy hook. Suggest based on stack:
  - Rails: bundle exec rails db:migrate
  - Django: python manage.py migrate
  - Prisma: npx prisma migrate deploy
  - Laravel: php artisan migrate --force
6. Health check — path input with stack default (Rails→/up, Django→/health/, Node→/health)
7. Pricing — one-time and/or subscription price
8. Resources — tier select (512MB/$6, 1024MB/$12, 2048MB/$18, 4096MB/$24)
9. Preview — render YAML with Glamour, confirm before writing

Stack detection reads: config/application.rb (Rails), manage.py (Django), artisan (Laravel), go.mod (Go), package.json (Node frameworks), schema.prisma (Prisma)

Dep detection reads: docker-compose.yml, Gemfile, package.json, requirements.txt, Pipfile. Maps:
- Gemfile: pg→postgres, mysql2→mysql, redis→redis, elasticsearch→elasticsearch, opensearch-ruby→opensearch
- package.json: pg/prisma→postgres, mysql2→mysql, redis/ioredis→redis, @elastic/elasticsearch→elasticsearch, @opensearch-project/opensearch→opensearch
- requirements.txt/Pipfile: psycopg2/psycopg→postgres, mysqlclient/PyMySQL→mysql, redis→redis, elasticsearch→elasticsearch, opensearch-py→opensearch

Version suggestions from lockfiles: read Gemfile.lock or package-lock.json for specific version hints.

kyper validate

Reads kyper.yml from CWD. Runs all checks, prints styled pass/fail per item. Exit 0 on pass, 1 on fail.

Validation rules:
- name: present, ≤100 chars
- version: present, matches ^\d+\.\d+\.\d+$
- category: present, in CATEGORIES list
- docker.dockerfile: present, no docker.image, referenced file exists
- processes: is a map, has web key
- deps: each is known dep, version in allow-list if pinned, storage_gb 1-500 if set
- hooks: on_deploy/on_update must be strings if present
- healthcheck.path: starts with / if present
- healthcheck.interval: 10-300 if present
- healthcheck.timeout: positive integer if present
- Warning (not error): DB dep present without hooks.on_deploy

--json mode: {"valid": bool, "errors": [...], "warnings": [...]}

kyper push

1. Require auth
2. Read + validate kyper.yml (fail fast)
3. Build zip archive (spinner: "Building archive..."):
  - archive/zip stdlib
  - Default excludes: .git/, *.log, tmp/, node_modules/
  - Read .kyperignore for additional patterns
  - Print archive size on success
4. Sync app to API:
  - GET /api/v1/apps/{slug}/status — if 404, create via POST; else update via PATCH
  - Extract metadata from kyper.yml: title (from name), tagline, description, category, pricing_type, prices, tech_stack
5. Upload version (spinner: "Uploading..."):
  - POST /api/v1/apps/{slug}/versions — multipart: kyper_yml (raw YAML string) + source_zip (file)
6. Tail build log (auto-starts after upload)
7. On build_failed: prompt "Retry?" (Huh confirm)

Slug derivation: strings.ToLower(name), replace [^a-z0-9]+ with -, trim leading/trailing -

Pricing type derivation: both prices → "both", only one_time → "one_time", only subscription → "subscription"

kyper logs

Read kyper.yml → get slug → GET /api/v1/apps/{slug}/status → get latest_version.id → tail build log.

tailLog(versionID, startCursor): poll GET /api/v1/versions/{id}/build_log?cursor=N every 2 seconds. Print incremental log content. Break when complete == true. 30-minute timeout. Print colored status banner at end.

kyper retry

Fetch status → verify latest_version.status == "build_failed" → POST /api/v1/versions/{id}/retry → tail log from beginning.

kyper status

Read kyper.yml → GET /api/v1/apps/{slug}/status → display app name, app status, latest version, review status, review notes.

kyper cancel

Fetch status → verify latest version is pending/building → POST /api/v1/versions/{id}/cancel.

kyper withdraw

Fetch status → verify version is not published/building → Huh confirm (default: no) → DELETE /api/v1/versions/{id}.

kyper whoami

Load token → GET /api/v1/me → print {email} ({role}).

kyper version

Print kyper {Version} ({Commit}, {Date}). Values injected via ldflags.

---
Testing

- internal/api/ — httptest.NewServer to mock all API responses
- internal/config/ — temp directories for config read/write
- internal/kyperfile/ — table-driven tests for every validation rule
- internal/detect/ — fixture directories with sample project files
- internal/archive/ — temp directories with .kyperignore patterns
- Integration: run compiled binary as subprocess, verify kyper version output
