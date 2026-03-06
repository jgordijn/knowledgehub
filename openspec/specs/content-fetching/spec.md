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


### Requirement: Single-article content extraction
The system SHALL support fetching and extracting content from a single article URL outside the scheduler context, for use by the Quick Add feature. The extraction SHALL use the same readability pipeline as existing watchlist scraping.

#### Scenario: Extract content from article URL
- **WHEN** the quick-add endpoint receives URL "https://example.com/article"
- **THEN** the system fetches the page, extracts title and content via readability, and returns the extracted content

#### Scenario: Extraction with readability fallback
- **WHEN** readability cannot extract meaningful content from the page
- **THEN** the system falls back to the page title and first 500 characters of body text

### Requirement: RSS feed discovery from page HTML
The system SHALL discover RSS/Atom/JSON Feed URLs by parsing `<link rel="alternate">` tags from page HTML. It SHALL check the article page first, then the site root if no feed is found.

#### Scenario: Discover RSS from link tags
- **WHEN** the page HTML contains `<link rel="alternate" type="application/rss+xml" href="/feed">`
- **THEN** the system resolves the href to an absolute URL and returns it as a discovered feed

#### Scenario: Discover Atom feed
- **WHEN** the page HTML contains `<link rel="alternate" type="application/atom+xml" href="/atom.xml">`
- **THEN** the system discovers it as a valid feed URL

#### Scenario: Fallback to site root
- **WHEN** the article page has no feed links
- **THEN** the system fetches the site root (scheme + host) and checks for feed links there

#### Scenario: Resolve relative feed URLs
- **WHEN** the feed link href is relative (e.g., "/feed.xml")
- **THEN** the system resolves it against the page URL to produce an absolute URL

### Requirement: Separator-based fragment splitting
The system SHALL support splitting fragment feed content by an explicit text separator. When a resource has `fragment_mode` set to `separated` and a non-empty `fragment_separator`, the system SHALL split content by finding DOM elements whose whitespace-normalized trimmed text content matches the separator string, using those as boundaries. Separator elements SHALL be discarded. Each group of elements between separators becomes a separate fragment.

#### Scenario: Split by separator
- **WHEN** a fragment feed resource has fragment_mode "separated" and fragment_separator "~~~", and the feed entry contains three sections of content separated by paragraphs containing only "~~~"
- **THEN** the system creates three fragment entries, one for each section, with the "~~~" separator paragraphs discarded

#### Scenario: Separator match tolerates internal whitespace variation
- **WHEN** a fragment feed resource has fragment_mode "separated" and fragment_separator "~ ~ ~", and the feed entry contains separator paragraphs like "~  ~ ~"
- **THEN** the system treats those paragraphs as matching separators and splits the entry at those boundaries

#### Scenario: Separator not found in content
- **WHEN** a fragment feed resource has fragment_mode "separated" and fragment_separator "~~~", but the feed entry content contains no elements matching "~~~"
- **THEN** the entire content is treated as a single fragment

#### Scenario: Separator at start or end of content
- **WHEN** a fragment feed entry starts with a "~~~" separator and ends with a "~~~" separator
- **THEN** leading and trailing empty fragments are discarded, and only non-empty content groups become fragments

### Requirement: Auto mode preserves existing behavior
The system SHALL treat resources with `fragment_mode` set to `auto` or empty/unset identically to the current heuristic + AI fragment splitting behavior.

#### Scenario: Auto mode fragment splitting
- **WHEN** a fragment feed resource has fragment_mode "auto" (or empty)
- **THEN** the system uses the heuristic paragraph-based splitter with optional AI regrouping, identical to current behavior

### Requirement: Separator mode skips AI regrouping
The system SHALL NOT invoke AI regrouping when fragment_mode is `separated`. The separator boundaries are authoritative.

#### Scenario: No AI call for separated mode
- **WHEN** a fragment feed resource has fragment_mode "separated" and fragment_separator "---"
- **THEN** the system splits by the separator without making any AI calls for regrouping