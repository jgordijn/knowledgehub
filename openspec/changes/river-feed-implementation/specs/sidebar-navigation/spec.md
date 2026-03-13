## ADDED Requirements

### Requirement: Sidebar layout on desktop
The system SHALL display a fixed-width sidebar (220px) on the left side of the viewport on screens wider than 768px. The sidebar SHALL contain the app logo, version text, GitHub icon link, navigation links, source filter section, and a footer with keyboard shortcut hints.

#### Scenario: Desktop sidebar visible
- **WHEN** the app is loaded on a screen wider than 768px
- **THEN** a 220px sidebar is visible on the left with navigation links (Feed, Saved, Sources, Settings) and the main content area fills the remaining width

#### Scenario: Logo area content
- **WHEN** the sidebar is visible
- **THEN** the logo area shows "KnowledgeHub" text, the current app version (fetched from `/api/version`), and a GitHub icon linking to the repository

### Requirement: Sidebar navigation links
The system SHALL display navigation links in the sidebar: Feed (with unread badge count), Saved (with bookmarked badge count), Sources, and Settings. The currently active page SHALL be visually highlighted.

#### Scenario: Active page highlight
- **WHEN** the user is on the Feed page
- **THEN** the Feed link in the sidebar is visually highlighted as active

#### Scenario: Unread badge count
- **WHEN** there are 23 unread entries
- **THEN** the Feed nav link shows a badge with "23"

#### Scenario: Bookmarked badge count
- **WHEN** there are 4 bookmarked entries
- **THEN** the Saved nav link shows a badge with "4"

### Requirement: Mobile sidebar as slide-out drawer
The system SHALL hide the sidebar off-screen on viewports ≤768px and show a hamburger menu button. Tapping the hamburger SHALL slide the sidebar in from the left with a semi-transparent backdrop overlay. Tapping the overlay or a navigation link SHALL close the sidebar.

#### Scenario: Mobile hamburger button
- **WHEN** the app is loaded on a screen ≤768px wide
- **THEN** the sidebar is hidden and a hamburger (☰) button is visible in the topbar

#### Scenario: Open mobile sidebar
- **WHEN** the user taps the hamburger button on mobile
- **THEN** the sidebar slides in from the left and a semi-transparent overlay covers the main content

#### Scenario: Close mobile sidebar via overlay
- **WHEN** the mobile sidebar is open and the user taps the overlay
- **THEN** the sidebar slides back off-screen and the overlay disappears

#### Scenario: Close mobile sidebar via navigation
- **WHEN** the mobile sidebar is open and the user taps a navigation link
- **THEN** the sidebar closes and the user navigates to the selected page

### Requirement: Theme toggle in sidebar
The system SHALL display a theme toggle button in the topbar area (desktop) or sidebar that allows switching between light, dark, and system theme modes.

#### Scenario: Toggle theme
- **WHEN** the user clicks the theme toggle
- **THEN** the theme cycles to the next mode and the UI updates immediately
