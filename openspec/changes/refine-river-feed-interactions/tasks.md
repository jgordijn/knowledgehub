## 1. Mockup: Multi-Select Source Toggles

- [x] 1.1 Replace single-select `filterSource()` JS with multi-select `toggleSource()` — each sidebar source item toggles independently
- [x] 1.2 Add checkbox indicator (`.toggle-check`) to each source list item — blue filled ✓ when active, empty border when inactive
- [x] 1.3 Remove "All Sources" list item — unfiltered state is now "nothing selected" (implicit all)
- [x] 1.4 Add "Clear" link in the source section header — calls `clearAllSources()`, hidden when no filters active
- [x] 1.5 Update topbar source indicator from single chip to multi-chip container — one `active-src-tag` per active source, each with ✕ dismiss via `deselectSource()`
- [x] 1.6 Add `syncSourceTags()` function that rebuilds topbar chips and toggles "Clear" link visibility whenever selection changes

## 2. Mockup: Universal Expand/Collapse on All Card Tiers

- [x] 2.1 Add ▸/▾ `.expand-btn` to Featured card — positioned in header row, defaults to ▾ (expanded)
- [x] 2.2 Add ▸/▾ `.expand-btn` to High Priority cards — positioned in side-btns area, defaults to ▾ (expanded)
- [x] 2.3 Add ▸/▾ `.expand-btn` to Worth a Look rows — positioned at right edge, defaults to ▸ (collapsed)
- [x] 2.4 Keep existing ▸/▾ `.expand-btn` on Low Priority rows (unchanged from v3)
- [x] 2.5 Wrap collapsible content in Featured card in `.card-detail` div with id `feat1-detail`
- [x] 2.6 Wrap collapsible content in HP cards in `.card-detail` divs with ids `hp1-detail`, `hp2-detail`
- [x] 2.7 Add `.wal-detail` panels below Worth a Look rows with summary, metadata, and action buttons
- [x] 2.8 Add CSS for `.feat.collapsed` (hides `.card-detail`), `.cm.collapsed` (hides `.card-detail` + `.side-acts`), `.cs.expanded`, `.wal-detail` / `.wal-detail.open`
- [x] 2.9 Implement unified `toggleCard(id)` JS function that handles all four tiers based on card class detection
- [x] 2.10 All card titles remain `<a>` links for accessibility; action buttons have `onclick="event.stopPropagation()"`

## 3. Mockup: Card-Level Click Opens Article

- [x] 3.1 Add `onclick="openArticle(event, url)"` to all card containers (`.feat`, `.cm`, `.cs`, `.mn`)
- [x] 3.2 Add `onclick="openArticle(event, url)"` to detail panels (`.wal-detail`, `.mn-detail`) so expanded content is also clickable
- [x] 3.3 Implement `openArticle(event, url)` — walks DOM from click target, bails if interactive element, otherwise opens article
- [x] 3.4 Add `cursor:pointer` to all card types via CSS
- [x] 3.5 Add visual click-flash feedback (blue outline animation) for mockup demo
- [x] 3.6 All expand buttons use `event.stopPropagation()` to prevent triggering card navigation

## 4. Mockup: Section-Level Collapse All

- [x] 4.1 Add "Collapse all" button to Worth a Look section header (`#walCollapseBtn`), hidden by default
- [x] 4.2 Keep "Collapse all" button in Low Priority section header (`#lpCollapseBtn`)
- [x] 4.3 Implement `collapseSection(prefix)` that collapses all expanded cards in a section ('wal' or 'lp')
- [x] 4.4 `updateCollapseAllButtons()` checks both WaL and LP sections, shows/hides buttons accordingly
- [x] 4.5 `updateCollapseAllButtons()` called after every `toggleCard()` and `collapseSection()`, and on page load

## 5. Mockup: Version Text + GitHub Icon

- [x] 5.1 Add version text (`v0.4.2`) next to "KnowledgeHub" in sidebar `.logo` area
- [x] 5.2 Add GitHub SVG icon as an `<a>` link to `https://github.com/jgordijn/knowledgehub` — 15×15px, muted color, hover brightens
- [x] 5.3 SVG path matches the one in `ui/src/lib/components/Nav.svelte` line 40

## 6. Proposal Documentation

- [x] 6.1 Add "Refinement v4 — Universal Expand/Collapse Across All Card Tiers" section to `designs/proposal.md`
- [x] 6.2 Document universal expand/collapse decision with per-tier default states table
- [x] 6.3 Document card-level click-opens-article interaction model with click-target table
- [x] 6.4 Document Worth a Look collapse-all extension
- [x] 6.5 Add v4 summary table showing before/after for each element
- [x] 6.6 Update "Design Clarifications — Quick Reference" table — expanded from 6 to 8 entries covering all-tier expand/collapse and card-click behavior
- [x] 6.7 Update tiered card prominence description (section 1) with v4 expand/collapse annotations
