## 1. Auth Store

- [x] 1.1 Create `ui/src/lib/auth-store.ts` — custom `AsyncAuthStore` that reads from localStorage/sessionStorage on init and writes to the active backend based on remember-me preference
- [x] 1.2 Update `ui/src/lib/pb.ts` — instantiate PocketBase with the custom auth store

## 2. Login Form

- [x] 2.1 Add `rememberMe` state variable to login page, default from `localStorage.getItem('kh_remember_me')` (default `true`)
- [x] 2.2 Add "Remember me" checkbox UI between password field and submit button (hidden during setup mode)
- [x] 2.3 On login: save `kh_remember_me` preference to localStorage, call auth store to switch backend, clear the other storage

## 3. Verification

- [x] 3.1 Build frontend (`bun run build` in `ui/`) and verify no build errors
