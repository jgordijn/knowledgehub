## ADDED Requirements

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
