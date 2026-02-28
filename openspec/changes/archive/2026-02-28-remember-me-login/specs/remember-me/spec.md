## ADDED Requirements

### Requirement: Remember-me checkbox on login form
The login form SHALL display a "Remember me" checkbox between the password field and the submit button.

The checkbox SHALL default to checked.

#### Scenario: Checkbox is visible on login page
- **WHEN** the user navigates to the login page
- **THEN** a "Remember me" checkbox is displayed below the password field, checked by default

#### Scenario: Checkbox is not shown during initial setup
- **WHEN** the login page is in setup mode (first-time account creation)
- **THEN** the "Remember me" checkbox SHALL NOT be displayed

### Requirement: Persistent session when remember-me is checked
When the user logs in with "Remember me" checked, the auth token SHALL be stored in `localStorage` so it persists across browser restarts.

#### Scenario: Login with remember-me checked
- **WHEN** the user logs in with "Remember me" checked
- **THEN** the auth token is stored in `localStorage`
- **THEN** closing and reopening the browser preserves the authenticated session

#### Scenario: Stale sessionStorage is cleared on persistent login
- **WHEN** the user logs in with "Remember me" checked
- **THEN** any existing auth data in `sessionStorage` SHALL be removed

### Requirement: Ephemeral session when remember-me is unchecked
When the user logs in with "Remember me" unchecked, the auth token SHALL be stored in `sessionStorage` so it is cleared when the browser tab/window closes.

#### Scenario: Login without remember-me
- **WHEN** the user logs in with "Remember me" unchecked
- **THEN** the auth token is stored in `sessionStorage`
- **THEN** closing the browser tab ends the session

#### Scenario: Stale localStorage is cleared on ephemeral login
- **WHEN** the user logs in with "Remember me" unchecked
- **THEN** any existing auth data in `localStorage` SHALL be removed

### Requirement: Remember-me preference is persisted
The user's last "Remember me" choice SHALL be stored in `localStorage` under the key `kh_remember_me` so the checkbox reflects their preference on next visit.

#### Scenario: Preference survives page reload
- **WHEN** the user unchecks "Remember me" and logs in
- **THEN** on next visit to the login page, the checkbox defaults to unchecked

#### Scenario: Default when no preference stored
- **WHEN** no `kh_remember_me` value exists in `localStorage`
- **THEN** the checkbox defaults to checked

### Requirement: Session restoration on page load
On application startup, the PocketBase client SHALL restore a valid auth session from whichever storage backend contains one, checking `localStorage` first, then `sessionStorage`.

#### Scenario: Restore persistent session
- **WHEN** the page loads and `localStorage` contains a valid auth token
- **THEN** the user is authenticated without needing to log in again

#### Scenario: Restore ephemeral session
- **WHEN** the page loads and `sessionStorage` contains a valid auth token (but `localStorage` does not)
- **THEN** the user is authenticated for the current tab

#### Scenario: No session to restore
- **WHEN** the page loads and neither storage backend contains a valid auth token
- **THEN** the user is redirected to the login page
