## Context

The KnowledgeHub feed view currently uses a flat card layout where every entry renders identically via `EntryCard.svelte`, displayed in a `+page.svelte` grid inside a top-nav app shell (`Nav.svelte` + `+layout.svelte`). Source filtering is a collapsible single-select pill bar embedded in the feed page.

The River redesign mockup (`designs/mockup.html`) demonstrates a complete visual redesign with tiered cards, sidebar navigation, multi-select source filtering, universal expand/collapse, and section-level batch controls. The mockup is a standalone HTML file — all logic is vanilla JS. This design describes how to translate that mockup into the real SvelteKit 5 + Tailwind CSS application.

The backend requires zero changes — all work is frontend.

## Goals / Non-Goals

**Goals:**
- Implement the River mockup's visual design and interactions in production SvelteKit
- Tiered card rendering based on effective star rating (5★, 4★, 3★, 1–2★)
- Universal expand/collapse on every card tier with per-tier defaults
- Card-level click-to-open (whole card is a tap target)
- Section headers with progressive-disclosure batch controls
- Sidebar-based layout with multi-select source filtering
- Mobile-responsive sidebar as slide-out drawer
- Preserve all existing functionality (realtime updates, chat, link panel, quick add, undo, bookmark/read later)

**Non-Goals:**
- Keyboard shortcuts (j/k navigation, hotkeys) — future change
- Swipe gestures — future change
- Renaming "Resources" → "Sources" in routes/URLs — only visual rename in sidebar label
- Chat panel integration into sidebar — chat remains as an overlay/modal
- Theme toggle redesign — keep existing light/dark/system toggle, just move it to sidebar

## Decisions

### 1. App shell: sidebar replaces top nav

**Decision:** Replace the `Nav.svelte` top bar with a new `Sidebar.svelte` component. The `+layout.svelte` changes from a single-column layout to a `flex` row: sidebar (fixed 220px) + main content area. On mobile (≤768px), the sidebar becomes `position: fixed` off-screen and slides in via a hamburger button.

**Rationale:** The mockup's sidebar provides space for always-visible source filters, which is the core UX improvement. A top nav can't hold a source filter list without collapsing it behind a dropdown.

**Alternative considered:** Keep top nav and add a collapsible sidebar panel. Rejected because it creates two navigation paradigms and the mockup already validated the sidebar approach.

**Implementation:**
- New `Sidebar.svelte` component with: logo + version + GitHub link, nav links (Feed, Saved, Sources, Settings), source filter section, footer with shortcuts hint
- `+layout.svelte` renders `<Sidebar>` + `<main>` in a flex row
- Mobile: hamburger button in a minimal topbar, sidebar slides from left with overlay backdrop
- `Nav.svelte` is kept but deprecated (may be removed in a follow-up)

### 2. Tiered card rendering via a single component with tier prop

**Decision:** Keep a single `EntryCard.svelte` component but introduce an `effectiveStars`-based tier that selects between four rendering modes internally. Do NOT split into four separate components.

**Rationale:** The four tiers share substantial logic — star rating, mark-read, bookmark, chat, link panel, fragment content, referenced links, realtime updates. Duplicating this across four components would create a maintenance burden. The visual differences are handled by conditional Tailwind classes and `{#if}` blocks within one component.

**Tier mapping:**
| Effective Stars | Tier | CSS class | Default state |
|----------------|------|-----------|---------------|
| 5 | Featured | `.tier-featured` | Expanded |
| 4 | High Priority | `.tier-hp` | Expanded |
| 3 | Worth a Look | `.tier-wal` | Collapsed |
| 1–2 (or 0/pending) | Low Priority | `.tier-lp` | Collapsed |

**Implementation:**
- Add `tier` derived state: `let tier = $derived(...)` based on `effectiveStars`
- Add `expanded` local state, initialized from tier default
- Render different layouts per tier using `{#if tier === 'featured'}` etc.
- Featured: full-width card with amber left border, large title, summary, takeaways, full action buttons
- HP: medium card with summary (2-line clamp), compact side buttons
- WaL: single-line row with stars, title, source avatar, time, expand button
- LP: minimal row, muted opacity (50%), same structure as WaL

### 3. Expand/collapse as local component state

**Decision:** Each `EntryCard` manages its own `expanded` boolean as local `$state`. No global expand/collapse store. Section-level batch controls communicate via a callback pattern.

**Rationale:** Expand/collapse is ephemeral UI state — it doesn't need to survive page navigation or be synced to the server. Local state keeps the component self-contained. Section batch controls use the parent page's knowledge of which entries are in each tier to iterate and call toggle functions.

**Implementation:**
- `EntryCard.svelte` exposes `expanded` state and a `toggle()` method via `bind:this` or via an `onToggle` callback
- Actually, simpler: the page passes an `expanded` prop and an `onToggle` callback. The page manages an `expandedSet: Set<string>` tracking which entry IDs are expanded. This allows section-level batch operations.
- Default expansion: Featured and HP entries start in the set; WaL and LP do not.

### 4. Source filtering: multi-select with Set state

**Decision:** Replace the single `resourceFilter: string` state in `+page.svelte` with `selectedSources: Set<string>` stored as reactive state. The PocketBase query builds a filter condition from the set.

**Behavior:**
- Empty set = all sources shown (unfiltered)
- Non-empty set = only entries from selected sources shown
- Client-side filtering (not server-side) for instant toggle response, since we already load up to 200 entries

**Rationale:** Client-side filtering avoids a round-trip on every toggle click. The entry list is small enough (200 max) that this is performant.

**Implementation:**
- `selectedSources = $state(new Set<string>())` in `+page.svelte`
- `filteredEntries` derived adds source filter: `result.filter(e => selectedSources.size === 0 || selectedSources.has(e.resource))`
- `Sidebar.svelte` receives `resources`, `selectedSources`, and callbacks `onToggleSource`, `onClearSources`
- Topbar area shows dismissable chips for each active source

### 5. Section headers as a lightweight inline pattern

**Decision:** Section headers are rendered directly in `+page.svelte`'s template, not as a separate component. Each section is an `{#if}` block that checks whether there are entries in that tier.

**Rationale:** Section headers are tightly coupled to the grouping logic in the page. Extracting them adds indirection without reuse benefit — there's only one feed page.

**Implementation:**
- `+page.svelte` groups `filteredEntries` into four arrays: `featuredEntries`, `hpEntries`, `walEntries`, `lpEntries` (derived)
- For each non-empty group, render a section header `<div>` with the tier label and batch buttons
- "Expand all" visible when any entry in the section is collapsed
- "Collapse all" visible when any entry in the section is expanded
- Buttons iterate over the tier's entries and update `expandedSet`

### 6. Card-level click handler preserved from current implementation

**Decision:** Keep the existing `handleCardClick` pattern from `EntryCard.svelte` — it walks up from `event.target` checking for interactive elements (a, button, input, select, textarea, role="button"). If none found, it opens the article. This already exists in the current code and works correctly.

**Rationale:** The current implementation already matches the mockup's `openArticle()` behavior. No changes needed to the click handler logic itself — only the visual layout changes.

### 7. CSS approach: Tailwind utility classes with minimal custom CSS

**Decision:** Use Tailwind utility classes for all styling. Use custom CSS only for the expand/collapse animation (`@keyframes slideDown`) and the click-flash feedback effect. No CSS modules or scoped styles for layout.

**Rationale:** The project already uses Tailwind exclusively. The mockup's CSS variables (`--bg`, `--card`, etc.) translate directly to Tailwind's dark mode utilities. Custom CSS is only needed for animations that Tailwind doesn't cover natively.

### 8. Mobile layout: hamburger + slide-out sidebar + overlay

**Decision:** On viewports ≤768px:
- Sidebar is `position: fixed; left: -220px` by default
- A hamburger button appears in a minimal topbar
- Clicking hamburger slides sidebar in (`left: 0`) with a semi-transparent overlay
- Clicking overlay or a nav link closes the sidebar

**Rationale:** This matches the mockup exactly and is a standard mobile sidebar pattern. The current `Nav.svelte` hamburger/mobile-menu logic can be adapted.

## Risks / Trade-offs

- **Risk: Large diff** — This change touches `+layout.svelte`, `+page.svelte`, `EntryCard.svelte`, and adds `Sidebar.svelte`. It's a significant frontend rewrite.
  → Mitigation: Implement incrementally — sidebar first, then tiered cards, then expand/collapse, then batch controls. Each step is independently testable.

- **Risk: Expand/collapse state lost on filter change** — When `readFilter` or source filter changes, `loadEntries()` replaces the entry list. The `expandedSet` references entry IDs that might not be in the new list.
  → Mitigation: Re-initialize `expandedSet` defaults after each `loadEntries()` call. Featured and HP entries get added to the set; WaL and LP don't.

- **Risk: Performance with many expanded cards** — Expanding all 200 entries simultaneously could cause layout thrash.
  → Mitigation: The "Expand all" buttons exist per-section, so at most one tier's cards expand at once. Typical sections have 5–20 entries.

- **Trade-off: Client-side source filtering vs server-side** — Client-side means we always fetch all 200 entries regardless of source filter. This is fine for the current scale but won't work if entry counts grow significantly.
  → Accepted: The 200-entry limit is already a hard cap. If growth demands it, source filtering can move server-side later without UI changes.

- **Trade-off: Single EntryCard component vs per-tier components** — A single component with four rendering paths is complex but avoids duplicating shared logic.
  → Accepted: The shared logic (star rating, mark-read, bookmark, chat, click handler) is substantial enough to justify one component.
