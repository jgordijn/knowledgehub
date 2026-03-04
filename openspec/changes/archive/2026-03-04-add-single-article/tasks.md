## 1. Database & Resource Type

- [x] 1.1 Add `quickadd` to the resources collection `type` field values in `collections.go` and add migration to update existing collections
- [x] 1.2 Create `ensureQuickAddResource(app)` function that auto-creates the Quick Add system resource on startup if it doesn't exist
- [x] 1.3 Call `ensureQuickAddResource` from `registerCollections` in `collections.go`

## 2. RSS Discovery

- [x] 2.1 Create `internal/engine/discovery.go` with `DiscoverFeeds(pageURL string, client *http.Client) ([]string, error)` — fetches page HTML, parses `<link rel="alternate">` tags for RSS/Atom/JSON Feed, resolves relative URLs, falls back to site root if none found
- [x] 2.2 Create `internal/engine/discovery_test.go` with tests: feed found on article page, feed found on site root fallback, no feed found, relative URL resolution, multiple feeds returns first

## 3. Quick Add API Endpoint

- [x] 3.1 Create `internal/routes/quickadd.go` with `POST /api/quick-add` handler: accepts `{ url: string }`, extracts content via readability, checks for duplicate URL, creates entry under Quick Add resource, discovers RSS feeds, fetches feed preview (last 5 articles: title, url, published date)
- [x] 3.2 Create `POST /api/quick-add/subscribe` handler: accepts `{ feed_url: string, name: string }`, creates a new RSS resource
- [x] 3.3 Register both routes in `main.go` via `routes.RegisterQuickAddRoutes(se)`
- [x] 3.4 Create `internal/routes/quickadd_test.go` with tests: successful quick-add, duplicate URL rejection, RSS discovery included in response, subscribe creates resource, invalid URL returns error

## 4. Scheduler Guard

- [x] 4.1 Verify the scheduler already skips `quickadd` type resources (it dispatches on `rss`/`watchlist` only) — add explicit test case in scheduler tests if not covered

## 5. Frontend: QuickAddModal Component

- [x] 5.1 Create `ui/src/lib/components/QuickAddModal.svelte` with three states: URL input → processing/result → optional RSS edit form
- [x] 5.2 Input state: URL text field + "Add" button, loading spinner during submission
- [x] 5.3 Result state: confirmation of added article (title), and if RSS found: feed URL as external link, last 5 articles as clickable links (title + date, open in new tab), three action buttons (Add RSS / Edit / No thanks)
- [x] 5.4 Edit state: reuse ResourceForm component pre-filled with feed URL and site name, save creates the resource and closes the modal
- [x] 5.5 Error handling: show error messages for failed fetches, network errors

## 6. Frontend: Feed View Integration

- [x] 6.1 Add Quick Add "+" button to the feed view (`+page.svelte`) — fixed FAB or top bar action
- [x] 6.2 Wire button to open QuickAddModal, handle modal close/success events
- [x] 6.3 On successful add, the new entry appears in the feed via existing realtime subscription

## 7. Frontend: Hide Quick Add Resource from Management

- [x] 7.1 Filter out `quickadd` type resources from the resources management page (`resources/+page.svelte`) so the system resource is not shown as editable

## 8. Testing & Verification

- [x] 8.1 Run full test suite (`go test ./internal/... -count=1`) and verify all pass
- [x] 8.2 Build and manual test: add article via Quick Add, verify entry appears in feed with AI processing, verify RSS discovery works on a site with known RSS, verify Add RSS creates a resource, verify Edit flow works
