## ADDED Requirements

### Requirement: Open entry card from internal reference
The system SHALL allow internal KnowledgeHub references to open a specific entry as an in-app entry card view without navigating directly to the original article URL.

#### Scenario: Open entry from Daily News reference
- **WHEN** a user clicks a Daily News reference for a KnowledgeHub entry
- **THEN** the system opens that entry in an in-app entry card view or modal

#### Scenario: Referenced entry is unavailable
- **WHEN** a user clicks an internal reference for an entry that no longer exists or is not visible to the user
- **THEN** the system shows a clear unavailable-entry message without leaving the current page
