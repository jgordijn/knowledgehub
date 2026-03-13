## ADDED Requirements

### Requirement: Section headers for each tier
The system SHALL display a section header above each entry tier group. The header SHALL show the tier label: "Featured", "High Priority", "Worth a Look" (with "— click ▸ to expand" hint), and "Low Priority" (with "— click ▸ to expand" hint).

#### Scenario: Section header display
- **WHEN** the feed contains entries in the High Priority tier
- **THEN** a section header labeled "High Priority" appears above the HP entries

#### Scenario: Section header with expand hint
- **WHEN** the Worth a Look section is displayed
- **THEN** the section header shows "Worth a Look — click ▸ to expand"

### Requirement: Expand all button per section
The system SHALL display an "Expand all" button in each section header when at least one entry in that section is in collapsed state. Clicking it SHALL expand all collapsed entries in that section. The button SHALL be hidden when all entries in the section are already expanded.

#### Scenario: Expand all visible when entries collapsed
- **WHEN** the Low Priority section has 3 entries, 2 of which are collapsed
- **THEN** the "Expand all" button is visible in the LP section header

#### Scenario: Expand all expands all collapsed entries
- **WHEN** the user clicks "Expand all" on the Worth a Look section with 2 collapsed entries
- **THEN** all 2 entries expand, showing their detail panels

#### Scenario: Expand all hidden when all expanded
- **WHEN** all entries in the Featured section are expanded
- **THEN** the "Expand all" button is not visible in the Featured section header

### Requirement: Collapse all button per section
The system SHALL display a "Collapse all" button in each section header when at least one entry in that section is in expanded state. Clicking it SHALL collapse all expanded entries in that section. The button SHALL be hidden when all entries in the section are already collapsed.

#### Scenario: Collapse all visible when entries expanded
- **WHEN** the High Priority section has 2 entries, both expanded
- **THEN** the "Collapse all" button is visible in the HP section header

#### Scenario: Collapse all collapses all expanded entries
- **WHEN** the user clicks "Collapse all" on the Featured section with 1 expanded entry
- **THEN** the entry collapses, hiding its summary and action buttons

#### Scenario: Collapse all hidden when all collapsed
- **WHEN** all entries in the Low Priority section are collapsed
- **THEN** the "Collapse all" button is not visible in the LP section header

### Requirement: Progressive disclosure of batch buttons
The system SHALL show both "Expand all" and "Collapse all" buttons simultaneously when a section contains a mix of expanded and collapsed entries. When all entries share the same state, only the opposite action button SHALL be visible.

#### Scenario: Mixed state shows both buttons
- **WHEN** the High Priority section has 1 expanded and 1 collapsed entry
- **THEN** both "Expand all" and "Collapse all" buttons are visible

#### Scenario: All expanded shows only collapse
- **WHEN** all 3 entries in the Worth a Look section are expanded
- **THEN** only "Collapse all" is visible; "Expand all" is hidden

#### Scenario: All collapsed shows only expand
- **WHEN** all 3 entries in the Low Priority section are collapsed
- **THEN** only "Expand all" is visible; "Collapse all" is hidden
