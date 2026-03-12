## 1. Mockup: Multi-Select Source Toggles

- [x] 1.1 Replace single-select `filterSource()` JS with multi-select `toggleSource()` — each sidebar source item toggles independently
- [x] 1.2 Add checkbox indicator (`.toggle-check`) to each source list item — blue filled ✓ when active, empty border when inactive
- [x] 1.3 Remove "All Sources" list item — unfiltered state is now "nothing selected" (implicit all)
- [x] 1.4 Add "Clear" link in the source section header — calls `clearAllSources()`, hidden when no filters active
- [x] 1.5 Update topbar source indicator from single chip to multi-chip container — one `active-src-tag` per active source, each with ✕ dismiss via `deselectSource()`
- [x] 1.6 Add `syncSourceTags()` function that rebuilds topbar chips and toggles "Clear" link visibility whenever selection changes

## 2. Mockup: Separated Open-Article vs Expand/Collapse

- [x] 2.1 Make all card titles `<a>` links — Featured, High Priority, Worth a Look, Low Priority (collapsed row and detail panel)
- [x] 2.2 Replace row-level `onclick` on `.mn` low-priority rows with a dedicated `.expand-btn` `<button>` (▸/▾) at the right edge
- [x] 2.3 Add `event.stopPropagation()` on title links in collapsed low-priority rows to prevent parent click interference
- [x] 2.4 Remove `cursor:pointer` from `.mn` row CSS — the row itself is no longer clickable

## 3. Mockup: Collapse All Control

- [x] 3.1 Add "Collapse all" `<button>` to the Low Priority section header (`.sl`)
- [x] 3.2 Button is hidden by default (CSS `display:none`), shown via JS when ≥1 `.mn.expanded` exists
- [x] 3.3 `collapseAllLowPriority()` iterates all `.mn.expanded` rows and collapses them
- [x] 3.4 `updateCollapseAllButton()` called after every expand/collapse toggle and on page load

## 4. Mockup: Version Text + GitHub Icon

- [x] 4.1 Add version text (`v0.4.2`) next to "KnowledgeHub" in sidebar `.logo` area
- [x] 4.2 Add GitHub SVG icon as an `<a>` link to `https://github.com/jgordijn/knowledgehub` — 15×15px, muted color, hover brightens
- [x] 4.3 SVG path matches the one in `ui/src/lib/components/Nav.svelte` line 40

## 5. Proposal Documentation

- [x] 5.1 Add "Refinement v3 — Interaction Clarity & Multi-Select Sources" section to `designs/proposal.md`
- [x] 5.2 Document multi-select source toggle decision with problem/resolution/rationale
- [x] 5.3 Document separated click targets decision
- [x] 5.4 Document collapse-all control decision
- [x] 5.5 Document version + GitHub link decision
- [x] 5.6 Add v3 summary table showing before/after for each element
- [x] 5.7 Update "Design Clarifications — Quick Reference" table with new entries (#5 open vs expand, #6 collapse all)
