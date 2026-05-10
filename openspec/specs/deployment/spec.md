## ADDED Requirements

### Requirement: Single binary with embedded frontend
The system SHALL compile to a single Go binary with all SvelteKit static files embedded via `//go:embed`. No external files are required to run the application.

#### Scenario: Run binary without additional files
- **WHEN** the knowledgehub binary is copied to a clean system and executed
- **THEN** the web UI is served from the embedded static files and the application is fully functional

### Requirement: SQLite database auto-creation
The system SHALL automatically create the SQLite database file and run migrations on first startup. The database location SHALL default to `./kh_data/` relative to the binary, configurable via environment variable.

#### Scenario: First startup
- **WHEN** the binary is run for the first time with no existing database
- **THEN** a `kh_data/` directory is created with the SQLite database and all tables/collections initialized

#### Scenario: Custom data directory
- **WHEN** the binary is run with `KH_DATA_DIR=/var/lib/knowledgehub`
- **THEN** the database is created in `/var/lib/knowledgehub/`

### Requirement: Systemd service support
The project SHALL include a systemd service unit file for running knowledgehub as a system service on Linux.

#### Scenario: Install as systemd service
- **WHEN** the service file is installed and enabled
- **THEN** knowledgehub starts on boot, runs as a dedicated user, and restarts on failure

### Requirement: Configurable listen address
The system SHALL listen on `:8090` by default, configurable via command-line flag or environment variable.

#### Scenario: Default port
- **WHEN** the binary is run without port configuration
- **THEN** it listens on `0.0.0.0:8090`

#### Scenario: Custom port
- **WHEN** the binary is run with `--http 0.0.0.0:3000`
- **THEN** it listens on port 3000

### Requirement: Build with justfile
The project SHALL include a justfile with recipes for building the frontend with Bun, building the Go binary, running tests, serving the development app, cleaning build artifacts, and creating a release tarball. The project SHALL NOT require or provide a Makefile as the supported task-runner interface.

#### Scenario: Build release locally
- **WHEN** `just release` is run
- **THEN** it builds the SvelteKit frontend with Bun, compiles the Go binary with embedded files, and produces a tarball containing the binary, systemd service file, updater script, updater service file, and updater timer file

#### Scenario: Build application locally
- **WHEN** `just build` is run
- **THEN** it builds the SvelteKit frontend with Bun, copies the static build into the embedded UI directory, and compiles the Go binary into the build directory

#### Scenario: Run tests locally
- **WHEN** `just test` is run
- **THEN** it runs the Go internal package tests with coverage output and prints total coverage

#### Scenario: GitHub CI uses just
- **WHEN** the CI workflow runs for a pull request
- **THEN** it installs `just` and invokes justfile recipe(s) for project build/test steps instead of invoking `make` targets or duplicating replaced Makefile commands inline

#### Scenario: GitHub release uses just
- **WHEN** the release workflow runs on `main`
- **THEN** it installs `just`, invokes justfile recipe(s) to test, build, and package the release artifacts, and passes the computed release version into the binary build
