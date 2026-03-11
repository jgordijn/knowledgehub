## MODIFIED Requirements

### Requirement: Combined summarization, scoring, and tagging (MODIFIED)
The system SHALL perform summarization, scoring, optional takeaway extraction, and tag extraction in a single LLM call per entry. The LLM SHALL return a `tags` array of 1-5 lowercase topic strings that capture the article's main subjects. Tags SHALL use lowercase kebab-case (e.g., "distributed-systems", "go", "performance"). When an article covers no clear topics, the tags array MAY be empty.

#### Scenario: Single prompt produces summary, score, and tags
- **WHEN** a new entry about Go concurrency patterns is processed
- **THEN** one OpenRouter API call returns the summary, star rating, and tags such as ["go", "concurrency", "goroutines"]

#### Scenario: Article with broad topics
- **WHEN** a new entry is a general industry news roundup
- **THEN** the tags array contains high-level topics like ["industry-news", "tech"] rather than enumerating every mentioned subject

#### Scenario: LLM returns invalid tags
- **WHEN** the LLM returns tags that are not lowercase or contain spaces
- **THEN** the system normalizes them to lowercase kebab-case before storing

## ADDED Requirements

### Requirement: Backfill tags for existing entries
The system SHALL provide a CLI command or API endpoint to backfill tags for existing entries that have a summary but no tags. The backfill SHALL reuse the same LLM prompt, sending the existing summary and title to extract tags without regenerating the summary or score.

#### Scenario: Backfill untagged entries
- **WHEN** the backfill command is run and 200 entries have null tags
- **THEN** the system processes each entry through the LLM to extract tags, storing the result in the tags field

#### Scenario: Backfill skips already-tagged entries
- **WHEN** the backfill command is run and an entry already has a non-null tags array
- **THEN** that entry is skipped
