## ADDED Requirements

### Requirement: Display entries as cards
The system SHALL display entries as cards showing: effective star rating, source name, time since discovery, title, and summary. Cards SHALL be ordered by effective stars descending, then discovered_at descending.

#### Scenario: Entry card display
- **WHEN** the feed view loads with entries
- **THEN** each entry shows as a card with star rating, source name, relative time, title, and 2-4 line summary

### Requirement: Filter by read status
The system SHALL provide tabs or buttons to filter entries: "Unread" (default) and "All". The unread count SHALL be visible.

#### Scenario: View unread entries
- **WHEN** user is on the feed view with 14 unread and 50 total entries
- **THEN** the view shows "Unread (14)" as active filter with 14 entry cards

#### Scenario: View all entries
- **WHEN** user switches to "All" filter
- **THEN** all entries are displayed regardless of read status

### Requirement: Filter by minimum stars
The system SHALL provide star filter buttons (All, 3+, 4+, 5) that filter entries by effective star rating.

#### Scenario: Filter 3+ stars
- **WHEN** user selects "3+" star filter
- **THEN** only entries with effective stars >= 3 are shown

#### Scenario: Combined filters
- **WHEN** user selects "Unread" and "4+" filters
- **THEN** only unread entries with effective stars >= 4 are shown

### Requirement: Open original article
The system SHALL provide a link on each entry card that opens the original article URL in a new browser tab.

#### Scenario: Open article link
- **WHEN** user taps the article link on an entry card
- **THEN** the original URL opens in a new browser tab

### Requirement: Auto-mark read on open
The system SHALL automatically mark an entry as read when the user opens the original article link.

#### Scenario: Opening article marks as read
- **WHEN** user opens the original article link for an unread entry
- **THEN** the entry is marked as read and the unread count decreases by 1

### Requirement: Manual read/unread toggle
The system SHALL allow the user to manually toggle the read status of any entry without opening the article.

#### Scenario: Mark as read without opening
- **WHEN** user clicks the read button on an unread entry
- **THEN** the entry is marked as read

#### Scenario: Mark as unread
- **WHEN** user clicks the unread button on a read entry
- **THEN** the entry is marked as unread

### Requirement: Star rating widget
The system SHALL display a clickable star rating widget (1-5) on each entry card. Tapping a star sets the user_stars value. Stars set by AI SHALL be visually distinct (e.g., dimmed) from user-set stars.

#### Scenario: User sets star rating
- **WHEN** user taps the 4th star on an entry with ai_stars 2
- **THEN** user_stars is set to 4, the card repositions in the sorted list, and the stars display as user-set (solid)

#### Scenario: AI stars display
- **WHEN** an entry has ai_stars 3 and no user_stars
- **THEN** 3 stars are displayed in a dimmed/italic style indicating AI-assigned

### Requirement: Mobile-responsive layout
The system SHALL render a mobile-friendly layout with cards that are easy to tap and read on phone screens. Card actions (read, rate, open, chat) SHALL have adequate tap targets.

#### Scenario: Phone viewport
- **WHEN** the feed view is opened on a 375px-wide screen
- **THEN** cards stack vertically, text is readable without zooming, and action buttons are at least 44px tap targets

### Requirement: Pending entries display
The system SHALL display entries with pending AI processing (null summary/stars) with a "processing" indicator instead of stars and summary.

#### Scenario: Pending entry display
- **WHEN** an entry has null summary and null ai_stars
- **THEN** the card shows a spinner or "Processing..." text in place of the summary and star rating
