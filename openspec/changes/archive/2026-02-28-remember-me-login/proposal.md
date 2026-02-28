## Why

Users must re-enter credentials every time they open KnowledgeHub after clearing browser data or using a new tab in private browsing. Currently, PocketBase's auth token is always stored in `localStorage`, meaning sessions persist indefinitely. There is no way for users to choose a temporary session (e.g., on a shared device) or explicitly opt into persistent login.

## What Changes

- Add a "Remember me" checkbox to the login form
- When checked: auth token persists in `localStorage` (current behavior)
- When unchecked: auth token is stored in `sessionStorage` and cleared when the browser tab/window closes
- PocketBase client initialization switches auth store backend based on the saved preference

## Capabilities

### New Capabilities
- `remember-me`: Login form checkbox that controls auth token persistence — localStorage for persistent sessions, sessionStorage for ephemeral sessions

### Modified Capabilities
_(none — no existing spec-level behavior changes)_

## Impact

- **Frontend only** — no backend changes required
- `ui/src/lib/pb.ts` — PocketBase client instantiation must select the correct auth store backend
- `ui/src/routes/login/+page.svelte` — add checkbox UI and wire preference to PocketBase client
- No new dependencies (PocketBase JS SDK already supports custom auth stores)
- No API changes, no database changes
