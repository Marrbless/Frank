GOLANGCI_LINT ?= golangci-lint
GOLANGCI_LINT_VERSION ?= v2.11.1
GOLANGCI_LINT_CACHE ?= /tmp/picobot-golangci-cache
COVERPKGS ?= ./...
COVERAGE_DIR ?= /tmp/picobot-coverage
COVERAGE_PROFILE ?= $(COVERAGE_DIR)/coverage.out
COVERAGE_FUNC ?= $(COVERAGE_DIR)/coverage.func.txt
RACEPKGS ?= ./internal/cron ./internal/session ./internal/channels
BUILD_TAG_CONTRACT_DIR ?= /tmp/picobot-build-tags
BUILD_TAG_GOCACHE ?= /tmp/picobot-go-cache

.PHONY: build build-all test test-lite test-build-tags test-scripts test-race coverage docker-smoke vet lint lint-version install-lint verify linux_amd64 linux_arm64 mac_arm64 linux_amd64_lite linux_arm64_lite mac_arm64_lite android_arm64_lite clean

build: build-all

test:
	go test -count=1 ./...

test-lite:
	go test -count=1 -tags lite ./...

test-build-tags:
	mkdir -p "$(BUILD_TAG_CONTRACT_DIR)" "$(BUILD_TAG_GOCACHE)"
	GOCACHE="$(BUILD_TAG_GOCACHE)" go test -count=1 -run TestWhatsAppBuildContract ./internal/channels
	GOCACHE="$(BUILD_TAG_GOCACHE)" go test -count=1 -tags lite -run TestWhatsAppBuildContract ./internal/channels
	GOCACHE="$(BUILD_TAG_GOCACHE)" CGO_ENABLED=0 go build -buildvcs=false -o "$(BUILD_TAG_CONTRACT_DIR)/picobot-full" ./cmd/picobot
	GOCACHE="$(BUILD_TAG_GOCACHE)" CGO_ENABLED=0 go build -buildvcs=false -tags lite -o "$(BUILD_TAG_CONTRACT_DIR)/picobot-lite" ./cmd/picobot

test-scripts:
	sh scripts/termux/test-update-and-restart-frank
	sh scripts/test-release-checksums
	sh scripts/check-env-examples
	sh scripts/check-doc-links
	sh scripts/check-doc-snippets

test-race:
	go test -race -count=1 $(RACEPKGS)

coverage:
	mkdir -p "$(COVERAGE_DIR)"
	go test -count=1 -covermode=atomic -coverprofile="$(COVERAGE_PROFILE)" $(COVERPKGS)
	go tool cover -func="$(COVERAGE_PROFILE)" > "$(COVERAGE_FUNC)"
	tail -n 1 "$(COVERAGE_FUNC)"
	@echo "coverage profile: $(COVERAGE_PROFILE)"
	@echo "coverage functions: $(COVERAGE_FUNC)"

docker-smoke:
	sh scripts/check-docker-smoke

vet:
	go vet ./...

lint:
	GOLANGCI_LINT_CACHE="$(GOLANGCI_LINT_CACHE)" $(GOLANGCI_LINT) run

lint-version:
	@echo "$(GOLANGCI_LINT_VERSION)"

install-lint:
	GOBIN="$${GOBIN:-$$(go env GOPATH)/bin}" go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION)

verify: vet test test-lite test-build-tags test-scripts lint

build-all: linux_amd64 linux_arm64 mac_arm64 linux_amd64_lite linux_arm64_lite mac_arm64_lite
		@echo "All builds completed."

linux_amd64:
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags="-s -w" -o picobot_linux_amd64 ./cmd/picobot

linux_arm64:
	GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -ldflags="-s -w" -o picobot_linux_arm64 ./cmd/picobot

mac_arm64:
	GOOS=darwin GOARCH=arm64 CGO_ENABLED=0 go build -ldflags="-s -w" -o picobot_mac_arm64 ./cmd/picobot

linux_amd64_lite:
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags="-s -w" -tags lite -o picobot_linux_amd64_lite ./cmd/picobot

linux_arm64_lite:
	GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -ldflags="-s -w" -tags lite -o picobot_linux_arm64_lite ./cmd/picobot

mac_arm64_lite:
	GOOS=darwin GOARCH=arm64 CGO_ENABLED=0 go build -ldflags="-s -w" -tags lite -o picobot_mac_arm64_lite ./cmd/picobot

android_arm64_lite:
	GOOS=android GOARCH=arm64 CGO_ENABLED=0 go build -ldflags="-s -w" -tags lite -o picobot_android_arm64_lite ./cmd/picobot

clean:
	rm -f picobot_linux_amd64 picobot_linux_arm64 picobot_mac_arm64 picobot_linux_amd64_lite picobot_linux_arm64_lite picobot_mac_arm64_lite picobot_android_arm64_lite
