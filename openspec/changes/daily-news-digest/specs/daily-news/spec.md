## ADDED Requirements

### Requirement: Daily News navigation
The system SHALL provide a Daily News option in the application navigation for authenticated users.

#### Scenario: User opens Daily News
- **WHEN** an authenticated user selects the Daily News navigation option
- **THEN** the system displays the Daily News page with the latest digest state for that user

### Requirement: User-specific Daily News settings
The system SHALL allow each authenticated user to configure Daily News enablement, generation time, timezone, and extra digest instructions. The default configuration SHALL be enabled with generation time 08:00 and timezone Europe/Amsterdam.

#### Scenario: Default settings are created
- **WHEN** an authenticated user has no Daily News settings
- **THEN** the system uses enabled=true, generation time 08:00, and timezone Europe/Amsterdam for that user

#### Scenario: User updates digest instructions
- **WHEN** a user saves extra Daily News instructions such as "Always include model releases"
- **THEN** subsequent digest generation for that user includes those instructions in the digest prompt

#### Scenario: User changes timezone
- **WHEN** a user changes the Daily News timezone to another valid IANA timezone
- **THEN** subsequent scheduled generation uses that timezone for local-time due checks

### Requirement: Scheduled user-specific digest generation
The system SHALL generate Daily News digests for each enabled user at the user's configured local time.

#### Scenario: Configured local time is due
- **WHEN** a user's Daily News settings are enabled and the configured local generation time is due in the configured timezone
- **THEN** the system starts digest generation for that user

#### Scenario: Daily News disabled
- **WHEN** a user's Daily News settings are disabled and the configured generation time is due
- **THEN** the system does not generate a digest for that user

#### Scenario: Digest already generated for local day
- **WHEN** a successful digest already exists for the user's current local date
- **THEN** the scheduler does not create a duplicate automatic digest for that local date

### Requirement: Digest input window
The system SHALL select candidate entries for digest generation using entries visible to the user that were published or discovered since the user's previous successful digest period end, or during the past 24 hours if no previous successful digest exists.

#### Scenario: Previous digest exists
- **WHEN** a user has a previous successful digest with period_end at 2026-05-07T08:00:00+02:00
- **THEN** the next digest includes visible entries whose published_at or discovered_at is after that period end and at or before the new period end

#### Scenario: No previous digest exists
- **WHEN** a user has no previous successful digest
- **THEN** digest generation uses entries visible to the user from the 24 hours before the current generation time

#### Scenario: Article was newly ingested but published earlier
- **WHEN** an entry has a published_at before the digest period but a discovered_at inside the digest period
- **THEN** the entry is eligible for the digest

### Requirement: Digest generation from existing entry summaries
The system SHALL generate Daily News using existing entry metadata, summaries, takeaways, source names, published/discovered dates, and effective star ratings rather than raw article content by default.

#### Scenario: Candidate entries have summaries
- **WHEN** digest generation runs with candidate entries that have summaries and takeaways
- **THEN** the AI prompt includes the summaries and takeaways as the article content basis

#### Scenario: Candidate entry has no summary
- **WHEN** a candidate entry has no summary yet
- **THEN** the system either omits that entry from the AI prompt or includes its title and metadata only without blocking digest generation

### Requirement: Newspaper-like digest structure
The system SHALL produce a structured Markdown digest that presents the most important items first and uses newspaper-like sections.

#### Scenario: Digest has important items
- **WHEN** digest generation succeeds with notable candidate entries
- **THEN** the stored digest contains Markdown with top-level sections for the day's most important news

#### Scenario: User has extra editorial instructions
- **WHEN** the user has configured extra editorial instructions
- **THEN** the generated digest reflects those instructions when selecting and organizing content

### Requirement: Importance-based ordering
The system SHALL prioritize digest content using effective star rating, recency, source context, AI-detected significance, breaking or developing signals, repeated themes, and the user's extra instructions.

#### Scenario: High-importance entries exist
- **WHEN** candidate entries include high-star or significant developments
- **THEN** those entries appear before lower-importance items in the digest

#### Scenario: User explicitly prioritizes model releases
- **WHEN** candidate entries include a model release and the user's instructions say model releases are important
- **THEN** the model release is included in the digest even if it is not among the highest-rated entries

### Requirement: Breaking and developing news section
The system SHALL include a dedicated breaking or developing news section when candidate entries contain urgent, time-sensitive, newly released, or rapidly changing developments.

#### Scenario: Breaking news is detected
- **WHEN** candidate entries contain breaking or developing news
- **THEN** the digest includes a dedicated breaking or developing section with links to relevant KnowledgeHub entries

#### Scenario: No breaking news is detected
- **WHEN** candidate entries contain no breaking or developing news
- **THEN** the digest may omit the breaking or developing section

### Requirement: Lower-rated interesting items section
The system SHALL include a concise "You May Also Find This Interesting" section when lower-rated candidate entries may still be useful or relevant.

#### Scenario: Lower-rated interesting entries exist
- **WHEN** lower-rated candidate entries are potentially useful based on significance or user instructions
- **THEN** the digest includes short bullet points for those entries near the bottom of the digest

#### Scenario: No lower-rated interesting entries exist
- **WHEN** no lower-rated candidate entries are worth highlighting
- **THEN** the digest may omit the lower-rated interesting section

### Requirement: KnowledgeHub entry references
The system SHALL store structured references to KnowledgeHub entry IDs used in each digest and SHALL render those references as in-app links or controls.

#### Scenario: Digest references an article
- **WHEN** a digest mentions a source article
- **THEN** the digest stores the corresponding KnowledgeHub entry ID and renders a control that opens that entry inside KnowledgeHub

#### Scenario: AI returns invalid entry reference
- **WHEN** AI output references an entry ID that was not part of the candidate set or is not visible to the user
- **THEN** the system excludes that reference from stored and rendered digest links

### Requirement: Manual generation and regeneration
The system SHALL allow authenticated users to manually generate a Daily News digest and regenerate an existing digest. Regeneration SHALL overwrite the selected digest version for the first implementation.

#### Scenario: User generates now
- **WHEN** a user clicks Generate now on the Daily News page
- **THEN** the system starts digest generation for that user using the current digest input window

#### Scenario: User regenerates existing digest
- **WHEN** a user clicks Regenerate for an existing digest
- **THEN** the system overwrites that digest's content, references, status, counts, and generated timestamp

### Requirement: Digest archive browsing
The system SHALL retain Daily News digests indefinitely and provide paginated browsing of previous digests for each user.

#### Scenario: Previous digests exist
- **WHEN** a user opens the Daily News page with multiple previous digests
- **THEN** the system shows the latest digest prominently and provides a paginated or load-more list of previous editions

#### Scenario: User selects previous digest
- **WHEN** a user selects a previous digest from the archive list
- **THEN** the system displays that digest without showing all historical digests at once

### Requirement: Empty digest handling
The system SHALL create or display an explicit "No articles today" digest state when there are no candidate entries for the generation window.

#### Scenario: No candidate entries
- **WHEN** digest generation runs and there are no visible candidate entries for the user
- **THEN** the system records a successful digest indicating that there were no articles today

### Requirement: Digest failure states
The system SHALL record and display pending or failed Daily News states when generation cannot complete.

#### Scenario: Missing AI configuration
- **WHEN** digest generation runs without required OpenRouter configuration
- **THEN** the system records a failed digest state with a clear error message for the user

#### Scenario: LLM generation fails
- **WHEN** OpenRouter returns an error during digest generation
- **THEN** the system records a failed digest state and displays the failure on the Daily News page
