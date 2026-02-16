## Why

Keeping up with interesting blogs, feeds, and technical content across the web is overwhelming. There's too much to read and no good way to surface what matters. We need a personal knowledge radar that monitors sources, distills content with AI, learns our preferences, and lets us triage quickly from any device.

## What Changes

- New Go backend with PocketBase (embedded) providing auth, REST API, realtime SSE, and SQLite storage
- SvelteKit frontend (static adapter, embedded in Go binary via `//go:embed`) with Tailwind CSS
- RSS feed parsing and blog scraping (go-readability) with a 30-minute scheduler
- AI-powered summarization and star rating (1-5) via OpenRouter, with preference learning from user corrections
- Read/unread tracking, star-based filtering, and browser notifications for 5★ entries
- Chat with article feature — ask questions about any article's content grounded only in that article
- Resource health monitoring with automatic quarantine after 5 consecutive failures
- Single binary deployment on LXC (Proxmox) behind Tailscale

## Capabilities

### New Capabilities
- `resource-management`: CRUD for RSS feeds and blog watchlist sources, health status tracking, quarantine mechanism
- `content-fetching`: RSS parsing, blog scraping via readability, 30-minute scheduled fetching
- `ai-processing`: Article summarization, star scoring (1-5), preference profile learning from user corrections, OpenRouter integration with model selection
- `feed-view`: Entry list with read/unread, star filtering, sort by stars/date, open original article (auto-marks read)
- `article-chat`: Ephemeral chat with article content via streaming OpenRouter API, grounded only in article text
- `notifications`: Browser push notifications for 5★ entries via PocketBase realtime SSE
- `deployment`: Single Go binary with embedded SvelteKit static files, systemd service, Tailscale networking

### Modified Capabilities
<!-- None — greenfield project -->

## Impact

- New project structure: Go module + SvelteKit app in `ui/` directory
- External dependency: OpenRouter API (requires API key)
- Runtime: SQLite database file on disk, single port (8090)
- Build tooling: requires Go (CGO_ENABLED=1), Bun (for SvelteKit build)
