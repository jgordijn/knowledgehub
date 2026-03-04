## ADDED Requirements

### Requirement: Quick Add button on feed view
The system SHALL display a Quick Add button on the feed view that opens a modal for adding a single article by URL.

#### Scenario: Quick Add button visible
- **WHEN** the user is on the feed view
- **THEN** a "+" button is visible in a fixed position (bottom-right FAB or top bar action)

#### Scenario: Quick Add button opens modal
- **WHEN** the user taps the Quick Add button
- **THEN** a modal opens with a URL text input and an "Add" submit button

### Requirement: Submit URL for one-off article
The system SHALL accept a URL in the Quick Add modal, fetch the article content, create an entry under the system Quick Add resource, and trigger AI summarize+score processing.

#### Scenario: Valid article URL submitted
- **WHEN** the user submits URL "https://example.com/great-article" in the Quick Add modal
- **THEN** the system fetches the article via readability, creates an entry with title, URL, and raw_content under the Quick Add resource, triggers AI processing, and shows a confirmation in the modal

#### Scenario: Invalid or unreachable URL
- **WHEN** the user submits a URL that cannot be fetched (HTTP error, timeout, invalid URL)
- **THEN** the modal displays an error message and the user can try a different URL

#### Scenario: Duplicate URL
- **WHEN** the user submits a URL that already exists as an entry in the database
- **THEN** the modal displays a message indicating the article was already added

### Requirement: RSS auto-discovery
The system SHALL check the submitted article's page for RSS/Atom/JSON Feed links via `<link rel="alternate">` tags. If not found on the article page, the system SHALL also check the site root URL.

#### Scenario: RSS feed discovered on article page
- **WHEN** the article page contains `<link rel="alternate" type="application/rss+xml" href="/feed.xml">`
- **THEN** the system discovers the RSS feed URL and includes it in the response

#### Scenario: RSS feed discovered on site root
- **WHEN** the article page has no feed links but the site root "https://example.com/" contains a feed link
- **THEN** the system discovers the RSS feed URL from the site root

#### Scenario: No RSS feed found
- **WHEN** neither the article page nor the site root contain any feed links
- **THEN** the modal shows only the article confirmation with no RSS section

#### Scenario: Multiple RSS feeds found
- **WHEN** the page contains multiple feed links (e.g., main feed + comments feed)
- **THEN** the system uses the first discovered feed

### Requirement: RSS feed preview
When an RSS feed is discovered, the system SHALL fetch the feed and display the last 5 articles as clickable links with their published dates.

#### Scenario: Feed preview with 5 articles
- **WHEN** an RSS feed is discovered and contains 20 articles
- **THEN** the modal shows the 5 most recent articles, each as a clickable link (title text, opens in new tab) with the published date displayed next to it

#### Scenario: Feed preview with fewer than 5 articles
- **WHEN** an RSS feed is discovered and contains only 3 articles
- **THEN** the modal shows all 3 articles as clickable links with dates

#### Scenario: RSS feed URL displayed as link
- **WHEN** an RSS feed is discovered
- **THEN** the modal displays the feed URL as a clickable link that opens in a new tab, so the user can investigate the full feed

### Requirement: Subscribe to discovered RSS feed
When an RSS feed is discovered, the modal SHALL present three options: Add RSS, Edit, or No thanks.

#### Scenario: User clicks Add RSS
- **WHEN** the user clicks "Add RSS" after RSS discovery
- **THEN** the system automatically creates a new RSS resource with the discovered feed URL and a name derived from the site, and the modal closes with a success message

#### Scenario: User clicks Edit
- **WHEN** the user clicks "Edit" after RSS discovery
- **THEN** the modal shows a resource form pre-filled with the feed URL and site name, allowing the user to customize the name and settings before saving

#### Scenario: User clicks No thanks
- **WHEN** the user clicks "No thanks" after RSS discovery
- **THEN** the modal closes, keeping only the one-off article entry without creating an RSS resource

#### Scenario: User submits edited resource form
- **WHEN** the user edits the pre-filled resource form and clicks save
- **THEN** a new RSS resource is created with the user's customized values and the modal closes
