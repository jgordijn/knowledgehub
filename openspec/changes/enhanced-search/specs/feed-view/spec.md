## ADDED Requirements

### Requirement: Search entries by text
The system SHALL provide a search bar on the feed view that filters entries by matching the search query against title, summary, and resource name. The search SHALL be case-insensitive and match substrings. The search SHALL compose with all other active filters (read status, stars, resource). When the search field is cleared, the full (filtered) entry list SHALL be restored.

#### Scenario: Search by article title
- **WHEN** user types "CRDT" into the search bar with readFilter set to "all"
- **THEN** only entries whose title, summary, or resource name contains "CRDT" (case-insensitive) are displayed

#### Scenario: Search by resource name
- **WHEN** user types "Yegge" into the search bar
- **THEN** entries from resources whose name contains "Yegge" are displayed, regardless of the entry's title or summary content

#### Scenario: Search combined with star filter
- **WHEN** user types "Go" into the search bar and star filter is set to 4+
- **THEN** only entries matching "Go" with effective stars >= 4 are displayed

#### Scenario: Empty search results
- **WHEN** user types a query that matches no entries
- **THEN** the view shows an empty state message indicating no entries match the search

#### Scenario: Clear search
- **WHEN** user clears the search bar (empty string)
- **THEN** the search filter is removed and all entries matching other active filters are shown

### Requirement: Debounced search
The system SHALL debounce search input by 300ms to avoid excessive API calls while typing.

#### Scenario: Rapid typing
- **WHEN** user types "dist" quickly (each character within 300ms)
- **THEN** only one API call is made after typing stops, filtering for "dist"

## MODIFIED Requirements

### Requirement: Filter by resource (MODIFIED)
The system SHALL provide a resource filter that allows selecting multiple resources simultaneously. When one or more resources are selected, only entries from those resources are shown. The selected resources SHALL be displayed as dismissible chips. Deselecting all resources restores the "all sources" view.

#### Scenario: Select multiple resources
- **WHEN** user selects "Go Blog" and "Hacker News" from the resource filter
- **THEN** only entries from those two sources are displayed

#### Scenario: Dismiss a resource chip
- **WHEN** user clicks the dismiss button on the "Go Blog" chip while "Go Blog" and "Hacker News" are selected
- **THEN** "Go Blog" is deselected and only "Hacker News" entries are shown

#### Scenario: Clear all resource filters
- **WHEN** user deselects all resource chips or clicks "All"
- **THEN** entries from all active resources are shown
