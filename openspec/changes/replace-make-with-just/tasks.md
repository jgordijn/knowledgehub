## 1. Baseline and Parity

- [x] 1.1 Inventory current Makefile targets and all repository references to `make`/`Makefile`.
- [x] 1.2 Capture expected outputs for current build, test, and release commands so justfile parity can be verified.

## 2. Justfile Implementation

- [x] 2.1 Add a root `justfile` with variables for app name, build directory, and command directory.
- [x] 2.2 Implement `ui`, `build`, `dev`, `release`, `clean`, and `test` recipes equivalent to the current Makefile targets.
- [x] 2.3 Make the release recipe accept a version value for `-X main.version=...` while still supporting local `just release` without an explicit version.
- [x] 2.4 Remove the root `Makefile` after justfile parity is in place.

## 3. GitHub Workflow Updates

- [x] 3.1 Update `.github/workflows/ci.yml` to install `just` and invoke justfile recipe(s) for frontend/embed and test steps.
- [x] 3.2 Update `.github/workflows/release.yml` to install `just`, invoke justfile recipe(s) for test/build/package steps, and pass the computed release version to the release build.
- [x] 3.3 Verify workflow artifact paths remain `build/knowledgehub` and `build/knowledgehub-linux-amd64.tar.gz`.

## 4. Documentation Updates

- [x] 4.1 Update README and project documentation examples from `make ...` to `just ...`.
- [x] 4.2 Update agent/project instructions that reference Makefile or make commands.
- [x] 4.3 Search for remaining `make ` and `Makefile` mentions and either update or explicitly justify intentional historical references.

## 5. Verification

- [x] 5.1 Run `just test` and confirm tests pass with coverage output.
- [x] 5.2 Run `just build` and confirm the frontend is embedded and the local binary is produced.
- [ ] 5.3 Run `just release` with and without an explicit version value and confirm the tarball contents match the expected release package.
- [ ] 5.4 Run `openspec validate replace-make-with-just --strict` and resolve any proposal/spec issues.
