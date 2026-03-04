## 1. Database Schema

- [x] 1.1 Add `fragment_mode` (select: `auto`, `separated`) field to resources collection in `collections.go` migration
- [x] 1.2 Add `fragment_separator` (text) field to resources collection in `collections.go` migration

## 2. Backend — Fragment Splitting

- [x] 2.1 Add `SplitFragmentsBySeparator(html, separator string) []Fragment` function in `internal/engine/fragment.go`
- [x] 2.2 Update `fetchRSSResource` in `internal/engine/fetcher.go` to dispatch between auto and separator modes based on resource `fragment_mode`
- [x] 2.3 Write tests for `SplitFragmentsBySeparator` in `internal/engine/fragment_test.go`: basic split, separator not found, leading/trailing separators, empty groups

## 3. Test Utilities

- [x] 3.1 Update `testutil.CreateResource` (if needed) to support the new `fragment_mode` and `fragment_separator` fields

## 4. Frontend — ResourceForm

- [x] 4.1 Add `initialFragmentMode` and `initialFragmentSeparator` props to `ResourceForm.svelte`
- [x] 4.2 Add mode selector (auto/separated) visible when `fragmentFeed` is checked and type is `rss`
- [x] 4.3 Add separator text input visible when mode is `separated`
- [x] 4.4 Include `fragment_mode` and `fragment_separator` in the form submit data

## 5. Frontend — Resources Page

- [x] 5.1 Pass `initialFragmentMode` and `initialFragmentSeparator` to ResourceForm in the edit flow
- [x] 5.2 Update the fragment badge to show separator value when mode is `separated`
