## Context

The River redesign mockup (v2) established the core tiered-card feed layout, sidebar source filtering, and expandable low-priority rows. Review feedback identified four interaction gaps: single-select source filters are too restrictive, row-level click handling conflicts with article navigation, there's no batch collapse, and the mockup is missing production chrome (version, GitHub link).

V3 addressed all four gaps. V4 extends the design further: expand/collapse is now universal across all card tiers, and the entire card is a click target for opening articles. V5 adds section-level "Expand all" and "Collapse all" batch controls to every section (Featured, High Priority, Worth a Look, Low Priority).

## Goals / Non-Goals

**Goals:**
- Multi-select source filter toggles in the sidebar with clear visual state *(v3)*
- Conflict-free click targets: dedicated ▸/▾ button for expand/collapse, card-level click for opening articles *(v3+v4)*
- Expand/collapse on ALL four card tiers: Featured, High Priority, Worth a Look, Low Priority *(v4)*
- Card-level click opens article — the entire card is a tap/click target *(v4)*
- Section-level "Expand all" and "Collapse all" batch controls on every section *(v5 — replaces v4's WaL/LP-only collapse-all)*
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

### 4. Section-level batch controls on every section

**Decision:** Every section header (Featured, High Priority, Worth a Look, Low Priority) now has both an "Expand all" and a "Collapse all" button. Both buttons use progressive disclosure: "Expand all" is visible when ≥1 card in the section is collapsed; "Collapse all" is visible when ≥1 card is expanded. When all cards share the same state, only the opposite action is shown.

**Rationale:** The v4 exception for Featured/HP ("few items, per-card toggles suffice") underestimated user expectations for consistency. Users explicitly expect batch controls on every section. A uniform section header pattern is easier to learn and reduces cognitive load.

**Per-section details:**
- **Featured**: Gets a new `<div class="sl" id="featHeader">` section header above the card. The card retains its internal "★★★★★ Featured" label.
- **High Priority**: Existing `<div class="sl">` header updated with `id="hpHeader"` and both batch buttons.
- **Worth a Look**: Existing header updated to include "Expand all" alongside the existing "Collapse all".
- **Low Priority**: Same update as Worth a Look — "Expand all" added.

**Implementation:**
- New `expandSection(prefix)` function handles all 4 section types
- `collapseSection(prefix)` extended from 'wal'/'lp' to also handle 'feat'/'hp'
- `updateSectionBatchButtons()` (renamed from `updateCollapseAllButtons()`) manages visibility of both buttons across all 4 sections
- CSS class `.section-btn` replaces `.collapse-all` for generic section-level button styling

### 5. Version + GitHub match production chrome

**Decision:** Show version text (e.g., `v0.4.2`) and a 15×15px GitHub SVG icon in the sidebar logo area, matching the existing `Nav.svelte` layout.

**Rationale:** The mockup should reflect actual production UI elements. Version text helps users verify their release. The GitHub icon is a subtle, discoverable link.

**Implementation notes:** The production app fetches version from `GET /api/version`. The GitHub SVG path is identical to the one in `Nav.svelte` line 40.

## Technical Details

All changes are to design/mockup files only:

| File | Changes |
|------|---------|
| `designs/mockup.html` | Universal expand/collapse on all 4 tiers, card-level click handler, WaL detail panels, section-level batch Expand all / Collapse all on all 4 sections, multi-select source toggle JS/CSS, version+GitHub in sidebar |
| `designs/proposal.md` | New "Refinement v5" section documenting section-level batch controls on every section, updated quick-reference table |
| `openspec/changes/refine-river-feed-interactions/*` | Proposal, design, and tasks updated to cover v5 requirements |
