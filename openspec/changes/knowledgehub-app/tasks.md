## 1. Project Scaffolding

- [x] 1.1 Initialize Go module (`go mod init`), create `cmd/knowledgehub/main.go` with PocketBase embedded setup
- [x] 1.2 Create PocketBase collections/migrations for resources, entries, preferences, and settings tables
- [x] 1.3 Scaffold SvelteKit app in `ui/` with Bun, static adapter, Tailwind CSS, and PocketBase JS SDK
- [x] 1.4 Set up `//go:embed` for `ui/build` directory and configure PocketBase to serve static files
- [x] 1.5 Create Makefile with `build`, `dev`, and `release` targets
- [x] 1.6 Create systemd service unit file (`knowledgehub.service`)

## 2. Resource Management (Backend)

- [x] 2.1 PocketBase collections for resources with all fields (name, url, type, selectors, status, consecutive_failures, last_error, quarantined_at, active, check_interval, last_checked)
- [x] 2.2 Add API rules for resources CRUD (single user auth)
- [x] 2.3 Add hook: on resource URL update, reset consecutive_failures to 0 and status to "healthy"
- [x] 2.4 Add hook: on resource delete, cascade delete associated entries

## 3. Resource Management (Frontend)

- [x] 3.1 Create resources page (`/resources`) with list of all resources showing health status indicators (green/yellow/red)
- [x] 3.2 Create add resource form (name, URL, type selector, optional CSS selectors)
- [x] 3.3 Create edit resource form (inline or modal)
- [x] 3.4 Add delete resource button with confirmation
- [x] 3.5 Add activate/deactivate toggle per resource
- [x] 3.6 Add "Retry Now" button for quarantined resources (resets status, triggers fetch)
- [x] 3.7 Add quarantine banner component (shows count + link to resources page)

## 4. Content Fetching Engine

- [x] 4.1 Implement scheduler (`internal/engine/scheduler.go`) — 30-minute interval, iterates active non-quarantined resources
- [x] 4.2 Implement RSS/Atom/JSON Feed parser (`internal/engine/rss.go`) using `gofeed`, deduplicates by GUID
- [x] 4.3 Implement blog scraper (`internal/engine/scraper.go`) — fetch page, find article links (CSS selector or heuristic), deduplicate by URL
- [x] 4.4 Implement readability content extraction for scraped articles using `go-readability`, with fallback to title + first 500 chars
- [x] 4.5 Implement quarantine state machine (`internal/engine/quarantine.go`) — track failures, transition healthy→failing→quarantined, reset on success
- [x] 4.6 Implement 30-second HTTP fetch timeout
- [x] 4.7 Wire scheduler into PocketBase startup (starts as background goroutine)

## 5. AI Processing

- [x] 5.1 Implement OpenRouter HTTP client (`internal/ai/client.go`) — send prompts, handle streaming, parse responses
- [x] 5.2 Implement combined summarizer + scorer (`internal/ai/summarizer.go`) — single prompt that returns 2-4 line summary and 1-5 star rating as structured output
- [x] 5.3 Implement preference profile generator (`internal/ai/preference.go`) — analyze all corrections, generate text profile, store in preferences table
- [x] 5.4 Add preference profile trigger: regenerate after every 20 user corrections or weekly
- [x] 5.5 Wire AI processing into fetch pipeline: new entries → summarize + score → save
- [x] 5.6 Handle AI failure gracefully: entries created with null summary/stars, marked as pending, retried next cycle

## 6. Settings

- [x] 6.1 Create settings page (`/settings`) with OpenRouter API key input and model selection dropdown
- [x] 6.2 Store settings in PocketBase settings collection (key-value)
- [x] 6.3 Backend reads API key and model from settings for all AI operations

## 7. Feed View (Frontend)

- [x] 7.1 Create feed page (`/`) with entry cards showing: effective stars, source name, relative time, title, summary
- [x] 7.2 Implement card sorting: effective stars DESC, then discovered_at DESC
- [x] 7.3 Add read/unread filter tabs with unread count badge
- [x] 7.4 Add star filter buttons (All, 3+, 4+, 5)
- [x] 7.5 Implement star rating widget component — clickable 1-5 stars, visual distinction between AI-set (dimmed) and user-set (solid)
- [x] 7.6 Implement open original article link (new tab) with auto-mark-read on click
- [x] 7.7 Add manual read/unread toggle button per card
- [x] 7.8 Add pending entry display (spinner/processing indicator for entries with null summary)
- [x] 7.9 Add quarantine banner to feed view (from task 3.7 component)

## 8. Article Chat

- [x] 8.1 Implement streaming chat endpoint (`POST /api/chat`) in Go — loads entry content, builds prompt with article + conversation history, streams OpenRouter response via SSE
- [x] 8.2 Create ChatPanel Svelte component — full-screen overlay on mobile, slide-in panel on desktop
- [x] 8.3 Implement chat message UI with streaming token display
- [x] 8.4 Add chat icon button to entry cards that opens ChatPanel with article context
- [x] 8.5 Implement ephemeral conversation state (browser only, cleared on panel close)

## 9. Notifications

- [x] 9.1 Subscribe to PocketBase realtime (entries collection) in SvelteKit using PocketBase JS SDK
- [x] 9.2 Request browser notification permission on first visit
- [x] 9.3 Trigger browser notification on new entry with effective stars = 5
- [x] 9.4 Update feed view in realtime when new entries arrive (insert card at correct position without refresh)

## 10. Mobile Responsiveness

- [x] 10.1 Make feed view responsive — single column card layout on mobile, comfortable tap targets (44px+)
- [x] 10.2 Make resources page responsive
- [x] 10.3 Make chat panel full-screen on mobile viewports
- [x] 10.4 Test and adjust on 375px viewport width

## 11. Build & Release

- [x] 11.1 Verify `make build` produces working single binary with embedded frontend
- [x] 11.2 Verify `make release` produces tarball with binary + systemd service file
- [x] 11.3 Test fresh deployment: copy binary to clean system, run, verify auto-creates database and serves UI
- [x] 11.4 Test configurable data directory via `KH_DATA_DIR` environment variable
- [x] 11.5 Test configurable listen address via `--http` flag
