## ADDED Requirements

### Requirement: User can add a resource
The system SHALL allow the user to add a resource by providing a name, URL, and type (RSS or watchlist). For watchlist type, the user MAY optionally provide CSS selectors for article links and content extraction.

#### Scenario: Add an RSS resource
- **WHEN** user submits the add resource form with name "Go Blog", URL "https://go.dev/blog/feed.atom", and type "rss"
- **THEN** system creates a resource with status "healthy", consecutive_failures 0, and active true

#### Scenario: Add a watchlist resource
- **WHEN** user submits the add resource form with name "Example Blog", URL "https://example.com/blog", type "watchlist", and optional article_selector "article h2 a"
- **THEN** system creates a watchlist resource with the provided selectors

### Requirement: User can edit a resource
The system SHALL allow the user to edit any field of an existing resource (name, URL, type, selectors, active status).

#### Scenario: Edit resource URL
- **WHEN** user changes the URL of an existing resource and saves
- **THEN** system updates the resource and resets consecutive_failures to 0 and status to "healthy"

### Requirement: User can remove a resource
The system SHALL allow the user to delete a resource. Deleting a resource SHALL also delete all associated entries.

#### Scenario: Remove a resource with entries
- **WHEN** user deletes a resource that has 50 associated entries
- **THEN** system deletes the resource and all 50 entries

### Requirement: User can deactivate a resource
The system SHALL allow the user to set a resource as inactive. Inactive resources SHALL NOT be checked by the scheduler.

#### Scenario: Deactivate a resource
- **WHEN** user sets a resource to inactive
- **THEN** the scheduler skips this resource during fetch cycles

### Requirement: Resource list shows health status
The system SHALL display each resource with a health indicator: healthy (green), failing (yellow with failure count), or quarantined (red with error message and quarantine date).

#### Scenario: Display failing resource
- **WHEN** a resource has 3 consecutive failures with last error "HTTP 503"
- **THEN** the resource list shows a yellow indicator with "3/5 failures — HTTP 503"

#### Scenario: Display quarantined resource
- **WHEN** a resource is quarantined since 2 days ago with error "DNS resolution failed"
- **THEN** the resource list shows a red indicator with "Quarantined 2 days ago — DNS resolution failed" and a "Retry Now" button

### Requirement: User can retry a quarantined resource
The system SHALL allow the user to manually retry a quarantined resource, resetting its status to "healthy" and consecutive_failures to 0, triggering an immediate fetch.

#### Scenario: Retry quarantined resource succeeds
- **WHEN** user clicks "Retry Now" on a quarantined resource and the fetch succeeds
- **THEN** resource status becomes "healthy" and new entries are processed

#### Scenario: Retry quarantined resource fails
- **WHEN** user clicks "Retry Now" on a quarantined resource and the fetch fails
- **THEN** resource status becomes "failing" with consecutive_failures set to 1

### Requirement: Quarantine banner on main views
The system SHALL display a prominent banner on the feed view when any resources are quarantined, showing the count of quarantined resources with a link to the resources page.

#### Scenario: Quarantine banner visible
- **WHEN** 2 resources are quarantined and user is on the feed view
- **THEN** a banner displays "2 resources are quarantined" with a link to the resources page
