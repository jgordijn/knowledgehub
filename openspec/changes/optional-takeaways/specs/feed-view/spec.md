## MODIFIED Requirements

### Requirement: Display entries as cards
The system SHALL display entries as cards showing: effective star rating, source name, time since discovery, title, summary, and optional takeaways. When an entry has takeaways, they SHALL be rendered as a compact bullet list below the summary. Cards SHALL be ordered by effective stars descending, then discovered_at descending.

#### Scenario: Entry card display
- **WHEN** the feed view loads with entries
- **THEN** each entry shows as a card with star rating, source name, relative time, title, and 2-4 line summary

#### Scenario: Entry card with takeaways
- **WHEN** an entry has a non-empty takeaways array
- **THEN** the card displays the takeaways as a bulleted list below the summary in a compact style

#### Scenario: Entry card without takeaways
- **WHEN** an entry has null or empty takeaways
- **THEN** the card displays only the summary with no takeaway section
