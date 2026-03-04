## Why

Fragment feeds currently always use a heuristic `<p>`-based splitting algorithm (optionally refined by AI). Some fragment sources like aishepherd.nl/moments use explicit separators (e.g. `~~~`) between entries. The heuristic splitter cannot detect these, resulting in incorrect fragment boundaries. Users need the ability to specify how fragments should be split per resource.

## What Changes

- Add a `fragment_mode` field to resources: `auto` (current heuristic behavior) or `separated` (split by explicit separator string)
- Add a `fragment_separator` field to resources: the separator string to use when mode is `separated`
- Add a `SplitFragmentsBySeparator` function that splits HTML content by a text separator
- Update the fetcher to use separator-based splitting when configured
- Update the ResourceForm UI to show fragment mode and separator options when "Fragment feed" is checked
- Pass the new fields through the resource edit flow in the resources page

## Capabilities

### New Capabilities

_(none — this extends existing capabilities)_

### Modified Capabilities

- `content-fetching`: Fragment splitting now supports separator-based mode in addition to the heuristic/AI mode
- `resource-management`: Resource form gains fragment mode and separator configuration fields

## Impact

- **Backend**: `internal/engine/fragment.go` (new `SplitFragmentsBySeparator` function), `internal/engine/fetcher.go` (mode dispatch), `cmd/knowledgehub/collections.go` (new fields + migration)
- **Frontend**: `ui/src/lib/components/ResourceForm.svelte` (mode selector + separator input), `ui/src/routes/resources/+page.svelte` (pass new props)
- **Database**: Two new fields on `resources` collection: `fragment_mode` (select) and `fragment_separator` (text)
- **Tests**: `internal/engine/fragment_test.go`, `internal/testutil/testutil.go`
