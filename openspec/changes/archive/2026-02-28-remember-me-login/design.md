## Context

KnowledgeHub uses PocketBase's JS SDK (`pocketbase@0.26.8`) with a singleton client exported from `ui/src/lib/pb.ts`. The client currently uses the default `LocalAuthStore` which persists auth tokens to `localStorage`. This means sessions always survive browser restarts — there is no way for a user to opt into a temporary session.

The login form (`ui/src/routes/login/+page.svelte`) calls `pb.collection('_superusers').authWithPassword()` and the layout (`+layout.svelte`) checks `pb.authStore.isValid` on navigation to gate authenticated routes.

## Goals / Non-Goals

**Goals:**
- Let users choose between persistent (remember me) and ephemeral (session-only) login
- Ephemeral sessions are cleared when the browser tab/window closes
- The preference is remembered for the login form's checkbox default

**Non-Goals:**
- Server-side session management or token expiry changes — this is purely client-side storage
- Auto-logout timer or idle timeout
- Multi-user support or per-device session management

## Decisions

### 1. Use `LocalAuthStore` with localStorage vs sessionStorage key

**Decision:** Create a thin wrapper that delegates to either `localStorage` or `sessionStorage` based on the remember-me preference, rather than using `AsyncAuthStore`.

**Rationale:** `LocalAuthStore` uses `localStorage` internally and doesn't accept a custom `Storage` backend. However, the PocketBase SDK constructor accepts an `authStore` parameter, so we can provide a custom `BaseAuthStore`-derived store or use `AsyncAuthStore` with save/clear functions that target the correct storage.

**Approach:** Use `AsyncAuthStore` with `save`/`clear` callbacks that write to `sessionStorage` or `localStorage` depending on the preference. This keeps us within the SDK's public API.

**Alternative considered:** Swapping the entire `PocketBase` instance at login time. Rejected because other modules import the singleton — replacing it would break existing references.

### 2. Store the remember-me preference in localStorage

**Decision:** Store `kh_remember_me` flag in `localStorage` (always, not sessionStorage).

**Rationale:** This flag controls which storage backend to use on the *next* page load. If we stored it in sessionStorage, a user who unchecked "remember me" would see the checkbox reset after closing the browser — which is actually the correct UX. But we want the checkbox to reflect their *last choice* so they don't have to keep unchecking it. localStorage is the right place.

### 3. Re-initialize the auth store at login time, not at module load

**Decision:** The PocketBase client initializes with a store that reads from *both* storage backends on startup (checking localStorage first, then sessionStorage). At login time, the remember-me choice determines which backend is used for the new token.

**Rationale:** On page load, we don't yet know the user's intent — we need to restore a valid session from wherever it was stored. The login action is the decision point for which backend to target going forward.

### 4. Clear the other storage on login

**Decision:** When logging in with "remember me" checked, clear any existing auth data from `sessionStorage`. When logging in without "remember me", clear any existing auth data from `localStorage`.

**Rationale:** Prevents stale tokens in the unused storage backend from causing confusion on subsequent loads.

## Risks / Trade-offs

- **[Risk] User opens multiple tabs with different remember-me choices** → Last login wins. The `kh_remember_me` flag is shared via localStorage. This is acceptable for a single-user app.
- **[Risk] sessionStorage is per-tab in some browsers** → If the user opens a link in a new tab, the session won't carry over. This is expected behavior for "don't remember me" and matches user mental model.
- **[Trade-off] Checkbox default** → Defaults to checked (remember me). Most users on a personal Tailscale-only app want persistent sessions. The unchecked case is the exception.
