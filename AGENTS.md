# KnowledgeHub — Agent Instructions

## What This Project Is

KnowledgeHub is a personal knowledge radar — a single-binary web app that monitors RSS feeds and blogs, summarizes content with AI via OpenRouter, learns user preferences through star ratings, and serves a mobile-friendly feed UI. It runs on an LXC container behind Tailscale (no public internet exposure).

## Project Structure

```
cmd/knowledgehub/
  main.go              # Entry point: PocketBase setup, embed UI, start scheduler
  collections.go       # DB schema: resources, entries, preferences, app_settings
  hooks.go             # PocketBase hooks: URL-change reset, cascade delete
  ui/build/            # Copied from ui/build at build time (gitignored)

internal/
  ai/
    client.go          # OpenRouter HTTP client (Complete + CompleteStream)
    summarizer.go      # Combined summarize + score in single LLM call
    preference.go      # Preference profile generation from correction history
    settings.go        # Helpers to read API key and model from app_settings
    chat.go            # (if exists) Chat-related AI helpers
  engine/
    scheduler.go       # 30-minute ticker, fetches all active resources
    fetcher.go         # Dispatches RSS vs watchlist, creates entries, triggers AI
    rss.go             # RSS feed parsing via gofeed, deduplication by GUID
    scraper.go         # HTML scraper for watchlist pages (finds article links)
    readability.go     # Article content extraction via go-readability
    quarantine.go      # State machine: healthy → failing → quarantined (5 failures)
    http.go            # Shared DefaultHTTPClient variable
  routes/
    chat.go            # POST /api/chat — streaming SSE chat with article context
  testutil/
    testutil.go        # Shared test helpers: NewTestApp, Create* functions

ui/                    # SvelteKit frontend (static adapter, built with Bun)
  src/lib/
    pb.ts              # PocketBase JS SDK client instance
    components/
      ChatPanel.svelte
      EntryCard.svelte
      Nav.svelte
      QuarantineBanner.svelte
      ResourceForm.svelte
      StarRating.svelte
  src/routes/
    +layout.svelte     # App shell with Nav
    +page.svelte       # Feed view (home page)
    resources/+page.svelte  # Resource CRUD management
    settings/+page.svelte   # OpenRouter API key and model config

openspec/              # Design documents and task tracking
knowledgehub.service   # systemd unit file
Makefile               # Build, dev, test, release targets
```

## Technology Stack

| Layer | Technology | Notes |
|-------|-----------|-------|
| Backend | Go 1.24+ | Single binary, no CGo optional (uses modernc SQLite) |
| Framework | PocketBase (embedded) | Provides auth, REST API, realtime SSE, SQLite |
| Frontend | SvelteKit 5 + Tailwind CSS | Static adapter, embedded in Go binary via `//go:embed` |
| Frontend build | Bun | Not npm/node — use `bun install` and `bun run build` |
| AI | OpenRouter API | Model-agnostic (Claude, GPT, Llama, etc.) |
| Database | SQLite | Managed by PocketBase, single file in `kh_data/` |
| RSS | gofeed | `github.com/mmcdole/gofeed` |
| Content extraction | go-readability | `github.com/go-shiori/go-readability` |
| HTML scraping | goquery | `github.com/PuerkitoBio/goquery` |

## Key Architecture Decisions

- **PocketBase is embedded in Go** — not run as a standalone server. We extend it with custom Go code (hooks, routes, scheduler). The PocketBase admin dashboard is available at `/_/`.
- **SvelteKit uses the static adapter** — no JS runtime needed at deployment. The built output is copied to `cmd/knowledgehub/ui/build/` and embedded via `//go:embed`.
- **Single LLM call per article** — summarize + score combined to minimize API costs.
- **Preference learning via prompt engineering** — user star corrections are periodically analyzed by the LLM to generate a text preference profile. This profile is included in future scoring prompts. No embeddings or fine-tuning.
- **Quarantine state machine** — resources that fail 5 consecutive times are quarantined and skipped. Manual retry from the UI resets the state.
- **Effective stars = user_stars ?? ai_stars** — user override always wins for display and sorting.
- **Ephemeral article chat** — chat history lives in the browser only, not persisted to the DB.
- **Test mocking** — the AI package exposes `clientCompleteFunc` as a package-level variable that tests override. The engine package exposes `DefaultHTTPClient`.

## PocketBase Collections

| Collection | Purpose |
|-----------|---------|
| `resources` | RSS feeds and watchlist URLs to monitor |
| `entries` | Discovered articles with summaries and star ratings |
| `preferences` | AI-generated preference profile (single record, updated periodically) |
| `app_settings` | Key-value settings (openrouter_api_key, openrouter_model) |

All collections require authentication (`@request.auth.id != ''`). Collections are auto-created on first startup via `registerCollections()`.

## How to Work on This Project

### Running locally

```bash
# Frontend dev (with hot reload)
cd ui && bun install && bun run dev

# Backend dev (serves on :8090)
make dev

# Full build (frontend + backend)
make build

# Run tests
make test
```

### Running tests

```bash
# All tests
go test ./internal/... -count=1

# With coverage
go test ./internal/ai/... ./internal/engine/... ./internal/routes/... \
  -coverprofile=coverage.out -covermode=atomic
go tool cover -func=coverage.out | grep total
```

Test coverage target is >90%. Current coverage: ~90.7%.

The `testutil` package provides helpers that create an in-memory PocketBase app with all collections registered. Use these in every test:
- `testutil.NewTestApp(t)` — returns `(core.App, cleanup func())`
- `testutil.CreateResource(...)`, `testutil.CreateEntry(...)`, etc.

### Coding conventions

- Keep changes minimal and focused — one concern per change
- All business logic goes in `internal/` packages, not in `cmd/`
- The `cmd/knowledgehub/` files only wire things together (PocketBase setup, hooks, collection registration)
- Tests use table-driven style where appropriate
- AI client calls are mocked via `clientCompleteFunc` variable override in tests
- HTTP calls are mocked via `DefaultHTTPClient` or `httptest.NewServer`

### Adding a new PocketBase collection

1. Add `ensure<Name>Collection(app core.App)` function in `cmd/knowledgehub/collections.go`
2. Call it from `registerCollections()`
3. Add matching `Create<Name>` helper in `internal/testutil/testutil.go`

### Adding a new API endpoint

1. Add handler in `internal/routes/`
2. Register it via `se.Router` in `main.go`'s `OnServe` hook
3. Write tests using `NewTestApp` and `httptest`

## How to Release a New Version

The release produces a single Linux amd64 binary with the frontend embedded.

```bash
# 1. Ensure tests pass
make test

# 2. Build the release binary
make release

# 3. The output is at:
#    build/knowledgehub                          — the binary
#    build/knowledgehub-linux-amd64.tar.gz       — tarball with binary + systemd unit
```

To deploy to the LXC host:

```bash
# Copy to the target machine (via Tailscale)
scp build/knowledgehub-linux-amd64.tar.gz knowledgehub-host:/tmp/

# On the target machine:
ssh knowledgehub-host
cd /opt/knowledgehub
sudo systemctl stop knowledgehub
tar xzf /tmp/knowledgehub-linux-amd64.tar.gz
sudo systemctl start knowledgehub
```

The systemd service file (`knowledgehub.service`) expects:
- Binary at `/opt/knowledgehub/knowledgehub`
- Data directory at `/opt/knowledgehub/data` (set via `KH_DATA_DIR`)
- Listens on `0.0.0.0:8090`
- Runs as `knowledgehub` user/group
