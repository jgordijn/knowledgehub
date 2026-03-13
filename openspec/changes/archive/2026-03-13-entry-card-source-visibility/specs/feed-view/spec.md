## MODIFIED Requirements

### Requirement: Display entries as cards
The system SHALL display entries as cards showing: effective star rating, title, source name with colored avatar, relative time, summary, and optional takeaways. The source name with its colored avatar and relative time SHALL be displayed directly below the title on all card tiers, visible in both collapsed and expanded states. When an entry has takeaways, they SHALL be rendered as a compact bullet list below the summary. Cards SHALL be ordered by effective stars descending, then discovered_at descending.

#### Scenario: Entry card display
- **WHEN** the feed view loads with entries
- **THEN** each entry shows as a card with star rating, title, source name with colored avatar below the title, relative time, and 2-4 line summary

#### Scenario: Source visible when collapsed on Featured tier
- **WHEN** a Featured (5★) entry card is collapsed
- **THEN** the source name with colored avatar and relative time are visible below the title

#### Scenario: Source visible when collapsed on High Priority tier
- **WHEN** a High Priority (4★) entry card is collapsed
- **THEN** the source name with colored avatar and relative time are visible below the title

#### Scenario: Source visible when collapsed on Worth a Look tier
- **WHEN** a Worth a Look (3★) entry card is in its compact collapsed row
- **THEN** the source name text is visible next to the source avatar

#### Scenario: Source visible when collapsed on Low Priority tier
- **WHEN** a Low Priority (1-2★) entry card is in its compact collapsed row
- **THEN** the source name text is visible next to the source avatar

#### Scenario: Entry card with takeaways
- **WHEN** an entry has a non-empty takeaways array
- **THEN** the card displays the takeaways as a bulleted list below the summary in a compact style

#### Scenario: Entry card without takeaways
- **WHEN** an entry has null or empty takeaways
- **THEN** the card displays only the summary with no takeaway section
