## Context

KnowledgeHub currently keeps local task automation in a root `Makefile` with targets for frontend build, backend build, dev server, release packaging, clean, and tests. GitHub Actions currently duplicate several of those commands inline rather than invoking the Makefile. The change should make `just` the single supported task runner while keeping the existing build, test, and release outputs equivalent.

## Goals / Non-Goals

**Goals:**
- Provide a root `justfile` with recipes equivalent to the current Makefile targets.
- Update CI and release workflows so GitHub Actions install `just` and use the justfile for project build/test/release operations.
- Preserve current release artifact contents, binary naming, embedded frontend behavior, and version injection in release builds.
- Update project and agent-facing documentation that tells contributors to use `make`.
- Remove the Makefile once parity is established.

**Non-Goals:**
- Changing application runtime behavior, ports, systemd units, updater behavior, or deployment topology.
- Reworking frontend package management away from Bun.
- Reworking Go test coverage thresholds beyond preserving current test commands.
- Adding a second supported task-runner path; `make` compatibility is intentionally removed.

## Decisions

1. Use a root `justfile` as the canonical command surface.
   - Recipes should map directly from existing targets: `ui`, `build`, `dev`, `release`, `clean`, and `test`.
   - Variables such as app name, build directory, and command directory should be declared at the top of the justfile for readability.
   - Alternative considered: keep Makefile as a shim that calls `just`. This was rejected because the goal is to remove make rather than maintain two entry points.

2. Let workflows call justfile recipes instead of duplicating project build commands.
   - CI should install `just`, run the frontend/embed recipe as needed, and run tests through `just` or dedicated CI recipe(s).
   - Release should install `just` and use a release recipe that supports passing the computed version into Go linker flags.
   - Alternative considered: keep inline workflow commands while only changing local development. This was rejected because the request explicitly includes GitHub workflows and because duplicated automation diverges over time.

3. Add a version-aware release recipe rather than hard-coding release linker flags.
   - The release workflow computes the tag, so the justfile should expose a way to pass that tag into the release binary build, for example via a `VERSION` environment variable or recipe argument.
   - Local `just release` should still work with an empty/default version if no version is provided.

4. Keep installation of `just` workflow-local.
   - GitHub Actions should install `just` using a maintained action or package manager before invoking recipes.
   - The repository does not need to vendor `just` or add it to the built application.

## Risks / Trade-offs

- [Risk] GitHub hosted runners do not include `just` by default → Mitigation: add an explicit setup/install step in every workflow job that invokes the justfile.
- [Risk] Release artifacts may change if the tar command or build paths differ → Mitigation: preserve the current tarball contents and binary output paths exactly and verify with tests or a local release command.
- [Risk] Workflow version injection may be lost when moving build commands into `just` → Mitigation: design the release recipe to accept `VERSION` and have the workflow pass `${{ steps.version.outputs.new_tag }}`.
- [Risk] Documentation references to `make` may remain stale → Mitigation: search the repository for `make ` and `Makefile` references during implementation and update intentional user-facing references.
