# River Feed Redesign — Test Plan

> Open the app at `http://<host>:8090` and walk through each section.
> Check the box as you go. If something fails, note it inline.

## 1. Sidebar — Desktop (≥769px viewport)

- [ ] Sidebar visible on left, 220px wide, main content fills rest
- [ ] "KnowledgeHub" logo with amber "Hub" accent text
- [ ] Version number shown next to logo (e.g. `v0.x.x`)
- [ ] GitHub icon links to repo (opens new tab)
- [ ] **Feed** link highlighted when on feed page, shows unread badge count
- [ ] **Saved** link shows bookmarked count badge
- [ ] **Sources** link navigates to `/resources`, highlights when active
- [ ] **Settings** link navigates to `/settings`, highlights when active
- [ ] **Logout** button works

## 2. Sidebar — Mobile (≤768px viewport)

- [ ] Sidebar hidden, hamburger ☰ visible in topbar
- [ ] Tap hamburger → sidebar slides in from left
- [ ] Semi-transparent overlay covers main content
- [ ] Tap overlay → sidebar closes
- [ ] Tap any nav link → sidebar closes + navigates
- [ ] Theme toggle visible in mobile topbar (icon only, no label)

## 3. Theme Toggle

- [ ] Theme toggle button in desktop topbar (top-right area)
- [ ] Cycles: Light ☀️ → Dark 🌙 → System 🖥️ → Light...
- [ ] UI updates immediately on toggle
- [ ] Persists after page reload

## 4. Source Filter (in sidebar)

- [ ] "Filter by Source" header visible
- [ ] Each source shows: checkbox, colored 2-letter avatar, name, entry count
- [ ] Click a source → checkbox fills blue ✓, feed filters to that source
- [ ] Click a second source → both active, feed shows entries from both
- [ ] Click active source again → deselects it
- [ ] **"Clear" link** appears when any source selected
- [ ] Click "Clear" → all deselected, feed shows all entries
- [ ] "Clear" hidden when no sources selected

## 5. Source Chips in Topbar

- [ ] When sources selected, blue chips appear in feed topbar
- [ ] Each chip shows source name + ✕ button
- [ ] Click ✕ → that source deselected (chip disappears, sidebar updates)
- [ ] No chips when no sources selected

## 6. Tiered Card Rendering

### Featured (5★)
- [ ] Full-width card with **amber/gold left border**
- [ ] "★★★★★ Featured" label at top
- [ ] Large title (19px), summary, takeaways with bullet points
- [ ] Source avatar + name, relative time
- [ ] Action buttons: **Read Now** (blue), **Save**, **🤖 Chat**, **Mark read**
- [ ] **Expanded by default**

### High Priority (4★)
- [ ] Medium card with border, title, 2-line clamped summary
- [ ] Star rating widget, source avatar + name, time
- [ ] Side action buttons (↗ open, bookmark)
- [ ] **Expanded by default**

### Worth a Look (3★)
- [ ] Compact single-line row: ★★★, truncated title, source avatar, time
- [ ] **Collapsed by default**
- [ ] Click ▸ → detail panel slides down (animated) with full title, summary, actions
- [ ] Click ▾ → panel collapses

### Low Priority (1-2★)
- [ ] Minimal muted row at **~50% opacity**
- [ ] **Collapsed by default**
- [ ] Click ▸ → detail panel slides down, row opacity increases to ~85%
- [ ] Click ▾ → panel collapses, opacity returns to 50%

### Pending (0★ / processing)
- [ ] Shows in "Processing" section with spinner

## 7. Expand/Collapse

- [ ] ▸/▾ button on every card tier
- [ ] Featured/HP: ▾ button visible (expanded), click to collapse → hides detail
- [ ] WaL/LP: ▸ button visible (collapsed), click to expand → shows detail
- [ ] Clicking ▸/▾ does **not** open the article

## 8. Section Headers & Batch Controls

- [ ] Section headers: "FEATURED", "HIGH PRIORITY", "WORTH A LOOK", "LOW PRIORITY"
- [ ] WaL header shows "— click ▸ to expand" hint
- [ ] LP header shows "— click ▸ to expand" hint
- [ ] Empty sections are **hidden** (no header if no entries in that tier)
- [ ] **"Expand all"** button visible when at least one entry collapsed
- [ ] **"Collapse all"** button visible when at least one entry expanded
- [ ] Both buttons visible when section has mixed state
- [ ] "Expand all" expands all entries in that section
- [ ] "Collapse all" collapses all entries in that section

## 9. Card Click Behavior

- [ ] Click anywhere on card body → opens article in new tab
- [ ] Click ▸/▾ button → toggles expand only, does **not** open article
- [ ] Click star rating → sets rating, does **not** open article
- [ ] Click "Save" button → toggles bookmark, does **not** open article
- [ ] Click "Chat" button → opens chat panel, does **not** open article
- [ ] Entry marked as read after card-click open

## 10. Feed Topbar

- [ ] **Unread** tab (with count) — default active
- [ ] **Saved** tab (with count) — shows bookmarked entries
- [ ] **All** tab — shows all entries
- [ ] Star filter dropdown (★ All / ★ 3+ / ★ 4+ / ★ 5)
- [ ] **Mark read ▾** dropdown with time options
- [ ] Mark read works and undo banner appears

## 11. Existing Feature Preservation

- [ ] **Chat panel** opens from Featured/HP action buttons and WaL/LP detail panels
- [ ] **Link panel** opens from referenced links (🔗) in any tier
- [ ] **Realtime updates**: new entries appear in correct tier section
- [ ] **Quick-add FAB** (+ button bottom-right) works, new entry appears in correct tier
- [ ] **Undo banner** appears after marking entries as read, undo works
- [ ] **Bookmark toggle** works across all tiers
- [ ] **Star rating** works across all tiers (including re-tiering when stars change)

## 12. Mobile Responsive (≤768px)

- [ ] Hamburger visible, sidebar hidden
- [ ] Cards have reasonable padding on phone viewport
- [ ] Tap targets at least 44px
- [ ] Source filter works in mobile sidebar
- [ ] Full flow: hamburger → sidebar → filter → close → scroll → expand → open article

---

**Result**: ☐ Pass / ☐ Fail — Notes:
