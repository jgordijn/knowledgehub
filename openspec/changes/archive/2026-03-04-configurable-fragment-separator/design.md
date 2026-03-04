## Context

Fragment feeds currently use a single splitting strategy: a heuristic that splits HTML on `<p>` elements, optionally refined by an AI call. This works for blog-like content but fails for pages that use explicit text separators (e.g. `~~~`) to delimit entries. The `resources` collection has a `fragment_feed` boolean but no configuration for how splitting should work.

## Goals / Non-Goals

**Goals:**
- Allow users to configure per-resource fragment splitting strategy: automatic (current heuristic + AI) or separator-based
- When separator mode is chosen, split content by the specified separator string
- Keep the default behavior unchanged for existing fragment feeds (backward compatible)
- Clean UI: only show separator options when fragment feed is enabled and mode is "separated"

**Non-Goals:**
- Regex-based separators — plain string matching is sufficient
- Auto-detection of separators — the user specifies them explicitly
- Changing how non-fragment feeds work

## Decisions

### 1. Two new fields on `resources`: `fragment_mode` and `fragment_separator`

**Decision**: Add `fragment_mode` as a select field (`auto` | `separated`) and `fragment_separator` as a text field.

**Rationale**: A select field makes the mode explicit and extensible. A separate text field for the separator keeps concerns clean. When `fragment_mode` is empty or `auto`, existing behavior is preserved (backward compatible with existing records where these fields are absent).

**Alternative considered**: Single `fragment_separator` field where empty = auto. Rejected because it conflates "not configured" with "auto mode" and doesn't allow future mode additions.

### 2. Separator splitting operates on rendered text, not raw HTML

**Decision**: When splitting by separator, extract text from the HTML, find separator positions in the text, then map those positions back to the original HTML DOM nodes. Practically: use goquery to iterate `body > *` children, extract each element's text, and split into groups separated by elements whose text content matches the separator.

**Rationale**: Separators like `~~~` appear as text content within `<p>` tags in the rendered HTML. Splitting raw HTML by string would break tags. Walking the DOM and grouping elements between separator-containing elements is clean and robust.

### 3. Separator elements are discarded

**Decision**: DOM elements whose trimmed text matches the separator exactly are discarded (not included in any fragment). This is the same approach used for `<hr>` in the heuristic splitter.

### 4. No AI regrouping for separator mode

**Decision**: When mode is `separated`, skip the AI regrouping step. The separators are the authoritative boundaries.

**Rationale**: The AI regrouping exists because the heuristic splitter's `<p>`-based boundaries are approximate. With explicit separators, boundaries are exact and AI regrouping would be wasteful.

### 5. Migration via `addFieldIfMissing`

**Decision**: Use the existing `migrateCollections` pattern with `addFieldIfMissing`.

**Rationale**: Consistent with how all other schema migrations work in this project. Existing resources get empty values for the new fields, which maps to `auto` behavior.

## Risks / Trade-offs

- **[Risk]** Separator might appear inside content, not just between fragments → **Mitigation**: Match only when the trimmed text of an entire DOM element equals the separator exactly. Partial matches within paragraphs are ignored.
- **[Risk]** Resources created before this change have no `fragment_mode` value → **Mitigation**: Empty/missing `fragment_mode` defaults to `auto` behavior in the fetcher.
