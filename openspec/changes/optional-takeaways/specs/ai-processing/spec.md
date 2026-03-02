## MODIFIED Requirements

### Requirement: Combined summarization and scoring
The system SHALL perform summarization, scoring, and optional takeaway extraction in a single LLM call per entry to minimize API costs and latency. The LLM SHALL include a `takeaways` array (up to 5 bullet points) only when the article is long or complex enough that the summary alone does not capture all key points. When the summary sufficiently covers the content, takeaways SHALL be omitted.

#### Scenario: Single prompt produces summary and score
- **WHEN** a new entry is processed
- **THEN** one OpenRouter API call returns both the summary text and the star rating

#### Scenario: Long article produces takeaways
- **WHEN** a new entry is processed and the article is long or covers multiple distinct points
- **THEN** the LLM response includes a `takeaways` array with up to 5 concise bullet-point strings alongside the summary and stars

#### Scenario: Short article omits takeaways
- **WHEN** a new entry is processed and the article is short or has a single clear point
- **THEN** the LLM response contains only `summary` and `stars` with no `takeaways` field, and the entry's takeaways field is stored as null

### Requirement: Summarize new entries
The system SHALL generate a 2-4 line summary for each new entry using the configured LLM via OpenRouter. The summary SHALL capture the key points of the article. When the LLM also returns takeaways, they SHALL be stored in the entry's `takeaways` field as a JSON array of strings.

#### Scenario: Entry summarized
- **WHEN** a new entry is created with raw_content of a blog post about CRDTs
- **THEN** the system generates a concise 2-4 line summary capturing the article's main arguments and stores it in the entry's summary field

#### Scenario: Entry summarized with takeaways
- **WHEN** a new entry is created with raw_content of a long research article covering multiple findings
- **THEN** the system stores the summary in the summary field and an array of key takeaway strings in the takeaways field

#### Scenario: LLM unavailable during summarization
- **WHEN** OpenRouter returns an error during summarization
- **THEN** the entry is created with summary set to null, takeaways set to null, and a "pending" processing status, to be retried on the next cycle
