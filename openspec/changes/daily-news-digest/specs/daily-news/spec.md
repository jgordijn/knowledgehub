## ADDED Requirements

### Requirement: Daily News navigation
The system SHALL provide a Daily News option in the application navigation for authenticated users.

#### Scenario: User opens Daily News
- **WHEN** an authenticated user selects the Daily News navigation option
- **THEN** the system displays the Daily News page with the latest digest state for that user

### Requirement: User-specific Daily News settings
The system SHALL allow each authenticated user to configure Daily News enablement, generation time, timezone, and extra digest instructions through authenticated server-side settings behavior. The default configuration SHALL be enabled with generation time 08:00 and timezone Europe/Amsterdam. Extra digest instructions SHALL have a maximum persisted length of 2000 Unicode code points and SHALL allow printable Unicode plus tab, line feed, and carriage return (`\t`, `\n`, `\r`) while rejecting other Unicode control/format characters before storage. Daily News settings SHALL be stored with a `user` owner field, SHALL enforce exactly one settings record per user with a database-level uniqueness invariant, and supported end-user access SHALL be through server-side settings routes that derive the owner from `@request.auth.id` and enforce ownership before lookup or mutation. Generic collection list/view SHALL be owner-scoped, generic create/delete SHALL be denied as defense in depth, and settings creation/update SHALL use idempotent server-side get-or-create/update behavior that never accepts an arbitrary owner ID. The settings API SHALL expose authenticated server-side read and save behavior equivalent to `GET /api/daily-news/settings` and `PUT /api/daily-news/settings`: missing settings are materialized and returned with `200 OK`, valid saves return `200 OK` with the saved settings, invalid saves return `400 Bad Request` with a sanitized validation error and preserve previous values, and unauthenticated requests are denied before lookup or materialization. PocketBase `_superusers` SHALL be treated as fully privileged administrative identities that can bypass collection rules; therefore generic collection rules SHALL NOT be the security boundary for Daily News superuser-authenticated end-user behavior.

#### Scenario: Settings GET materializes defaults
- **WHEN** an authenticated user calls the Daily News settings read route and has no Daily News settings
- **THEN** the system creates or materializes one settings record for that user and returns `200 OK` with enabled=true, generation time 08:00, and timezone Europe/Amsterdam

#### Scenario: Default settings are created
- **WHEN** an authenticated user has no Daily News settings
- **THEN** the system creates or materializes one settings record for that user with enabled=true, generation time 08:00, and timezone Europe/Amsterdam

#### Scenario: Duplicate settings creation is prevented
- **WHEN** settings materialization or user saves race for the same authenticated user
- **THEN** the system preserves exactly one settings record for that user and returns or updates that record idempotently

#### Scenario: Scheduler sees default settings
- **WHEN** a PocketBase `_superusers` user has not opened the Daily News settings page
- **THEN** scheduled generation still considers that user by using the persisted default settings record

#### Scenario: User is created after startup
- **WHEN** a new PocketBase `_superusers` user is created after application startup
- **THEN** a later scheduler/settings materialization pass discovers that user and creates the default settings record

#### Scenario: User updates digest instructions
- **WHEN** a user saves extra Daily News instructions such as "Always include model releases"
- **THEN** subsequent digest generation for that user includes those instructions in the digest prompt

#### Scenario: User saves oversized digest instructions
- **WHEN** a user saves extra Daily News instructions longer than 2000 Unicode code points
- **THEN** the backend rejects the settings change, the previous valid instructions remain unchanged, and the frontend prevents or reports the same limit before submitting where possible

#### Scenario: User saves unsupported control content in digest instructions
- **WHEN** a user saves extra Daily News instructions containing Unicode control or format characters other than `\t`, `\n`, or `\r`
- **THEN** the backend rejects the settings change, keeps the previous valid instructions unchanged, and does not store unsafe control content

#### Scenario: User changes timezone
- **WHEN** a user changes the Daily News timezone to another valid IANA timezone
- **THEN** subsequent scheduled generation uses that timezone for local-time due checks

#### Scenario: User saves invalid timezone
- **WHEN** a user saves a timezone that is not a valid IANA timezone name
- **THEN** the system rejects the settings change and keeps the previous valid timezone

#### Scenario: User saves invalid generation time
- **WHEN** a user saves a generation time that is not a valid 24-hour `HH:MM` value
- **THEN** the system rejects the settings change and keeps the previous valid generation time

#### Scenario: User accesses another user's settings
- **WHEN** an authenticated user lists or views Daily News settings
- **THEN** the operation is allowed only for settings whose `user` equals `@request.auth.id`

#### Scenario: User attempts generic settings create or delete
- **WHEN** an authenticated user attempts to create or delete Daily News settings through the generic collection API
- **THEN** the operation is denied and settings creation/deletion remains controlled by server-side materialization behavior

#### Scenario: User updates settings through server route
- **WHEN** an authenticated user saves valid Daily News settings through the settings route
- **THEN** the system returns `200 OK` and updates or creates that user's single settings record without accepting an arbitrary `user` owner from the request body

#### Scenario: Settings save validation fails
- **WHEN** an authenticated user saves invalid Daily News settings through the settings route
- **THEN** the system returns `400 Bad Request` with a sanitized validation error and preserves the user's previously valid settings values

#### Scenario: Unauthenticated settings request is denied
- **WHEN** a request without valid authentication reads or saves Daily News settings through server routes or generic collection APIs
- **THEN** the system denies the request without materializing settings for an anonymous user or revealing any user's settings

### Requirement: Scheduled user-specific digest generation
The system SHALL generate Daily News digests for each enabled user at the user's configured local time. Daily digests SHALL be stored with a `user` owner field and a status of `pending`, `running`, `success`, or `failed`. Supported end-user reads and mutations SHALL be performed only by server-side routes that derive the user from authenticated context and enforce owner checks before lookup, response, or mutation. User-facing collection rules SHALL allow owner-scoped list/view only and deny create, update, and delete mutations through the generic collection API as defense in depth, but PocketBase `_superusers` SHALL be considered fully privileged administrators that can bypass collection rules. The system SHALL enforce at most one active (`pending` or `running`) digest job per canonical `(user, local_date, period_start, period_end)` window using deterministic job/window keys or equivalent transaction-safe locks, and SHALL enforce at most one active scheduled/due job and at most one successful automatic/scheduled digest per `(user, local_date)`. Ad-hoc manual digests created before the configured due time SHALL be marked as manual and SHALL NOT count as the successful scheduled digest for that local date.

#### Scenario: Configured local time is due
- **WHEN** a user's Daily News settings are enabled and the configured local generation time is due in the configured timezone
- **THEN** the system starts digest generation for that user

#### Scenario: Missed local time after same-day downtime
- **WHEN** the application starts or the scheduler checks after the configured local generation time but before the user's next local date, and no successful or active automatic digest exists for that local date
- **THEN** the system treats that local day's digest as due and starts exactly one digest generation for that user

#### Scenario: Missed previous local date is too late
- **WHEN** the application starts or the scheduler checks on a later local date after a prior day's configured generation time was missed
- **THEN** the system does not backfill the missed prior local date automatically and evaluates only the current local date for due generation

#### Scenario: Daily News disabled
- **WHEN** a user's Daily News settings are disabled and the configured generation time is due
- **THEN** the system does not generate a digest for that user

#### Scenario: Digest already generated for local day
- **WHEN** a successful scheduled digest already exists for the user's current local date
- **THEN** the scheduler does not create a duplicate automatic digest for that local date

#### Scenario: Pre-due manual digest does not suppress scheduled run
- **WHEN** a user creates a successful ad-hoc manual digest before the configured generation time for the current local date
- **THEN** the scheduler still treats the configured generation time as due later that day and may create exactly one scheduled digest whose input window starts after the manual digest period end

#### Scenario: Active digest job already exists
- **WHEN** a pending or running scheduled digest already exists for the same user and local digest date, or any pending/running digest already exists for the same user, local digest date, `period_start`, and `period_end` window
- **THEN** scheduled or manual generation does not create another active digest job and returns or displays the existing active digest state

#### Scenario: Concurrent active job creation races
- **WHEN** scheduled and manual generation attempt to create an active digest for the same user and local date/window at the same time, even with slightly different observed `now` instants
- **THEN** shared canonical window derivation and an atomic uniqueness or locking mechanism allow at most one active digest job to be created for that `(user, local_date)` and canonical `(user, local_date, period_start, period_end)` key

#### Scenario: Failed digest retry does not violate active uniqueness
- **WHEN** a failed digest exists for a user's local date/window and the user or scheduler retries that window
- **THEN** the system may create or claim a new `pending` active job because failed digests are historical records and do not count as active jobs

#### Scenario: Digest status transitions
- **WHEN** a digest generation job is claimed and processed
- **THEN** its status transitions from `pending` to `running` and then to either `success` or `failed` with no other user-visible terminal state

#### Scenario: Stale pending job is recovered
- **WHEN** the application starts or the scheduler checks and finds a `pending` digest job older than the configured pending timeout
- **THEN** the system either resumes the pending job for processing or marks it `failed` with a sanitized timeout message before allowing a new claim for that user/date/window

#### Scenario: Stale running job is recovered
- **WHEN** the application starts or the scheduler checks and finds a `running` digest job whose `started_at` or heartbeat is older than the configured running timeout
- **THEN** the system marks it `failed` with a sanitized timeout message so it no longer blocks generation or regeneration for that user/date/window

#### Scenario: Non-stale running job remains active
- **WHEN** a `running` digest job is within the configured running timeout
- **THEN** stale-job recovery leaves it active and duplicate generation/regeneration remains blocked

#### Scenario: DST spring-forward due check
- **WHEN** the configured local generation time falls on a daylight-saving spring-forward day
- **THEN** the scheduler evaluates due generation using the configured timezone's local date/time rules and creates at most one digest for that local date

#### Scenario: DST fall-back due check
- **WHEN** the configured local generation time occurs during a daylight-saving fall-back repeated hour
- **THEN** the scheduler creates at most one digest for that user and local date

#### Scenario: User reads another user's digest
- **WHEN** an authenticated user lists or views Daily News digests
- **THEN** the operation is allowed only for digests whose `user` equals `@request.auth.id`

#### Scenario: User attempts generic digest mutation
- **WHEN** an authenticated user attempts to create, update, or delete Daily News digests through the generic collection API
- **THEN** the operation is denied even if the payload uses that user's ID

### Requirement: Digest input window
The system SHALL select candidate entries for digest generation using entries visible to the user that were published or discovered since the user's previous successful digest period end, or during the past 24 hours before the canonical period end if no previous successful digest exists. The system SHALL derive `local_date`, `period_start`, `period_end`, `trigger`, and `job_key` through a shared deterministic claim routine with UTC-normalized timestamp precision so scheduled and manual attempts for the same local day/window produce the same key. Manual generation before the configured due time SHALL use a deterministic ad-hoc manual period end and SHALL NOT mark the local date's scheduled digest as completed; later scheduled generation SHALL start after that manual digest's successful period end. Failed digests SHALL NOT advance the next generation window.

#### Scenario: Previous digest exists
- **WHEN** a user has a previous successful digest with period_end at 2026-05-07T08:00:00+02:00
- **THEN** the next digest includes visible entries whose published_at or discovered_at is after that period end and at or before the new period end

#### Scenario: No previous digest exists
- **WHEN** a user has no previous successful digest
- **THEN** digest generation uses entries visible to the user from the 24 hours before the current generation time

#### Scenario: Article was newly ingested but published earlier
- **WHEN** an entry has a published_at before the digest period but a discovered_at inside the digest period
- **THEN** the entry is eligible for the digest

#### Scenario: Previous digest failed
- **WHEN** a user's most recent digest is failed and an earlier successful digest exists
- **THEN** the next digest input window starts after the earlier successful digest's period_end

### Requirement: Digest generation from existing entry summaries
The system SHALL generate Daily News using existing entry metadata, summaries, takeaways, source names, published/discovered dates, and effective star ratings rather than raw article content by default. Prompt construction SHALL be bounded by a deterministic candidate preselection ordered by importance signals and SHALL store candidate_count and included_count metadata.

#### Scenario: Candidate entries have summaries
- **WHEN** digest generation runs with candidate entries that have summaries and takeaways
- **THEN** the AI prompt includes the summaries and takeaways as the article content basis

#### Scenario: Candidate entry has no summary
- **WHEN** a candidate entry has no summary yet
- **THEN** the system either omits that entry from the AI prompt or includes its title and metadata only without blocking digest generation

#### Scenario: Candidate volume exceeds prompt limit
- **WHEN** more visible candidate entries exist than can be safely included in one digest prompt
- **THEN** the system deterministically selects entries by effective stars, recency, breaking/developing signals, source, and title tie-breakers, stores the total candidate_count and included_count, and marks that the digest used a subset

#### Scenario: Digest is based on a subset
- **WHEN** a stored digest used fewer included entries than the total candidate count
- **THEN** the Daily News page indicates that the digest is based on a subset of available articles

### Requirement: Prompt injection boundaries
The system SHALL construct Daily News prompts so entry fields and user extra instructions are treated as untrusted data, not as model/system instructions. Entry titles, summaries, takeaways, source names, dates, IDs, and user extra instructions SHALL be delimited or encoded, and user extra instructions SHALL be bounded to the persisted 2000-code-point limit before inclusion.

#### Scenario: Article summary contains adversarial instructions
- **WHEN** a candidate entry summary says to ignore previous instructions or change output format
- **THEN** the prompt identifies that text as article data and instructs the model not to follow instructions contained inside article fields

#### Scenario: User instructions exceed safe bounds
- **WHEN** a user's extra Daily News instructions exceed the configured length or contain unsupported control/format content outside printable Unicode plus `\t`, `\n`, and `\r`
- **THEN** settings validation rejects the invalid value before storage, and prompt construction only includes the last valid bounded instructions while preserving valid editorial preferences

#### Scenario: Delimited data is included in prompt
- **WHEN** digest prompt construction includes entry fields and user instructions
- **THEN** tests verify those fields are placed inside explicit data delimiters or encoded sections separate from system task instructions

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

### Requirement: Safe digest rendering
The system SHALL render Daily News Markdown through a sanitizer with an explicit allowlist. Allowed Markdown elements SHALL be limited to headings, paragraphs, emphasis/strong, blockquotes, ordered/unordered lists, tables, and inline/fenced code. Raw HTML, scripts, event-handler attributes, iframes, styles, SVG, model-generated images, and untrusted model-generated links SHALL be stripped or neutralized. Markdown links SHALL either be rendered as text or allowed only for `https://` URLs with safe link attributes such as `rel="noopener noreferrer"`; `javascript:`, `data:`, `file:`, protocol-relative, and other non-allowlisted schemes SHALL be removed or neutralized. KnowledgeHub article controls SHALL be rendered only from validated structured IDs, not Markdown URLs.

#### Scenario: Digest Markdown contains raw HTML or scripts
- **WHEN** a digest body contains raw HTML, script tags, event-handler attributes, iframes, styles, SVG, or similar executable content
- **THEN** the rendered Daily News page strips or neutralizes that content before display

#### Scenario: LLM returns arbitrary external links or images
- **WHEN** model-generated Markdown includes arbitrary external links or image references
- **THEN** the renderer removes images and renders links only when they satisfy the explicit allowlist policy; KnowledgeHub article links are not trusted from Markdown URLs

#### Scenario: LLM returns dangerous link schemes
- **WHEN** model-generated Markdown includes `javascript:`, `data:`, `file:`, or protocol-relative links
- **THEN** the renderer removes or neutralizes those links before display

#### Scenario: LLM returns allowed HTTPS link
- **WHEN** model-generated Markdown includes an allowed `https://` link and external links are enabled by policy
- **THEN** the renderer preserves the link with safe attributes such as `rel="noopener noreferrer"`

### Requirement: KnowledgeHub entry references
The system SHALL store structured references to KnowledgeHub entry IDs used in each digest and SHALL render those references as in-app links or controls. Internal KnowledgeHub references SHALL be rendered only from validated structured IDs, not from model-generated Markdown URLs. Digest Markdown MAY contain inline entry markers in the exact form `[[kh-entry:<entry_id>]]`; the renderer SHALL replace a marker at that location with an in-app entry control only when `<entry_id>` is present in the validated, deduplicated `referenced_entry_ids` for that digest and remains visible to the digest owner. Invalid markers, duplicate IDs in AI structured output, and marker IDs missing from validated references SHALL NOT create trusted links or controls. Stored digest Markdown SHALL be treated as an immutable historical snapshot visible to the digest owner; if a referenced entry later becomes unavailable, the snapshot body remains visible to that owner while structured links are removed or shown as unavailable.

#### Scenario: Digest references an article
- **WHEN** a digest mentions a source article with a `[[kh-entry:<entry_id>]]` marker and the same ID appears in validated structured references
- **THEN** the digest stores the corresponding KnowledgeHub entry ID and renders a control at the marker location that opens that entry inside KnowledgeHub

#### Scenario: AI returns invalid entry reference
- **WHEN** AI output references an entry ID that was not part of the candidate set or is not visible to the user
- **THEN** the system excludes that reference from stored and rendered digest links

#### Scenario: AI returns duplicate entry references
- **WHEN** AI structured output repeats the same valid entry ID multiple times
- **THEN** the system stores that ID once in `referenced_entry_ids` while allowing valid inline marker occurrences to render at their marker locations

#### Scenario: Markdown marker is not validated
- **WHEN** digest Markdown contains a `[[kh-entry:<entry_id>]]` marker whose ID is absent from the validated structured references
- **THEN** the renderer does not create an in-app entry control for that marker

#### Scenario: Referenced entry visibility changes
- **WHEN** a stored digest references an entry that is no longer visible to the requesting user
- **THEN** the system keeps the archived digest body visible to the digest owner but does not render an in-app link for that entry and shows an unavailable-entry state if needed

#### Scenario: Referenced entry is deleted after archive retention
- **WHEN** an entry mentioned in an older retained digest is deleted after the digest was generated
- **THEN** the digest remains in the owner's archive as a historical snapshot and any structured control for that entry is unavailable rather than opening stale or unauthorized entry data

### Requirement: Manual generation and regeneration
The system SHALL allow authenticated users to manually generate a Daily News digest and regenerate an existing digest for their own user only through asynchronous server-side routes. A newly queued generation SHALL persist a digest/job record in `pending` state before returning `202 Accepted`, processing SHALL be performed by a durable worker/claimer that can pick up persisted pending jobs after restart, processing SHALL advance `pending -> running -> success|failed`, and the Daily News page SHALL observe completion by polling or realtime updates. Regeneration SHALL be the explicit exception to immutable archive snapshots: it targets the selected digest period while preserving that digest's original period_start, period_end, and local digest date, SHALL NOT overwrite a digest while that digest or another digest for the same user/date/window is `pending` or `running`, SHALL preserve previously successful content while regeneration is pending/running, SHALL replace content only on successful regeneration, and SHALL preserve prior successful content plus a sanitized failure state if regeneration fails.

#### Scenario: User generates now
- **WHEN** a user clicks Generate now on the Daily News page and no same-day/window active job or same-day successful scheduled digest exists for a due scheduled window
- **THEN** the system atomically claims a job for that authenticated user using the canonical digest input window and returns `202 Accepted` with the `pending` digest/job record

#### Scenario: User generates before scheduled time
- **WHEN** a user clicks Generate now before the configured generation time is due for the current local date
- **THEN** the system creates an ad-hoc manual digest job for a deterministic manual window and does not mark the local date's scheduled digest as completed

#### Scenario: Pending job survives request-process interruption
- **WHEN** Generate now returns `202 Accepted` after persisting a pending job but the process exits before an in-request goroutine starts
- **THEN** startup or scheduler worker scanning later claims that pending job or stale-job recovery marks it failed according to the configured pending timeout

#### Scenario: Worker claims one pending job
- **WHEN** multiple workers or wakeups attempt to process the same pending digest
- **THEN** a transactional compare-and-set claim changes the job to `running` for only one worker and all other workers leave it unchanged

#### Scenario: Generate now finds active digest
- **WHEN** a user clicks Generate now and a pending or running digest already exists for that user and local day/window
- **THEN** the system returns the existing active digest record instead of creating another digest job

#### Scenario: Generate now finds successful digest for local day
- **WHEN** a user clicks Generate now and a successful scheduled digest already exists for that user and local day
- **THEN** the system returns `200 OK` with the existing digest and does not overwrite it unless the user chooses Regenerate

#### Scenario: Generate now after failed digest
- **WHEN** a user clicks Generate now after a failed digest for the same local day/window
- **THEN** the system may start a new digest job with `202 Accepted` because failed digests do not block retry and do not advance the automatic window

#### Scenario: Daily News page observes queued generation
- **WHEN** Generate now or scheduled generation returns or displays a `pending` or `running` digest
- **THEN** the Daily News page shows the active status and refreshes the digest by polling or realtime updates until the record reaches `success` or `failed`

#### Scenario: User regenerates existing digest successfully
- **WHEN** a user clicks Regenerate for an existing successful or failed digest they own, no digest for the same user, local date, and period is pending or running, and the regeneration succeeds
- **THEN** the system replaces that digest's content, references, status, counts, and generated timestamp while preserving its period_start, period_end, and local digest date

#### Scenario: User regenerates pending or running digest
- **WHEN** a user clicks Regenerate for a digest that is pending or running
- **THEN** the system returns the existing active digest state and does not start another job or overwrite existing successful content

#### Scenario: User regenerates while same-day or same-window job is active
- **WHEN** a user clicks Regenerate for a digest and another pending or running digest exists for the same user and local date or for the same user, local date, `period_start`, and `period_end`
- **THEN** the system returns or displays the active digest state and does not overwrite either digest until the active job reaches a terminal state

#### Scenario: User regenerates another user's digest
- **WHEN** a user attempts to regenerate a digest whose `user` does not equal `@request.auth.id`
- **THEN** the system denies the request without revealing that digest's contents

#### Scenario: Unauthenticated manual generation is denied
- **WHEN** a request without valid authentication calls Generate now
- **THEN** the system denies the request without creating a job or revealing digest existence

#### Scenario: Unauthenticated regeneration is denied
- **WHEN** a request without valid authentication calls Regenerate for any digest ID
- **THEN** the system denies the request without creating a job, overwriting content, or revealing whether the digest exists

#### Scenario: Regeneration preserves previous success while active
- **WHEN** regeneration of a previously successful digest is pending or running
- **THEN** the digest keeps `has_successful_snapshot=true` and preserves the prior successful body, title, references, counts, and last-success timestamp for display while the latest attempt status is active

#### Scenario: Regeneration fails after previous success
- **WHEN** regeneration of a previously successful digest fails after entering an active state
- **THEN** the system keeps `has_successful_snapshot=true`, keeps the prior successful body and validated references visible, stores a sanitized failure state/message for the failed regeneration attempt, and does not replace the digest content with partial or failed output

#### Scenario: Regeneration fails without previous success
- **WHEN** regeneration or retry of a digest that has never succeeded fails
- **THEN** the digest has no successful snapshot to display, stores only a sanitized failure state/message, and leaves body/reference/count snapshot fields empty or non-authoritative

### Requirement: Digest archive browsing
The system SHALL retain Daily News digests indefinitely and provide authenticated route-level paginated browsing of previous digests for each user. Digest retrieval routes SHALL derive the owner from authentication, return only caller-owned digests, and deny unauthenticated requests before lookup.

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
The system SHALL record and display pending or failed Daily News states when generation cannot complete. Stored and displayed error messages SHALL be sanitized and safe for end users.

#### Scenario: Missing AI configuration
- **WHEN** digest generation runs without required OpenRouter configuration
- **THEN** the system records a failed digest state with a clear error message for the user

#### Scenario: LLM generation fails
- **WHEN** OpenRouter returns an error during digest generation
- **THEN** the system records a failed digest state and displays the failure on the Daily News page

#### Scenario: Failure contains sensitive details
- **WHEN** an upstream AI or internal error includes API keys, provider payloads, stack traces, or other sensitive details
- **THEN** the system stores and displays only a sanitized user-safe error message and excludes secrets from user-visible digest fields
