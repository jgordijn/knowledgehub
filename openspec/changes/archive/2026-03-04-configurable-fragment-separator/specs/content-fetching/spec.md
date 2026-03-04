## ADDED Requirements

### Requirement: Separator-based fragment splitting
The system SHALL support splitting fragment feed content by an explicit text separator. When a resource has `fragment_mode` set to `separated` and a non-empty `fragment_separator`, the system SHALL split content by finding DOM elements whose trimmed text content exactly matches the separator string, using those as boundaries. Separator elements SHALL be discarded. Each group of elements between separators becomes a separate fragment.

#### Scenario: Split by separator
- **WHEN** a fragment feed resource has fragment_mode "separated" and fragment_separator "~~~", and the feed entry contains three sections of content separated by paragraphs containing only "~~~"
- **THEN** the system creates three fragment entries, one for each section, with the "~~~" separator paragraphs discarded

#### Scenario: Separator not found in content
- **WHEN** a fragment feed resource has fragment_mode "separated" and fragment_separator "~~~", but the feed entry content contains no elements matching "~~~"
- **THEN** the entire content is treated as a single fragment

#### Scenario: Separator at start or end of content
- **WHEN** a fragment feed entry starts with a "~~~" separator and ends with a "~~~" separator
- **THEN** leading and trailing empty fragments are discarded, and only non-empty content groups become fragments

### Requirement: Auto mode preserves existing behavior
The system SHALL treat resources with `fragment_mode` set to `auto` or empty/unset identically to the current heuristic + AI fragment splitting behavior.

#### Scenario: Auto mode fragment splitting
- **WHEN** a fragment feed resource has fragment_mode "auto" (or empty)
- **THEN** the system uses the heuristic paragraph-based splitter with optional AI regrouping, identical to current behavior

### Requirement: Separator mode skips AI regrouping
The system SHALL NOT invoke AI regrouping when fragment_mode is `separated`. The separator boundaries are authoritative.

#### Scenario: No AI call for separated mode
- **WHEN** a fragment feed resource has fragment_mode "separated" and fragment_separator "---"
- **THEN** the system splits by the separator without making any AI calls for regrouping
