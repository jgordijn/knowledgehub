## ADDED Requirements

### Requirement: Featured card rendering (5★)
The system SHALL render entries with effective stars of 5 as a full-width "Featured" card with an amber/gold left border accent, large title, summary text, takeaways list, source with colored avatar, relative time, and full action buttons (Read Now, Save, Chat). The card SHALL default to expanded state.

#### Scenario: Featured card layout
- **WHEN** an entry has effective stars of 5
- **THEN** it renders as a full-width card with amber left border, "★★★★★ Featured" label, large title, summary, takeaways, source avatar, time, and action buttons

#### Scenario: Featured card with pending processing
- **WHEN** a 5★ entry has no summary yet (processing)
- **THEN** the card shows the Featured layout with a processing spinner in place of summary and takeaways

### Requirement: High Priority card rendering (4★)
The system SHALL render entries with effective stars of 4 as a medium card with title, 2-line clamped summary, star rating, source with avatar, time, and compact side action buttons (Read, Save). The card SHALL default to expanded state.

#### Scenario: High Priority card layout
- **WHEN** an entry has effective stars of 4
- **THEN** it renders as a medium card with title, summary (max 2 lines), stars, source, time, and side action buttons

### Requirement: Worth a Look row rendering (3★)
The system SHALL render entries with effective stars of 3 as a compact single-line row showing star rating, truncated title, source avatar, relative time, and an expand button. The row SHALL default to collapsed state.

#### Scenario: Worth a Look collapsed row
- **WHEN** an entry has effective stars of 3 and is in collapsed state
- **THEN** it renders as a single-line row with "★★★", truncated title, source avatar, time, and a ▸ expand button

#### Scenario: Worth a Look expanded detail
- **WHEN** a 3★ entry is expanded
- **THEN** a detail panel appears below the row showing full title, summary, source name, time, and action buttons (Read, Save, Chat)

### Requirement: Low Priority row rendering (1–2★)
The system SHALL render entries with effective stars of 1 or 2 as a minimal muted row at 50% opacity showing star rating, truncated title, source avatar, time, and an expand button. The row SHALL default to collapsed state.

#### Scenario: Low Priority collapsed row
- **WHEN** an entry has effective stars of 2 and is in collapsed state
- **THEN** it renders as a muted (50% opacity) single-line row with "★★", truncated title, source avatar, time, and a ▸ expand button

#### Scenario: Low Priority expanded detail
- **WHEN** a 1–2★ entry is expanded
- **THEN** a detail panel appears below the row showing full title, summary, source, time, and action buttons (Read, Save, Mark read), with the row header lifting to ~85% opacity

### Requirement: Expand/collapse toggle on every card tier
The system SHALL provide a dedicated ▸/▾ button on every card tier. Featured and HP cards show ▾ (collapse) by default; WaL and LP rows show ▸ (expand) by default. Clicking the button toggles the detail visibility without opening the article.

#### Scenario: Collapse a Featured card
- **WHEN** the user clicks the ▾ button on a Featured card
- **THEN** the summary, takeaways, and action buttons are hidden, leaving only the Featured label and title visible, and the button changes to ▸

#### Scenario: Expand a collapsed Featured card
- **WHEN** the user clicks the ▸ button on a collapsed Featured card
- **THEN** the summary, takeaways, and action buttons become visible again, and the button changes to ▾

#### Scenario: Collapse a High Priority card
- **WHEN** the user clicks the ▾ button on an HP card
- **THEN** the summary and side action buttons are hidden, leaving only the title, stars, source, and time visible

#### Scenario: Expand a Worth a Look row
- **WHEN** the user clicks the ▸ button on a collapsed WaL row
- **THEN** a detail panel slides open below the row with summary, metadata, and action buttons

#### Scenario: Expand a Low Priority row
- **WHEN** the user clicks the ▸ button on a collapsed LP row
- **THEN** a detail panel slides open below the row with summary, metadata, and action buttons, and the row opacity increases

### Requirement: Card-level click opens article
The system SHALL open the article in a new browser tab when the user clicks anywhere on a card, EXCEPT when clicking on interactive elements (buttons, links, inputs, star rating widget). The system SHALL also mark the entry as read when opening via card click.

#### Scenario: Click card body opens article
- **WHEN** the user clicks on the card body (not on a button or link)
- **THEN** the article URL opens in a new browser tab and the entry is marked as read

#### Scenario: Click expand button does not open article
- **WHEN** the user clicks the ▸/▾ expand button
- **THEN** the card expands/collapses but the article does NOT open

#### Scenario: Click action button does not open article
- **WHEN** the user clicks the "Save" button
- **THEN** the save action is performed but the article does NOT open

#### Scenario: Click star rating does not open article
- **WHEN** the user clicks on a star in the star rating widget
- **THEN** the star rating is set but the article does NOT open

### Requirement: Entries grouped by tier
The system SHALL group entries into four tier sections in the feed: Featured (5★), High Priority (4★), Worth a Look (3★), and Low Priority (1–2★). Sections with no entries SHALL be hidden. Within each section, entries SHALL be ordered by publication/discovery time descending.

#### Scenario: Tiered grouping
- **WHEN** the feed contains entries with stars 5, 4, 3, and 2
- **THEN** they are displayed in four sections: Featured at top, then High Priority, then Worth a Look, then Low Priority

#### Scenario: Empty section hidden
- **WHEN** there are no entries with effective stars of 5
- **THEN** the "Featured" section and its header are not rendered

#### Scenario: Entries within section ordered by time
- **WHEN** the High Priority section contains entries from 2h ago and 6h ago
- **THEN** the 2h entry appears before the 6h entry within the section
