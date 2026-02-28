## ADDED Requirements

### Requirement: Scheduler runs every 30 minutes
The system SHALL check all active, non-quarantined resources for new content every 30 minutes.

#### Scenario: Scheduled fetch cycle
- **WHEN** 30 minutes have elapsed since the last fetch cycle
- **THEN** the system fetches all active, non-quarantined resources and updates their last_checked timestamp

#### Scenario: Skips inactive and quarantined resources
- **WHEN** a fetch cycle runs with 10 active healthy resources, 2 inactive resources, and 1 quarantined resource
- **THEN** only the 10 active healthy resources are fetched

### Requirement: RSS feed parsing
The system SHALL parse RSS, Atom, and JSON Feed formats. New entries are identified by their unique GUID/ID. Only entries not already in the database SHALL be created.

#### Scenario: New RSS entries discovered
- **WHEN** an RSS feed returns 20 items and 5 are already in the database
- **THEN** 15 new entries are created with title, URL, raw_content (from feed description/content), and published_at

#### Scenario: No new entries
- **WHEN** an RSS feed returns only items already in the database
- **THEN** no new entries are created and the resource's last_checked is updated

### Requirement: Blog scraping via readability
The system SHALL scrape watchlist resources by fetching the page HTML, extracting article links (using CSS selector if provided, otherwise heuristic link detection), and for each new link, fetching the article page and extracting content using readability algorithm.

#### Scenario: New blog post discovered via scraping
- **WHEN** a watchlist resource page contains a link to a new blog post not yet in the database
- **THEN** the system fetches the article, extracts content via readability, and creates an entry with title, URL, and raw_content

#### Scenario: Scraping with CSS selector
- **WHEN** a watchlist resource has article_selector "article h2 a"
- **THEN** the system uses that selector to find article links instead of heuristic detection

#### Scenario: Readability extraction fallback
- **WHEN** readability extraction fails for an article
- **THEN** the system stores the page title and first 500 characters of body text as raw_content

### Requirement: Quarantine after consecutive failures
The system SHALL track consecutive failures per resource. After 5 consecutive failures, the resource status SHALL be set to "quarantined". A successful fetch SHALL reset consecutive_failures to 0 and status to "healthy".

#### Scenario: Resource reaches quarantine threshold
- **WHEN** a resource fails for the 5th consecutive time with error "connection timeout"
- **THEN** resource status is set to "quarantined", quarantined_at is set to current time, and last_error is "connection timeout"

#### Scenario: Failing resource recovers
- **WHEN** a resource with 3 consecutive failures succeeds on the next fetch
- **THEN** consecutive_failures resets to 0 and status returns to "healthy"

### Requirement: Fetch timeout
The system SHALL enforce a 30-second timeout per HTTP request when fetching resources. Timeouts count as failures.

#### Scenario: Fetch times out
- **WHEN** fetching a resource takes longer than 30 seconds
- **THEN** the fetch is aborted, counted as a failure, and last_error is set to "timeout after 30s"
