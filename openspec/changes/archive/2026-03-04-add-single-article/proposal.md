## Why

Users sometimes encounter a single interesting article they want to process through KnowledgeHub without subscribing to the entire feed. Currently, the only way to add content is by creating a full resource (RSS or watchlist), which is overkill for a one-off article. A "Quick Add" button on the main feed page lets users paste a URL, get it summarized and scored immediately, and optionally discover and subscribe to the site's RSS feed.

## What Changes

- Add a "Quick Add" button/FAB on the feed view that opens a URL input modal
- New backend endpoint that accepts a URL, fetches the article via readability, creates a one-off entry, and returns the result
- RSS auto-discovery: when processing the URL, the backend checks the site for RSS feed links (via `<link rel="alternate" type="application/rss+xml">` and similar tags)
- If an RSS feed is found, the modal shows:
  - The RSS feed URL as a clickable link (opens in new tab)
  - Last 5 articles from the feed as clickable links (title + published date, each opens in a new tab) so the user can investigate individual articles and gauge the content
  - Three options: **Add RSS** (creates the resource automatically), **Edit** (opens resource form pre-filled), or **No thanks** (just keeps the one-off article)
- One-off entries are parented under an auto-created system resource named "Quick Add" (type `quickadd`, always active) so existing filters and queries continue to work
- The one-off entry goes through the same AI summarize+score pipeline as regular entries

## Capabilities

### New Capabilities
- `quick-add-article`: Covers the Quick Add button, URL input modal, one-off article ingestion, RSS auto-discovery with preview, and the option to subscribe to the discovered feed

### Modified Capabilities
- `resource-management`: Add support for the `quickadd` resource type — auto-created system resource that parents one-off entries, not shown in the resource management UI as a regular editable resource
- `content-fetching`: Add a single-article fetch path (given a URL, extract content via readability without a scheduled resource context)

## Impact

- **Backend**: New API endpoint `POST /api/quick-add` that accepts `{ url: string }`, returns article data + discovered RSS info. New helper in `internal/engine/` for single-URL readability extraction and RSS discovery.
- **Frontend**: New `QuickAddModal.svelte` component, "+" button on the feed view. The modal handles URL input, loading state, RSS discovery display, and the add-RSS flow.
- **Database**: New `quickadd` resource type. An auto-created "Quick Add" resource record (created on first use or at startup). Entries created under it are normal entries with all existing fields.
- **Existing code**: No breaking changes. The feed view `resource.active = true` filter works because the Quick Add resource is always active. The resource management page should hide or visually distinguish the system Quick Add resource.
