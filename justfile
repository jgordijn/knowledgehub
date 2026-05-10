app_name := "knowledgehub"
build_dir := "./build"
cmd_dir := "./cmd/knowledgehub"

_default:
    @just --list

ui:
    cd ui && bun install && bun run build
    rm -rf {{cmd_dir}}/ui/build
    mkdir -p {{cmd_dir}}/ui
    cp -r ui/build {{cmd_dir}}/ui/build
    touch {{cmd_dir}}/ui/build/.gitkeep

build: ui
    mkdir -p {{build_dir}}
    CGO_ENABLED=1 go build -o {{build_dir}}/{{app_name}} {{cmd_dir}}

dev:
    go run {{cmd_dir}} serve

release version="": ui
    mkdir -p {{build_dir}}
    if [ -n "{{version}}" ]; then \
        ldflags="-s -w -X main.version={{version}}"; \
    else \
        ldflags="-s -w"; \
    fi; \
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="$ldflags" -o {{build_dir}}/{{app_name}} {{cmd_dir}}
    tar czf {{build_dir}}/{{app_name}}-linux-amd64.tar.gz -C {{build_dir}} {{app_name}} -C .. knowledgehub.service knowledgehub-updater.sh knowledgehub-updater.service knowledgehub-updater.timer

clean:
    rm -rf {{build_dir}}
    rm -rf ui/build
    rm -rf {{cmd_dir}}/ui/build
    mkdir -p {{cmd_dir}}/ui/build
    touch {{cmd_dir}}/ui/build/.gitkeep

test:
    go test ./internal/... -count=1 -coverprofile=coverage.out -covermode=atomic
    go tool cover -func=coverage.out | grep total
