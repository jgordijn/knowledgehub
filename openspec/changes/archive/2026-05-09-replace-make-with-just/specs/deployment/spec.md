## MODIFIED Requirements

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
