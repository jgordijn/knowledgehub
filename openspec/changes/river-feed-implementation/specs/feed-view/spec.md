## MODIFIED Requirements

### Requirement: Display entries as cards
The system SHALL display entries using a tiered card system based on effective star rating. Entries with 5 stars SHALL render as Featured cards, 4 stars as High Priority cards, 3 stars as Worth a Look compact rows, and 1–2 stars as Low Priority muted rows. Entries SHALL be grouped by tier (Featured first, then HP, WaL, LP) with each tier section ordered by publication/discovery time descending. When an entry has takeaways, they SHALL be rendered as a compact bullet list below the summary (visible in Featured and HP expanded cards, and in WaL/LP detail panels when expanded).

#### Scenario: Tiered entry display
- **WHEN** the feed view loads with entries of varying star ratings
- **THEN** entries are grouped into tier sections (Featured, High Priority, Worth a Look, Low Priority) with appropriate visual rendering per tier

#### Scenario: Entry card with takeaways
- **WHEN** an entry has a non-empty takeaways array and is in a Featured or HP tier
- **THEN** the expanded card displays the takeaways as a bulleted list below the summary

#### Scenario: Entry card without takeaways
- **WHEN** an entry has null or empty takeaways
- **THEN** no takeaway section is rendered

### Requirement: Filter by read status
The system SHALL provide tabs in the topbar to filter entries: "Unread" (default, with unread count), "Saved" (with bookmarked count), and "All". The tabs SHALL be visually styled as a segmented control.

#### Scenario: View unread entries
- **WHEN** user is on the feed view with 14 unread entries
- **THEN** the view shows "Unread 14" as active tab with 14 entries displayed in tier groups

#### Scenario: View saved entries
- **WHEN** user switches to the "Saved" tab
- **THEN** only bookmarked entries are displayed in tier groups

#### Scenario: View all entries
- **WHEN** user switches to "All" tab
- **THEN** all entries are displayed regardless of read status

### Requirement: Open original article
The system SHALL open the original article URL in a new browser tab when the user clicks anywhere on a card (except on interactive elements such as buttons, links, inputs, or the star rating widget). Title text SHALL remain wrapped in an `<a>` tag for accessibility.

#### Scenario: Click card opens article
- **WHEN** user clicks on the card body of any entry
- **THEN** the original URL opens in a new browser tab

#### Scenario: Click interactive element does not open article
- **WHEN** user clicks on a button, link, or star rating within a card
- **THEN** the article does NOT open; the interactive element handles its own event

### Requirement: Mobile-responsive layout
The system SHALL render a mobile-friendly layout on viewports ≤768px. The sidebar becomes a slide-out drawer. Cards SHALL adapt their padding and font sizes for phone screens. Action buttons SHALL have adequate tap targets (at least 44px).

#### Scenario: Phone viewport
- **WHEN** the feed view is opened on a 375px-wide screen
- **THEN** the sidebar is hidden (accessible via hamburger), cards render with reduced padding, and action buttons are at least 44px tap targets
