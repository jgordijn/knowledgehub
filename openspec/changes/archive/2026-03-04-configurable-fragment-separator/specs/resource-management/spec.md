## ADDED Requirements

### Requirement: Fragment mode configuration
The system SHALL allow the user to configure the fragment splitting mode when adding or editing a fragment feed resource. The available modes SHALL be "auto" (heuristic + AI splitting) and "separated" (split by explicit separator string).

#### Scenario: Configure fragment mode as separated
- **WHEN** user checks "Fragment feed" on an RSS resource and selects mode "separated" with separator "~~~"
- **THEN** the resource is saved with fragment_mode "separated" and fragment_separator "~~~"

#### Scenario: Configure fragment mode as auto
- **WHEN** user checks "Fragment feed" on an RSS resource and leaves mode as "auto"
- **THEN** the resource is saved with fragment_mode "auto" and fragment_separator empty

#### Scenario: Fragment mode hidden when not fragment feed
- **WHEN** user unchecks "Fragment feed" or selects type "watchlist"
- **THEN** the fragment mode and separator fields are not visible

### Requirement: Fragment mode display in resource list
The system SHALL display the fragment mode in the resource list when a resource is a fragment feed with mode "separated", showing the separator value.

#### Scenario: Separated fragment badge
- **WHEN** a resource has fragment_feed true, fragment_mode "separated", and fragment_separator "~~~"
- **THEN** the resource list shows a "Fragment (~~~)" badge instead of just "Fragment"

#### Scenario: Auto fragment badge
- **WHEN** a resource has fragment_feed true and fragment_mode "auto" or empty
- **THEN** the resource list shows a "Fragment" badge (unchanged from current behavior)
