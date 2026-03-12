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
- **1–2 stars**: Minimal row — title only, just a "mark read" button visible

This creates a natural visual river where the eye is drawn to high-value content first.

**2. Action buttons redesigned as labeled pills**
Instead of tiny icon-only buttons, use clear labeled buttons:
- **"Read ↗"** — opens article, marks as read (primary blue for 4-5 star, muted for lower)
- **"Save 📌"** — bookmarks for later (amber when active)
- **"Chat 🤖"** — opens chat panel

On low-star entries, only "Mark read ✓" is shown to reduce noise and encourage quick dismissal.

**3. Unified filter bar**
Replace three separate filter groups with a clean single-line bar:
- Left: segmented toggle for **Unread (23) | Saved (4) | All**
- Center: dropdown for **Sources** (shows active source name when filtered)
- Right: star minimum dropdown **"★ 3+" / "★ 4+" / "★ 5"** and **"Mark read ▾"** action button (visually separated from filters by a divider)

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

Each phase is a single PR. No big-bang migration.
