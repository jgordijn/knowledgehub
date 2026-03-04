## Context

KnowledgeHub currently requires users to create a full resource (RSS feed or watchlist) to add content. There's no way to quickly add a single article URL. Users who encounter an interesting article while browsing want a fast path: paste the URL, get it summarized, and optionally subscribe to the site if it has an RSS feed.

The existing infrastructure already supports the building blocks:
- `engine.ExtractContent()` fetches a URL and extracts article text via readability
- `engine.FetchRSS()` parses RSS/Atom/JSON feeds via gofeed
- `engine.createEntry()` creates entries with AI processing
- Entries require a `resource` relation (foreign key to resources collection)

## Goals / Non-Goals

**Goals:**
- One-tap "Quick Add" from the feed view to paste a URL and create a summarized entry
- Auto-discover RSS feeds on the article's site
- Show a preview of the RSS feed (last 5 articles as links with dates) so the user can evaluate the content
- Let the user subscribe to the RSS feed in one click, edit first, or skip
- Seamless integration — one-off entries appear in the feed like any other entry

**Non-Goals:**
- Watchlist auto-discovery (only RSS detection, not scraping pattern detection)
- Bulk URL import (one URL at a time)
- Persisting RSS discovery results for later (ephemeral, shown once in the modal)
- Changing how the scheduler handles resources (Quick Add resource is never fetched by scheduler)

## Decisions

### 1. System "Quick Add" resource for one-off entries

**Decision:** Create an auto-created system resource (type `quickadd`, name "Quick Add") that parents all one-off entries.

**Rationale:** Entries have a required `resource` relation. Rather than making the relation optional (which would break existing filters like `resource.active = true`), we use a dedicated resource. This resource is always active, never fetched by the scheduler, and hidden from the resources management page.

**Alternatives considered:**
- Make `resource` optional on entries → breaks all existing PocketBase filters, queries, and the expand pattern
- Create a new `quickadd_entries` collection → duplicates schema, breaks unified feed view

### 2. Single endpoint that does everything

**Decision:** One `POST /api/quick-add` endpoint that accepts `{ url: string }` and returns: the created entry, plus any discovered RSS feed info (feed URL, last 5 articles with title/date/link).

**Rationale:** The frontend needs all the info in one round-trip to show the modal. The article is already created by the time the response comes back — the RSS discovery is additive info. This keeps the UX fast: the article is added immediately, and the RSS preview is a bonus.

**Flow:**
1. Backend receives URL
2. Fetches page HTML via readability → creates entry under Quick Add resource → triggers AI processing
3. Concurrently: fetches page HTML headers looking for RSS `<link>` tags
4. If RSS found: fetches the feed, parses last 5 articles (title, URL, published date)
5. Returns: `{ entry: {...}, rss: { feed_url, articles: [{title, url, published}] } | null }`

### 3. RSS auto-discovery via HTML `<link>` tags

**Decision:** Fetch the article page HTML and look for `<link rel="alternate" type="application/rss+xml">` (and atom+xml, application/feed+json) in the `<head>`. If the article URL doesn't have these tags, also try the site root (e.g., `https://example.com/`).

**Rationale:** This is the standard RSS discovery mechanism used by browsers and feed readers. Checking both the article page and site root covers most cases. We already have the article page HTML from the readability fetch, so we can extract `<link>` tags from the same response.

### 4. Add RSS resource endpoint

**Decision:** A second endpoint `POST /api/quick-add/subscribe` that accepts `{ feed_url: string, name: string }` and creates an RSS resource. The frontend calls this when the user clicks "Add RSS" or submits the edit form.

**Rationale:** Separating subscribe from the initial quick-add keeps concerns clean. The frontend already has the resource form component (`ResourceForm.svelte`) that can be reused for the "Edit" flow.

### 5. Frontend modal with three states

**Decision:** The QuickAddModal has three states:
1. **Input** — URL text field + "Add" button
2. **Processing/Result** — shows the added article confirmation + RSS discovery (if found): feed URL link, 5 article previews as links, and three action buttons (Add RSS / Edit / No thanks)
3. **Edit** — inline resource form pre-filled with feed URL and site name (reuses ResourceForm component)

### 6. Scheduler skips quickadd resources

**Decision:** The scheduler's fetch loop already filters by resource type (rss/watchlist). Since `quickadd` is neither, it's automatically skipped. No code change needed in the scheduler.

## Risks / Trade-offs

- **[Risk] RSS discovery may not find feeds on all sites** → Acceptable. The modal gracefully handles the no-RSS case by just confirming the article was added. Many modern sites do have RSS `<link>` tags.
- **[Risk] Article fetch may fail (paywalls, JS-rendered content)** → Same limitations as existing watchlist scraping. The endpoint returns an error and the frontend shows it. Could be enhanced later with browser-based fetching.
- **[Risk] Quick Add resource could accumulate many entries** → No different from any other active resource. User can mark entries as read or filter them out. The resource is always active so entries always show up.
- **[Trade-off] Two HTTP requests for subscribe flow** → The quick-add endpoint creates the entry, then a second call creates the resource if the user wants RSS. This is simpler than trying to handle both in one request with conditional behavior.
