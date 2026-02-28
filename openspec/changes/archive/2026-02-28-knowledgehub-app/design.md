## Context

Greenfield personal project. No existing codebase beyond a Docker sandbox setup. The goal is a single-binary web application that monitors RSS feeds and blogs, summarizes content with AI, and presents a filterable feed on any device via Tailscale.

The project runs on an LXC container on Proxmox. No Docker at runtime. The binary must be self-contained with the frontend embedded.

## Goals / Non-Goals

**Goals:**
- Single binary deployment with zero runtime dependencies (no Node, no Docker)
- Mobile-friendly web UI for triage workflow (read/unread, star filter, open article)
- AI-powered summarization and relevance scoring that learns from user feedback
- Robust source monitoring with automatic quarantine for broken sources
- Chat with article content for deeper understanding

**Non-Goals:**
- Multi-user support — single user behind Tailscale
- Full-text search — not in v1, can add later
- Offline/PWA support — requires network to reach the LXC anyway
- Social features (sharing, commenting)
- Mobile native app — responsive web is sufficient

## Decisions

### 1. PocketBase embedded in Go

**Choice**: Embed PocketBase as a Go library, not run it standalone.

**Why**: PocketBase gives us auth, auto-generated REST API, realtime SSE, and SQLite management for free. Embedding it lets us add custom Go code (scheduler, fetcher, AI integration) in the same process. Single binary, single SQLite file.

**Alternatives considered**:
- Pure Go + SQLite (e.g., go-chi + modernc.org/sqlite): Full control but we'd rewrite auth, API routing, migrations, and realtime. Too much plumbing for a personal tool.
- Standalone PocketBase + separate Go worker: Two processes to manage, IPC complexity.

### 2. SvelteKit with static adapter + Bun

**Choice**: SvelteKit built with Bun, output as static files, embedded in Go binary via `//go:embed`.

**Why**: Svelte is a learning goal. Static adapter means no JS runtime at deployment. Bun is already in the project toolchain and faster than npm. Go's embed directive bakes the files into the binary.

**Alternatives considered**:
- HTMX + Go templates: Simpler but misses the Svelte learning goal.
- React/Next.js: Heavier, more complex for this use case.

### 3. OpenRouter for LLM access

**Choice**: Use OpenRouter as the single LLM gateway.

**Why**: Model-agnostic — switch between Claude, GPT-4, Llama, etc. without code changes. Single API key, unified billing. Good for experimenting with which model gives best summaries.

**Integration points**:
- Summarization: called per new entry
- Star scoring: called per new entry (can batch with summarization in one prompt)
- Preference profile generation: called periodically (every N corrections)
- Article chat: streaming endpoint, called on-demand

### 4. Preference learning via prompt engineering

**Choice**: Store user corrections, periodically regenerate a text-based preference profile via LLM, include profile + recent corrections in scoring prompts.

**Why**: No embeddings, no vector DB, no fine-tuning. The LLM extracts patterns from correction history. Simple, effective, and works with any model.

**Data flow**:
1. User changes star rating on an entry → correction stored (ai_stars vs user_stars)
2. Every 20 corrections (or weekly), LLM generates a preference profile from all corrections
3. New articles scored using: preference profile + last 20 corrections + article content

### 5. Two-mode content fetching

**Choice**: RSS parser for feeds, go-readability for blog scraping.

**Why**: RSS covers most blogs. For blogs without RSS, readability extraction (same as Firefox Reader Mode) handles article content cleanly without site-specific selectors.

**Libraries**:
- RSS: `gofeed` (well-maintained, handles RSS/Atom/JSON Feed)
- Readability: `go-readability` (port of Mozilla's readability)
- HTTP: standard `net/http` with timeouts

### 6. Quarantine after 5 consecutive failures

**Choice**: Track consecutive failures per resource. After 5, quarantine (stop checking). User can manually retry or remove.

**Why**: Prevents wasting cycles on dead sources. 5 failures at 30-min intervals = 2.5 hours of grace, enough to survive temporary outages.

**State machine**: healthy → failing (1-4 failures) → quarantined (5+). Success resets to healthy.

### 7. Browser notifications via existing PocketBase realtime

**Choice**: PocketBase SSE realtime subscriptions + Browser Notification API.

**Why**: PocketBase already emits realtime events on record changes. Frontend subscribes, filters for 5★ entries, shows browser notification. Zero additional infrastructure.

**Limitation**: Only works when tab is open (or backgrounded). Acceptable for v1.

### 8. Ephemeral article chat

**Choice**: Chat history lives in browser state only. Not persisted to DB.

**Why**: The value is in-the-moment understanding. Simplifies data model. Can add persistence later if needed.

**Streaming**: Custom Go endpoint (`POST /api/chat`) pipes OpenRouter streaming response back to client via SSE.

## Risks / Trade-offs

- **[CGO dependency]** SQLite requires CGO_ENABLED=1, complicating cross-compilation. → Build on the target architecture (LXC is linux/amd64) or use zig as C cross-compiler.
- **[OpenRouter availability]** If OpenRouter is down, new entries won't get summaries/scores. → Queue entries for processing, retry on next cycle. Show "pending" state in UI.
- **[Scraping fragility]** Blog layouts change, readability extraction may fail. → Fall back to title + first paragraph. Quarantine mechanism handles persistent failures.
- **[LLM cost]** Each entry costs an API call for summarization + scoring. At 10-20 sources, maybe 50-100 entries/day. → Batch summary + scoring into one prompt per entry. Estimate: ~$0.50-2.00/day depending on model.
- **[Preference profile quality]** Early on, few corrections means weak profile. → Start with no profile (generic scoring), activate after 10+ corrections.

## Project Structure

```
knowledgehub/
├── cmd/knowledgehub/
│   └── main.go              # Entry point, PocketBase setup
├── internal/
│   ├── engine/
│   │   ├── scheduler.go     # 30-min fetch cycle
│   │   ├── fetcher.go       # RSS + scrape dispatch
│   │   ├── rss.go           # RSS/Atom parsing
│   │   ├── scraper.go       # Readability extraction
│   │   └── quarantine.go    # Health state management
│   ├── ai/
│   │   ├── client.go        # OpenRouter HTTP client
│   │   ├── summarizer.go    # Summarize + score prompt
│   │   ├── preference.go    # Profile generation
│   │   └── chat.go          # Article chat streaming
│   └── routes/
│       └── chat.go          # Custom /api/chat endpoint
├── ui/                       # SvelteKit app
│   ├── src/
│   │   ├── routes/
│   │   │   ├── +page.svelte          # Feed view
│   │   │   ├── resources/
│   │   │   │   └── +page.svelte      # Resource management
│   │   │   └── settings/
│   │   │       └── +page.svelte      # Settings
│   │   └── lib/
│   │       ├── components/
│   │       │   ├── EntryCard.svelte   # Article card
│   │       │   ├── StarRating.svelte  # Star widget
│   │       │   ├── ChatPanel.svelte   # Article chat
│   │       │   └── ResourceForm.svelte
│   │       └── pb.ts                  # PocketBase client
│   ├── static/
│   ├── svelte.config.js
│   └── package.json
├── go.mod
├── go.sum
├── Makefile
└── knowledgehub.service      # systemd unit
```
