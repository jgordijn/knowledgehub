APP_NAME := knowledgehub
BUILD_DIR := ./build
CMD_DIR := ./cmd/knowledgehub

.PHONY: build dev release clean test ui

ui:
	cd ui && bun install && bun run build
	rm -rf $(CMD_DIR)/ui/build
	cp -r ui/build $(CMD_DIR)/ui/build

build: ui
	CGO_ENABLED=1 go build -o $(BUILD_DIR)/$(APP_NAME) $(CMD_DIR)

dev:
	go run $(CMD_DIR) serve

release: ui
	CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o $(BUILD_DIR)/$(APP_NAME) $(CMD_DIR)
	tar czf $(BUILD_DIR)/$(APP_NAME)-linux-amd64.tar.gz -C $(BUILD_DIR) $(APP_NAME) -C .. knowledgehub.service

clean:
	rm -rf $(BUILD_DIR)
	rm -rf ui/build
	rm -rf $(CMD_DIR)/ui/build

test:
	go test ./internal/... -coverprofile=coverage.out -covermode=atomic
	go tool cover -func=coverage.out | grep total
