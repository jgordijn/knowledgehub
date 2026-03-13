## 1. Sidebar Component & App Shell

- [x] 1.1 Create `Sidebar.svelte` component with logo area (KnowledgeHub text, version from `/api/version`, GitHub SVG icon link), nav links section, source filter section placeholder, and footer
- [x] 1.2 Add nav links to sidebar: Feed (with unread badge), Saved (with bookmarked badge), Sources, Settings — highlight active page based on current route
- [x] 1.3 Update `+layout.svelte` to use sidebar layout: flex row with `Sidebar` (fixed 220px) + main content area, replacing the current `Nav` top bar
- [x] 1.4 Add mobile hamburger button in a minimal topbar (visible ≤768px), sidebar slides from left (`position: fixed; left: -220px` → `left: 0`), with semi-transparent overlay backdrop
- [x] 1.5 Close mobile sidebar when overlay is clicked or a nav link is tapped
- [x] 1.6 Move theme toggle into the topbar area — keep existing light/dark/system logic from `theme.ts`

## 2. Multi-Select Source Filter in Sidebar

- [x] 2.1 Add source filter section to `Sidebar.svelte`: "Filter by Source" header with a "Clear" link (visible when any source selected), source list items with checkbox indicator, colored 2-letter avatar, name, and entry count
- [x] 2.2 Replace `resourceFilter: string` state in `+page.svelte` with `selectedSources: Set<string>` — pass as prop to Sidebar along with `onToggleSource` and `onClearSources` callbacks
- [x] 2.3 Implement client-side source filtering in `filteredEntries` derived: when `selectedSources` is non-empty, filter entries to only include those whose `resource` ID is in the set
- [x] 2.4 Add active source chips in the topbar area of the feed page: one blue chip per active source with ✕ dismiss button, hidden when no sources selected
- [x] 2.5 Remove the old collapsible single-select source pill bar from `+page.svelte`
- [x] 2.6 Compute per-source entry counts from the loaded entries and pass to Sidebar for display

## 3. Tiered Card Rendering in EntryCard

- [x] 3.1 Add `tier` derived state to `EntryCard.svelte` based on `effectiveStars`: 5→'featured', 4→'hp', 3→'wal', 1-2→'lp'
- [x] 3.2 Accept `expanded` prop and `onToggle` callback from parent page for controlling expand/collapse state
- [x] 3.3 Implement Featured (5★) card layout: full-width card with amber left border, "★★★★★ Featured" label, ▸/▾ expand button in header, large title, summary, takeaways, source avatar + name, time, action buttons (Read Now, Save, Chat)
- [x] 3.4 Implement High Priority (4★) card layout: medium card with title, 2-line clamped summary, star rating, source avatar + name, time, side buttons area with ▸/▾ expand button + Read/Save buttons
- [x] 3.5 Implement Worth a Look (3★) row layout: compact single-line row with stars, truncated title, source avatar, time, ▸/▾ expand button — when expanded, show detail panel below with full title, summary, source, time, action buttons
- [x] 3.6 Implement Low Priority (1–2★) row layout: minimal muted row at 50% opacity, same structure as WaL — when expanded, detail panel appears and row opacity lifts to ~85%
- [x] 3.7 Preserve existing card-level click handler (`handleCardClick`) — it already walks DOM for interactive elements and opens article; ensure it works with all four tier layouts
- [x] 3.8 Preserve fragment content rendering, referenced links, pending/processing state, bookmark toggle, read toggle, chat button, and link panel button across all tiers
- [x] 3.9 Add slideDown animation for WaL and LP detail panels (CSS keyframes: opacity 0 + translateY(-4px) → opacity 1 + translateY(0))

## 4. Entry Grouping & Section Headers in Feed Page

- [x] 4.1 Add tier grouping logic in `+page.svelte`: split `filteredEntries` into four derived arrays — `featuredEntries` (5★), `hpEntries` (4★), `walEntries` (3★), `lpEntries` (1-2★), each sorted by time descending
- [x] 4.2 Add `expandedSet: Set<string>` state in `+page.svelte` — initialize Featured and HP entry IDs as expanded, WaL and LP as collapsed. Re-initialize defaults when entries change (filter change, realtime update).
- [x] 4.3 Render section headers for each non-empty tier group: "Featured", "High Priority", "Worth a Look — click ▸ to expand", "Low Priority — click ▸ to expand"
- [x] 4.4 Render `EntryCard` components within each section, passing `expanded` (derived from `expandedSet.has(entry.id)`) and `onToggle` callback
- [x] 4.5 Replace the old flat `<div class="grid gap-3">` entry list with the tiered section layout

## 5. Section Batch Controls

- [x] 5.1 Add "Expand all" button to each section header — visible when at least one entry in the section is collapsed. Clicking adds all section entry IDs to `expandedSet`.
- [x] 5.2 Add "Collapse all" button to each section header — visible when at least one entry in the section is expanded. Clicking removes all section entry IDs from `expandedSet`.
- [x] 5.3 Implement progressive disclosure: show both buttons when section has mixed expand/collapse state; show only the opposite button when all entries share the same state

## 6. Topbar Redesign

- [x] 6.1 Redesign the topbar in `+page.svelte`: Unread/Saved/All tab buttons (styled as segmented control), active source chips area, star filter dropdown, spacer, mark-read dropdown button
- [x] 6.2 Update read filter tabs: rename "Read Later" → remove (now in sidebar nav as "Saved"), keep "Unread (count)" / "All" / add "Saved" tab
- [x] 6.3 Preserve undo banner, mark-as-read dropdown, and quick-add FAB functionality

## 7. Styling & Responsive Polish

- [x] 7.1 Apply dark/light theme styling for sidebar using Tailwind dark: variants — sidebar background, border, text colors matching mockup's color system
- [x] 7.2 Style tiered cards with Tailwind: Featured card amber border accent, HP card border, WaL/LP row bottom borders, muted opacity for LP
- [x] 7.3 Style expand/collapse buttons: small bordered button with ▸/▾ text, hover state
- [x] 7.4 Style source filter items: checkbox indicator (14×14px border square, blue filled with ✓ when active), colored avatars, active state background
- [x] 7.5 Style active source chips in topbar: blue background, rounded pill, ✕ dismiss button
- [x] 7.6 Style section headers: uppercase label, small font, batch buttons as subtle bordered pills
- [x] 7.7 Mobile responsive adjustments: reduced card padding, smaller font sizes on ≤768px and ≤400px, hamburger button styling
- [x] 7.8 Add click-flash feedback animation (optional — blue outline keyframe on card click) for visual tap feedback

## 8. Integration & Cleanup

- [x] 8.1 Verify chat panel still opens correctly from all card tiers (Featured/HP action button, WaL/LP detail panel button)
- [x] 8.2 Verify link panel still opens correctly from referenced links across all tiers
- [x] 8.3 Verify realtime updates (create/update/delete) work with tiered grouping — new entries appear in correct section, updates move entries between tiers if stars change
- [x] 8.4 Verify quick-add FAB still works and new entries appear in the correct tier
- [x] 8.5 Verify undo mark-read banner still works with the new layout
- [x] 8.6 Remove or deprecate unused code from old layout (old source filter pills, old flat grid rendering)
- [x] 8.7 Test mobile layout end-to-end: hamburger → sidebar → source filter → close → scroll feed → expand card → open article
