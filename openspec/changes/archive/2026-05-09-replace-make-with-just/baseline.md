# Parity Baseline

## Current Makefile targets

- `ui`: `cd ui && bun install && bun run build`; replace `cmd/knowledgehub/ui/build` with `ui/build`.
- `build`: depends on `ui`; `CGO_ENABLED=1 go build -o ./build/knowledgehub ./cmd/knowledgehub`.
- `dev`: `go run ./cmd/knowledgehub serve`.
- `release`: depends on `ui`; `CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o ./build/knowledgehub ./cmd/knowledgehub`; creates `./build/knowledgehub-linux-amd64.tar.gz` containing `knowledgehub`, `knowledgehub.service`, `knowledgehub-updater.sh`, `knowledgehub-updater.service`, and `knowledgehub-updater.timer`.
- `clean`: removes `./build`, `ui/build`, and `./cmd/knowledgehub/ui/build`.
- `test`: runs `go test ./internal/... -coverprofile=coverage.out -covermode=atomic`; prints total coverage with `go tool cover -func=coverage.out | grep total`.

## References inventory

User-facing `make`/`Makefile` references requiring updates:
- `README.md`: `make build`, `make release`, `make test`, `make dev`.
- `AGENTS.md`: project tree and local command examples.
- `openspec/specs/deployment/spec.md`: current accepted deployment spec will be updated when this change is archived; do not edit during implementation.

Intentional/historical/code references not requiring user-facing command updates:
- This change's OpenSpec artifacts mention the migration from Makefile to justfile.
- Archived OpenSpec changes preserve historical Makefile references.
- Go/Svelte source occurrences of `make` are language built-ins or prose unrelated to task runner usage.

## Expected outputs to preserve

- Build binary: `build/knowledgehub`.
- Release tarball: `build/knowledgehub-linux-amd64.tar.gz`.
- Release tarball members: `knowledgehub`, `knowledgehub.service`, `knowledgehub-updater.sh`, `knowledgehub-updater.service`, `knowledgehub-updater.timer`.
- Embedded frontend output directory: `cmd/knowledgehub/ui/build` copied from `ui/build`.
- Test coverage file: `coverage.out`, with total coverage printed.
