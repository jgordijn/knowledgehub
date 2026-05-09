## Why

The project currently exposes common developer and release commands through `make`, while the desired workflow is to standardize on `just` for clearer command definitions, argument handling, and cross-platform developer ergonomics. Replacing the Makefile with a justfile also requires aligning CI and release automation so local and GitHub workflows invoke the same task runner.

## What Changes

- Remove the root `Makefile` as the supported task runner entry point.
- Add a root `justfile` that provides equivalent commands for development, testing, building, release packaging, cleaning, and any documented helper tasks currently exposed by `make`.
- Update project documentation to use `just` commands instead of `make` commands.
- Update GitHub Actions workflows to install/use `just` and call the justfile recipes for CI and release build steps where project task-runner commands are needed.
- **BREAKING**: Developers and automation must use `just <recipe>` instead of `make <target>` after this change.

## Capabilities

### New Capabilities

### Modified Capabilities
- `deployment`: Build and release task-runner requirements change from Makefile targets to justfile recipes, including GitHub workflow compatibility.

## Impact

- Affected files include `Makefile`, new `justfile`, `.github/workflows/*.yml`, `README.md`, `AGENTS.md`, and any other documentation or scripts that mention `make` commands.
- CI and release workflows must remain functionally equivalent after switching to `just`.
- No runtime application APIs or database schemas are expected to change.
