## ADDED Requirements

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

### Requirement: Score entries with stars
The system SHALL assign an ai_stars rating (1-5) to each new entry based on the preference profile and article content. The rating reflects predicted relevance to the user.

#### Scenario: Entry scored with preference profile
- **WHEN** a new article about distributed systems arrives and the preference profile indicates high interest in distributed systems
- **THEN** the system assigns ai_stars of 4 or 5

#### Scenario: Entry scored without preference profile
- **WHEN** a new article arrives and no preference profile exists yet (fewer than 10 user corrections)
- **THEN** the system assigns ai_stars based on general quality signals (depth, originality) without personalization

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

### Requirement: User can override star rating
The system SHALL allow the user to change the star rating of any entry. The user rating is stored separately from the AI rating. The effective rating for display and sorting SHALL be the user rating if set, otherwise the AI rating.

#### Scenario: User overrides AI rating
- **WHEN** an entry has ai_stars 2 and user changes it to 5
- **THEN** user_stars is set to 5, ai_stars remains 2, and the entry sorts as a 5-star entry

#### Scenario: Entry with no user override
- **WHEN** an entry has ai_stars 4 and no user_stars
- **THEN** the entry displays and sorts as 4 stars

### Requirement: Preference profile generation
The system SHALL generate a text-based preference profile by analyzing all user star corrections. The profile SHALL be regenerated every 20 corrections or weekly, whichever comes first. The profile is used in scoring prompts for new entries.

#### Scenario: Profile generated after corrections
- **WHEN** the user has made 20 star corrections since the last profile generation
- **THEN** the system sends all corrections to the LLM with a prompt to summarize user interests and preferences, storing the result as the current preference profile

#### Scenario: Profile used in scoring
- **WHEN** a new entry is being scored and a preference profile exists
- **THEN** the scoring prompt includes the preference profile text and the last 20 corrections as context

### Requirement: OpenRouter model selection
The system SHALL allow the user to configure which OpenRouter model to use for summarization, scoring, and chat. The model can be changed in settings.

#### Scenario: Change model
- **WHEN** user changes the model from "anthropic/claude-sonnet-4" to "openai/gpt-4o" in settings
- **THEN** subsequent AI operations use the new model

### Requirement: OpenRouter API key configuration
The system SHALL require an OpenRouter API key configured in settings before AI features function.

#### Scenario: No API key configured
- **WHEN** the scheduler processes new entries but no OpenRouter API key is set
- **THEN** entries are created with null summary and null ai_stars, marked as pending processing
