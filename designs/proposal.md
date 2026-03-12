# KnowledgeHub Redesign Proposal — "River"

## Executive Summary

The current KnowledgeHub UI is functional but treats every article identically — a uniform stack of cards with small action buttons that require deliberate clicks. **River** reimagines the feed as a fast-flowing triage experience where the three core decisions (read now, save for later, dismiss) are visually obvious and kinetically satisfying, while surfacing AI intelligence more prominently to help the user decide faster.

---

## Problems with the Current UI

### 1. Visual monotony — every card looks the same
All entries use the same card weight regardless of star rating. A 5-star breakthrough article and a 2-star noise post are visually identical until you squint at tiny star icons. The eye has no anchor.

### 2. Triage friction
The action buttons (✓, bookmark, 🤖, ↗) are small, unlabeled, and clustered at the bottom of each card. On mobile these are hard targets. There's no clear "this is what you should do next" signal.

### 3. Summary buried under metadata
The card hierarchy is: stars → source → time → title → summary → takeaways → links → actions. But the *decision-relevant information* is the title + summary + star rating. Source and time are secondary context, yet they dominate the top row.

### 4. Filter bar is busy and unclear
Three separate filter groups (read/unread, star minimum, mark-as-read dropdown) plus a collapsible source filter creates a dense toolbar that's hard to parse on first glance. The "Mark read ▾" dropdown is ambiguously placed — it looks like a filter but it's actually a destructive action.

### 5. Chat panel competes with the feed
The chat panel slides in from the right and covers the feed on mobile. There's no visual relationship between the article being discussed and the chat.

---

## Design Principles

1. **Density with hierarchy** — Show more entries per viewport, but make important ones visually louder
2. **Triage-first actions** — The three key actions (Read, Save, Dismiss) should be the most prominent interactive elements on each card
3. **AI as guide, not chrome** — Star ratings and summaries should directly shape visual weight, not just be data displayed
4. **Progressive disclosure** — Keep the first scan clean; reveal depth on hover/expand
5. **Keyboard-friendly** — Support j/k navigation, Enter to open, b to bookmark, x to mark read

---

## Information Architecture

### Feed View (Home) — The River

```
┌──────────────────────────────────────────────────────────┐
│ KnowledgeHub                    [Unread 23] [Saved 4]    │
│ ─────────────────────────────────────────────────────── │
│ Sources ▾   Stars ▾   Mark read ▾            [+ Add]    │
├──────────────────────────────────────────────────────────┤
│                                                          │
│ ┌─ ★★★★★ ──────────────────────────── Featured ──────┐ │
│ │  Understanding Rust's Borrow Checker              2h │ │
│ │  The article explains ownership semantics through   │ │
│ │  practical examples, comparing to C++ RAII...       │ │
│ │  • Key insight about lifetime elision              │ │
│ │  • Comparison with Swift's ARC model               │ │
│ │  Rust Blog                                          │ │
│ │  [Read Now ↗]  [Save 📌]  [Chat 🤖]               │ │
│ └────────────────────────────────────────────────────┘ │
│                                                          │
│  ★★★★  Cache Invalidation Strategies        4h ago     │
│        Two approaches to distributed cache...           │
│        Engineering Blog · [Read] [Save] [Chat]          │
│                                                          │
│  ★★★★  Building Event-Driven Systems        6h ago     │
│        A comparison of Kafka, Pulsar, and...            │
│        InfoQ · [Read] [Save] [Chat]                     │
│                                                          │
│  ★★★   TLS 1.3 Explained                   1d ago     │
│        Overview of the TLS 1.3 handshake...             │
│        Cloudflare Blog · [Read] [Save] [Chat]           │
│                                                          │
│  ★★    New npm Package Naming Rules         1d ago     │
│        npm is changing how package names...              │
│        Node.js Blog · [✓ Mark read]                     │
│                                                          │
└──────────────────────────────────────────────────────────┘
```

### Key layout decisions:

**1. Tiered card prominence based on stars**
- **5 stars**: Full-width featured card with border accent, expanded summary, takeaways visible, large action buttons
- **4 stars**: Medium card — title + 2-line summary, visible actions
- **3 stars**: Compact — title + 1-line summary, actions on hover
- **1–2 stars**: Minimal row — title only, muted at 50% opacity. **Click to expand in-place** reveals summary + full action set (Read, Save, Mark read). Collapses back on second click. *(Refined in v2)*

This creates a natural visual river where the eye is drawn to high-value content first.

**2. Action buttons redesigned as labeled pills**
Instead of tiny icon-only buttons, use clear labeled buttons:
- **"Read ↗"** — opens article, marks as read (primary blue for 4-5 star, muted for lower)
- **"Save 📌"** — bookmarks for later (amber when active)
- **"Chat 🤖"** — opens chat panel

On low-star entries, the collapsed row shows only the title and a ✓ button. Expanding reveals the full action set (Read, Save, Mark read) so users can still engage if the AI under-scored something relevant. *(Refined in v2)*

**3. Unified filter bar** *(Refined in v2)*
Replace three separate filter groups with a clean single-line bar:
- Left: segmented toggle for **Unread (23) | Saved (4) | All**
- Center: **active source indicator** — a read-only pill showing the sidebar-selected source with a ✕ clear button; hidden when "All Sources" is active. This is *not* a filter control — it's feedback.
- Right: star minimum dropdown **"★ 3+" / "★ 4+" / "★ 5"** and **"Mark read ▾"** action button (visually separated from filters by a divider)

**Source filtering lives exclusively in the sidebar source list** — the topbar only reflects the current selection. The old "All Sources" `<select>` dropdown has been removed to eliminate the dual-control confusion.

**4. Source shown as a subtle colored tag**
Each source gets a consistent 2-letter avatar + color (deterministic from name hash). This provides instant visual recognition of sources without needing to read the name. The full source name appears below the summary in a smaller font.

**5. Relative time moved to the right edge**
Time is useful for sorting context but not for the read/skip decision. Push it to the right margin in a subdued color.

### Chat Panel — "Sidebar"

- Opens from the right edge at 420px width (same as current)
- **New**: Shows the article's summary at the top of the chat, collapsed behind a "Context" toggle, so you can reference it while chatting
- **New**: Article title in the header is a clickable link that opens the article
- **New**: Suggested quick prompts appear before the first message: "Summarize the key argument", "What are the practical implications?", "What does the author get wrong?"
- Keyboard: Escape to close, Ctrl+Arrow to resize (preserved)

### Resources Page — "Sources"

Mostly preserved with minor improvements:
- Rename "Resources" → "Sources" in the nav (clearer mental model)
- Group sources by health status: Healthy at top, Failing in a warning section, Quarantined in a danger section
- Inline the quarantine banner directly into the sources page rather than showing it on the feed

### Settings Page

No significant changes needed. Already clean and well-organized.

---

## Color System

The current app uses Tailwind's slate palette effectively. The redesign keeps this foundation but adds:

- **Accent color for high-value content**: A warm amber/gold left-border for 5-star featured cards
- **Source colors**: Deterministic pastel colors for source avatars (generated from name hash)
- **Action button colors**: Blue for "Read", Amber for "Save", Slate for "Chat" — each action has a distinct color identity
- **Star visualization**: Replace the discrete ★/☆ toggle with a horizontal gold bar that fills proportionally — faster to scan than counting individual stars

---

## Interaction Patterns

### Swipe gestures (mobile)
- **Swipe right** → Mark as read (green flash confirmation)
- **Swipe left** → Bookmark for later (amber flash confirmation)
- Swipe targets are generous (full card width)

### Keyboard shortcuts (desktop)
- **j/k** — Move focus between entries
- **Enter** or **o** — Open article (Read Now)
- **b** — Toggle bookmark
- **x** — Mark as read
- **c** — Open chat
- **/** — Focus source filter
- **?** — Show keyboard shortcut overlay

### Undo pattern (preserved and improved)
The current undo toast is good. Enhance it:
- Show the article title in the undo toast, not just count
- Auto-dismiss after 10s (down from 15s — faster flow)
- Allow multiple sequential undos (current behavior is correct)

---

## Why This Wins

1. **Faster scanning** — Tiered cards let you skip 1-2 star content instantly. The eye goes to featured cards first, then scans down through decreasing prominence.

2. **Clearer decisions** — Every card answers "what should I do?" with labeled, color-coded action buttons instead of ambiguous icons.

3. **Less cognitive load** — The filter bar is one clean line instead of three rows. Source context is ambient (colors) rather than requiring reading.

4. **Respects the AI** — The star rating actually *shapes* the UI rather than being a passive number. The AI's judgment is reflected in how much screen real estate and visual weight each article gets.

5. **Mobile-first triage** — Swipe gestures make the phone experience fast. The current UI requires precise taps on small button targets.

6. **Implementable today** — Everything proposed uses standard Tailwind utilities and Svelte 5 patterns. No new libraries needed. The tiered card system is just conditional CSS classes based on `effectiveStars()`. Keyboard shortcuts are a single `keydown` handler. Swipe gestures can use `touch-action` and pointer events.

---

## Migration Path

This is an incremental redesign, not a rewrite:

1. **Phase 1**: Implement tiered cards (change EntryCard.svelte to accept star tier and render differently)
2. **Phase 2**: Redesign filter bar as a single-line component
3. **Phase 3**: Add keyboard navigation (new Svelte action)
4. **Phase 4**: Add swipe gestures (new Svelte action)
5. **Phase 5**: Enhance chat panel with context and quick prompts

---

## Refinement v2 — Addressing Feedback

This section documents changes made to the original River proposal after review.

### 1. Low-priority items are now expandable/collapsible in-place

**Problem**: The original design showed 1–2 star items as minimal rows with only a "✓ Mark read" button. This made them essentially write-off-only — there was no way to inspect the content without opening the full article externally.

**Resolution**: Low-priority rows now have click-to-expand behavior. The collapsed state remains visually muted (50% opacity, small text, single-line title). Clicking a row expands a detail panel directly below it, revealing:
- Full title (readable size)
- AI summary
- Source + time metadata
- Full action set: Read ↗, Save 📌, Mark read ✓

The expand/collapse is in-place — no navigation, no modal. The row gets a subtle `▸`/`▾` chevron hint. On expand, opacity lifts to ~85% so the content is fully readable while still being visually subordinate to higher-tier cards.

**Rationale**: Not every 2-star article is noise — sometimes the AI under-scores something the user cares about. Expand-in-place lets users quickly check without losing their scroll position in the River.

### 2. Expanded low-priority items are readable, not muted

**Problem**: If expanded items stayed at 50% opacity, the summary text would be unreadable.

**Resolution**: The collapsed row stays muted (opacity 0.5). When expanded, the row header lifts to 0.85 opacity and the detail panel renders at full contrast inside a bordered card — identical styling to the "Worth a Look" tier. This gives a clear visual signal that the user has chosen to engage with this item.

### 3. Source filtering lives in the sidebar

**Problem**: The original design had an "All Sources" dropdown in the topbar *and* a Sources nav link in the sidebar. This created two source-related controls with different purposes — one for filtering, one for CRUD management — but their labels didn't make the distinction clear.

**Resolution**: The sidebar now contains a dedicated **"Filter by Source"** section showing all subscribed sources as a clickable list. Each source shows its avatar color, name, and unread count. The active source gets a highlighted background + accent border. "All Sources" at the top is the default/reset option.

This makes the sidebar the **single, authoritative place** to filter by source. The source list is always visible (on desktop) which makes it faster than a dropdown — no click-to-reveal needed.

### 4. Topbar "All Sources" dropdown removed

**Problem**: Having a source dropdown in the topbar duplicated the sidebar's source filtering, creating confusion about which control is canonical.

**Resolution**: The topbar "All Sources" `<select>` has been removed entirely. When a source filter is active (selected in sidebar), a small blue tag appears in the topbar showing the active source name with an ✕ to clear it — this serves as a visible reminder + quick escape, not as a filter control.

The topbar now contains only: Unread/Saved/All tabs, the active-source tag (when applicable), star minimum filter, theme toggle, and Mark read action. This is cleaner and avoids the "which source control do I use?" confusion.

### 5. River direction intact

The core River metaphor is preserved: content flows top-to-bottom with decreasing visual prominence. Featured (5★) → High Priority (4★) → Worth a Look (3★) → Low Priority (1–2★). The expand/collapse addition to low-priority items doesn't break this hierarchy — collapsed items remain the least visually prominent elements in the feed. Expanding one temporarily lifts it for inspection, then it collapses back to muted.

### Summary of mockup changes

| Element | Before (v1) | After (v2) |
|---------|------------|------------|
| Low-priority rows | Title + ✓ button only | Expandable: click reveals summary + full actions |
| Low-priority expanded | N/A | Card-style panel, full contrast, readable |
| Source filtering | Topbar `<select>` dropdown | Sidebar source list (always visible) |
| Topbar "All Sources" | `<select>` dropdown | Removed; replaced with active-source tag when filtered |
| River ordering | Top→bottom by stars | Unchanged |


---

## Refinement v3 — Interaction Clarity & Multi-Select Sources

This section documents changes made to the River v2 mockup to resolve interaction ambiguities and add missing affordances.

### 1. Sidebar source filters become multi-select toggles

**Problem**: The v2 sidebar source list was single-select (radio-style) — clicking "Rust Blog" hid everything else. Users who follow 2–3 sources closely but want to exclude noise from the rest had no way to express "show me Rust Blog + InfoQ, hide everything else."

**Resolution**: Each sidebar source item is now an independent toggle. Clicking a source activates/deactivates it without affecting other sources. Visual indicator: a small checkbox square replaces the implicit radio behavior. When active, the checkbox fills with a blue check. When no sources are selected, all sources are shown (unfiltered state). A "Clear" link appears in the section header when any filter is active.

**Topbar feedback**: When multiple sources are active, the topbar shows one blue chip per active source, each with its own ✕ dismiss button. This replaces the single-source tag from v2.

**Rationale**: Multi-select is strictly more expressive than single-select. The previous radio-style forced users to choose one source or all — no middle ground. Multi-select toggles are a familiar pattern (email label filters, GitHub project boards) and map naturally to "show me these, hide the rest."

### 2. Card interactions: open-article vs expand/collapse without conflict

**Problem**: In v2, the entire low-priority row was a click target for expand/collapse. This created a conflict: clicking the article title to open it would also toggle the expand state. There was no way to open an article directly from the collapsed row.

**Resolution**: The two interactions are now separated:
- **Title** is an `<a>` link — clicking it navigates to the article (or opens in a new tab). On the collapsed row, the title is the only inline link.
- **Expand/collapse** uses a dedicated ▸/▾ button at the right edge of the row. Only this button toggles the detail panel.
- The row background is no longer a click target.

This same pattern applies consistently across all tiers: Featured and High Priority card titles are links; action buttons (Read ↗, Save, Chat) are separate click targets. "Worth a Look" row titles are links; the Read button is a separate target.

**Rationale**: Separating navigation from disclosure is a standard accessibility pattern. It avoids the "I clicked to read but it expanded instead" frustration. The expand button is intentionally small and at the edge — it's a secondary action. The title link is front and center — it's the primary action.

### 3. Visible "Collapse all" control for low-priority section

**Problem**: When multiple low-priority rows are expanded, there's no quick way to reset the section. Users have to click each row's collapse button individually.

**Resolution**: A "Collapse all" button appears in the "Low Priority" section header, but only when at least one row is expanded. It collapses all expanded detail panels in one click. When no rows are expanded, the button is hidden to avoid clutter.

**Rationale**: This is a standard progressive-disclosure pattern — the batch action only appears when it's useful. It respects the "clean by default" principle while providing an escape hatch for power users who expand several rows during triage.

### 4. Version text and GitHub link in sidebar

**Problem**: The v2 mockup didn't show the app version or link to the GitHub repository. The real app (Nav.svelte) displays both — a version string next to the logo and a small GitHub icon that links to the repo.

**Resolution**: The sidebar logo area now shows `v0.4.2` text next to the "KnowledgeHub" name, plus a small GitHub SVG icon linking to the repository. This matches the production app's existing pattern: version fetched from `/api/version`, GitHub icon as a subtle link.

**Rationale**: The mockup should reflect real app chrome so design reviews aren't surprised by elements that "appear" in production. The version text helps users confirm they're running the expected release. The GitHub link provides a discoverable path to the repo without taking up navigation space.

### Summary of v3 mockup changes

| Element | Before (v2) | After (v3) |
|---------|------------|------------|
| Source filters | Single-select (radio) | Multi-select toggles with checkbox indicators |
| Topbar source chips | Single chip when filtered | One chip per active source, each dismissable |
| Sidebar "Clear" link | N/A (had "All Sources" reset item) | "Clear" link in section header, visible when filters active |
| Low-priority row click | Entire row toggles expand/collapse | Title is a link (opens article); dedicated ▸ button expands |
| Collapse all | None | Button in section header, visible when any row expanded |
| Version + GitHub | Not shown | Version text + GitHub SVG icon in sidebar logo area |

---

## Design Clarifications — Quick Reference

These six points are the definitive answers to recurring review questions. They are already reflected in the mockup and the v2/v3 refinements above, collected here for fast lookup.

| # | Concern | Answer |
|---|---------|--------|
| 1 | **Can compact / low-priority rows expand?** | Yes. 1–2 ★ rows are collapsed by default (muted, title-only). Click the ▸ button to expand in-place — no modal, no navigation. Click ▾ to collapse. |
| 2 | **Are muted rows readable when expanded?** | Yes. Collapsed rows stay at 50 % opacity. On expand the header lifts to 85 % and the detail panel renders at **full contrast** inside a bordered card (same styling as "Worth a Look" tier). |
| 3 | **Where do users filter by source?** | The **sidebar source list** is the single, authoritative source filter. It uses multi-select toggles — each source can be independently activated/deactivated. |
| 4 | **Does the topbar duplicate source filtering?** | No. The topbar has **no source dropdown**. When sidebar source filters are active, the topbar shows read-only blue chips with each source name and ✕ to remove — purely indicators, not filter controls. |
| 5 | **How do I open an article vs expand a low-priority row?** | Click the **title link** to open the article. Click the **▸ button** to expand the detail panel. These are separate click targets — no conflict. |
| 6 | **Is there a quick way to collapse all expanded low-priority rows?** | Yes. A "Collapse all" button appears in the Low Priority section header when any row is expanded. It hides when all rows are collapsed. |
