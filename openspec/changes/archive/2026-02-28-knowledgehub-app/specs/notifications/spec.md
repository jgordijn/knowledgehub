## ADDED Requirements

### Requirement: Browser notification for 5-star entries
The system SHALL send a browser notification when a new entry with effective stars of 5 is created, if the user has the application open and has granted notification permission.

#### Scenario: 5-star entry notification
- **WHEN** the engine creates a new entry with ai_stars 5 and the user has the app open in a browser tab
- **THEN** a browser notification appears with the title "KnowledgeHub ★★★★★" and the entry title as the body

#### Scenario: Non-5-star entry
- **WHEN** the engine creates a new entry with ai_stars 3
- **THEN** no browser notification is sent

### Requirement: Realtime feed updates
The system SHALL update the feed view in realtime when new entries are created, without requiring a page refresh. New entries SHALL appear at the appropriate position based on sort order.

#### Scenario: New entry appears in feed
- **WHEN** the user has the feed view open and a new entry is created by the engine
- **THEN** the entry card appears in the feed at the correct sorted position

### Requirement: Notification permission request
The system SHALL request browser notification permission on first visit. The user can grant or deny. If denied, realtime feed updates still work but no OS-level notifications are shown.

#### Scenario: Permission granted
- **WHEN** user grants notification permission
- **THEN** 5-star entries trigger OS-level browser notifications

#### Scenario: Permission denied
- **WHEN** user denies notification permission
- **THEN** no OS-level notifications are shown, but new entries still appear in the feed in realtime
