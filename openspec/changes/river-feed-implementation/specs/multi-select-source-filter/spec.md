## ADDED Requirements

### Requirement: Multi-select source filter toggles
The system SHALL display a source filter list in the sidebar showing all active sources. Each source SHALL be an independent toggle — clicking a source activates or deactivates it without affecting other sources. Each source item SHALL show a checkbox indicator (filled blue ✓ when active, empty border when inactive), a colored 2-letter avatar, the source name, and an entry count.

#### Scenario: Toggle source on
- **WHEN** the user clicks on "Rust Blog" in the source list (currently inactive)
- **THEN** "Rust Blog" becomes active (checkbox fills blue with ✓) and the feed filters to show only entries from active sources

#### Scenario: Toggle source off
- **WHEN** the user clicks on "Rust Blog" in the source list (currently active)
- **THEN** "Rust Blog" becomes inactive (checkbox shows empty border) and the feed updates to remove that source filter

#### Scenario: Multiple sources selected
- **WHEN** the user activates "Rust Blog" and "InfoQ"
- **THEN** the feed shows only entries from Rust Blog and InfoQ

### Requirement: No selection means all sources shown
The system SHALL show entries from all sources when no sources are selected in the filter. This is the default state on page load.

#### Scenario: Default unfiltered state
- **WHEN** the feed loads with no source filters active
- **THEN** entries from all active sources are displayed

#### Scenario: Deselecting last source shows all
- **WHEN** the user deselects the only remaining active source filter
- **THEN** the feed returns to showing entries from all sources

### Requirement: Clear all sources button
The system SHALL display a "Clear" link in the source filter section header when at least one source is selected. Clicking it SHALL deselect all sources, returning to the unfiltered state. The "Clear" link SHALL be hidden when no sources are selected.

#### Scenario: Clear link visible when filtering
- **WHEN** 2 sources are selected in the filter
- **THEN** a "Clear" link is visible in the source section header

#### Scenario: Clear link clears all selections
- **WHEN** the user clicks "Clear" with 3 active source filters
- **THEN** all source filters are deselected and the feed shows all sources

#### Scenario: Clear link hidden when no filters
- **WHEN** no sources are selected
- **THEN** the "Clear" link is not visible

### Requirement: Active source chips in topbar
The system SHALL display one blue chip per active source in the topbar when source filters are active. Each chip SHALL show the source name and an ✕ dismiss button. Clicking ✕ on a chip SHALL deselect that source. The chip container SHALL be hidden when no sources are selected.

#### Scenario: Source chips displayed
- **WHEN** "Rust Blog" and "InfoQ" are active source filters
- **THEN** the topbar shows two blue chips: "Rust Blog ✕" and "InfoQ ✕"

#### Scenario: Dismiss source chip
- **WHEN** the user clicks ✕ on the "Rust Blog" chip
- **THEN** "Rust Blog" is deselected in the sidebar and the chip disappears from the topbar

#### Scenario: No chips when unfiltered
- **WHEN** no sources are selected
- **THEN** no source chips appear in the topbar

### Requirement: Source filter applied client-side
The system SHALL apply source filtering on the client side by filtering the already-loaded entry list. The system SHALL NOT make additional server requests when toggling source filters.

#### Scenario: Instant filter toggle
- **WHEN** the user toggles a source filter
- **THEN** the feed updates immediately without a loading spinner or server request
