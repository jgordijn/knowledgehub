## Context

The River redesign mockup (v2) established the core tiered-card feed layout, sidebar source filtering, and expandable low-priority rows. Review feedback identified four interaction gaps: single-select source filters are too restrictive, row-level click handling conflicts with article navigation, there's no batch collapse, and the mockup is missing production chrome (version, GitHub link).

V3 addressed all four gaps. V4 extends the design further: expand/collapse is now universal across all card tiers, and the entire card is a click target for opening articles.

## Goals / Non-Goals

**Goals:**
- Multi-select source filter toggles in the sidebar with clear visual state *(v3)*
- Conflict-free click targets: dedicated ▸/▾ button for expand/collapse, card-level click for opening articles *(v3+v4)*
- Expand/collapse on ALL four card tiers: Featured, High Priority, Worth a Look, Low Priority *(v4)*
- Card-level click opens article — the entire card is a tap/click target *(v4)*
- "Collapse all" affordance for Worth a Look and Low Priority sections *(v3+v4)*
- Version text + GitHub icon in the sidebar matching the production `Nav.svelte` *(v3)*

**Non-Goals:**
- Implementing these changes in the SvelteKit app (that's a separate change)
- Changing the River tier hierarchy or star-based card prominence
- Adding new features beyond what the mockup already demonstrates

## Decisions

### 1. Multi-select toggles with checkbox indicators

**Decision:** Replace the single-select (radio-style) sidebar source list with independent toggles. Each source item gets a 14×14px checkbox square that fills blue with a ✓ when active.

**Rationale:** Single-select forced an all-or-nothing filter. Multi-select is strictly more expressive and matches familiar patterns (Gmail labels, GitHub filters). The checkbox provides unambiguous state: checked = active, unchecked = inactive.

**Behavior:**
- When no sources are selected → all sources shown (unfiltered, default state)
- When ≥1 source selected → only those sources' entries appear
- "Clear" link in section header removes all selections
- Topbar shows one blue chip per active source, each with ✕ dismiss

### 2. Universal expand/collapse across all card tiers

**Decision:** Every card tier has a dedicated ▸/▾ `<button>` for expand/collapse. Featured and High Priority cards default to expanded (detail visible, button shows ▾). Worth a Look and Low Priority cards default to collapsed (detail hidden, button shows ▸).

**Per-tier behavior:**

| Tier | Default | Collapse hides | Expand reveals |
|------|---------|---------------|----------------|
| Featured (5★) | Expanded | Summary, takeaways, actions | — |
| High Priority (4★) | Expanded | Summary, side action buttons | — |
| Worth a Look (3★) | Collapsed | — | Detail panel: summary, metadata, actions |
| Low Priority (1–2★) | Collapsed | — | Detail panel: summary, metadata, actions |

**Rationale:** A uniform interaction model removes guesswork. In v3, only LP cards had expand/collapse — users couldn't collapse a featured card to reclaim space during scanning or expand a 3★ row to preview content. Universal expand/collapse makes the feed fully user-controllable.

### 3. Card-level click opens the article

**Decision:** Clicking anywhere on a card (except the ▸/▾ button or action buttons) opens the article. The `openArticle(event, url)` handler walks up the DOM from the click target — if it encounters a `<button>`, `<a>`, `<select>`, or `<input>`, it bails out and lets that element handle the event. Otherwise it navigates to the article URL.

**Rationale:** Making the full card a click target is a standard mobile-first pattern. It provides a large, forgiving hit area. The expand button and action buttons use `event.stopPropagation()` as a belt-and-suspenders guard in addition to the target-type check.

**Implementation notes for future app work:**
- `EntryCard.svelte` wraps the card in a `<div>` with `on:click={openArticle}`
- The `openArticle` handler checks `event.target` ancestry for interactive elements
- Expand/collapse buttons call `event.stopPropagation()` in their handler
- Action buttons (Read, Save, Chat, Mark read) also call `event.stopPropagation()`

### 4. Collapse-all extended to Worth a Look section

**Decision:** The "Collapse all" button now appears in both the Worth a Look and Low Priority section headers, following the same progressive-disclosure pattern: visible when ≥1 row is expanded, hidden when all are collapsed.

**Rationale:** With Worth a Look rows now expandable, users triaging multiple 3★ articles need the same batch-reset affordance that LP had. Featured and HP sections don't need collapse-all — they have few items (1–3 cards) where per-card toggles suffice.

### 5. Version + GitHub match production chrome

**Decision:** Show version text (e.g., `v0.4.2`) and a 15×15px GitHub SVG icon in the sidebar logo area, matching the existing `Nav.svelte` layout.

**Rationale:** The mockup should reflect actual production UI elements. Version text helps users verify their release. The GitHub icon is a subtle, discoverable link.

**Implementation notes:** The production app fetches version from `GET /api/version`. The GitHub SVG path is identical to the one in `Nav.svelte` line 40.

## Technical Details

All changes are to design/mockup files only:

| File | Changes |
|------|---------|
| `designs/mockup.html` | Universal expand/collapse on all 4 tiers, card-level click handler, WaL detail panels, WaL collapse-all, multi-select source toggle JS/CSS, version+GitHub in sidebar |
| `designs/proposal.md` | New "Refinement v4" section documenting universal expand/collapse and card-level click model, updated quick-reference table |
| `openspec/changes/refine-river-feed-interactions/*` | Proposal, design, and tasks updated to cover v4 requirements |
