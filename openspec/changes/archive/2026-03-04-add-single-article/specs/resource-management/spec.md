## ADDED Requirements

### Requirement: Quick Add system resource
The system SHALL maintain an auto-created system resource of type `quickadd` named "Quick Add" that serves as the parent for one-off article entries. This resource SHALL be created automatically on first use or at application startup. It SHALL always be active and SHALL NOT be fetched by the scheduler.

#### Scenario: Quick Add resource auto-created
- **WHEN** the application starts and no `quickadd` resource exists
- **THEN** the system creates a resource with name "Quick Add", type "quickadd", status "healthy", and active true

#### Scenario: Quick Add resource already exists
- **WHEN** the application starts and a `quickadd` resource already exists
- **THEN** no duplicate is created

#### Scenario: Quick Add resource hidden from management
- **WHEN** the user views the resources management page
- **THEN** the Quick Add system resource is NOT listed among user-managed resources

#### Scenario: Quick Add resource visible as source filter
- **WHEN** entries exist under the Quick Add resource and the user views the feed
- **THEN** "Quick Add" appears in the source filter pills so the user can filter to see only one-off articles
