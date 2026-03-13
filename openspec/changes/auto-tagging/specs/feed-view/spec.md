## ADDED Requirements

### Requirement: Display tags on entry cards
The system SHALL display an entry's tags as small chips on the entry card, below the summary/takeaways area. Tags are rendered in a compact, muted style that does not dominate the card layout. Entries with null or empty tags show no tag section.

#### Scenario: Entry with tags
- **WHEN** an entry has tags ["go", "concurrency", "performance"]
- **THEN** three small tag chips are displayed on the card

#### Scenario: Entry without tags
- **WHEN** an entry has null or empty tags
- **THEN** no tag section is rendered on the card

### Requirement: Filter by tags
The system SHALL provide a tag filter section on the feed view (collapsible, similar to the resource filter) showing the most frequently used tags across visible entries. The user can select one or more tags. When tags are selected, only entries containing at least one of the selected tags are shown (OR logic). The tag filter composes with all other active filters.

#### Scenario: Select a single tag
- **WHEN** user selects the "go" tag from the tag filter
- **THEN** only entries that have "go" in their tags array are shown

#### Scenario: Select multiple tags
- **WHEN** user selects "go" and "rust" from the tag filter
- **THEN** entries that have "go" OR "rust" (or both) in their tags are shown

#### Scenario: Tag frequency ordering
- **WHEN** the tag filter section is expanded
- **THEN** tags are ordered by frequency (most common first), showing up to 20 tags

#### Scenario: Tag filter combined with search
- **WHEN** user has a search query "memory" and selects tag "performance"
- **THEN** only entries matching "memory" in title/summary/resource AND having "performance" in tags are shown
