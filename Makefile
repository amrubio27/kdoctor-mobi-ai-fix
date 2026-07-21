# kdoctor — convenience make targets
# Works on Linux, macOS, and Windows (via GNU Make or nmake with caveats).

BIN_NAME := kdoctor
ifeq ($(OS),Windows_NT)
	BIN_NAME := kdoctor.exe
endif

GO := go
RICK_MORTY_APP ?= D:/Programacion/RickMortyApp
DETEKT_BIN ?= D:/tools/detekt.cmd

.PHONY: all
all: build

.PHONY: build
build:
	$(GO) build -o $(BIN_NAME) ./cmd/kdoctor

.PHONY: test
test:
	$(GO) test -count=1 -timeout=120s ./...

.PHONY: test-race
test-race:
	$(GO) test -race -count=1 -timeout=120s ./...

.PHONY: vet
vet:
	$(GO) vet ./...

.PHONY: fmt
fmt:
	gofmt -w .

.PHONY: fmt-check
fmt-check:
	@gofmt -l . > gofmt-drift.txt
	@if [ -s gofmt-drift.txt ]; then \
		echo "gofmt drift detected in:"; \
		cat gofmt-drift.txt; \
		rm gofmt-drift.txt; \
		exit 1; \
	fi
	@rm gofmt-drift.txt

.PHONY: tidy
tidy:
	$(GO) mod tidy

.PHONY: smoke
smoke: build
	$(GO) test -race -run TestEval ./scripts/evalprojects/...

.PHONY: e2e-rick-morty
e2e-rick-morty: build
	./$(BIN_NAME) scan --type=kmp --prefer-standalone \
		--detekt-bin=$(DETEKT_BIN) \
		--project-dir=$(RICK_MORTY_APP)

.PHONY: doctor
doctor: build
	./$(BIN_NAME) doctor

.PHONY: rules
rules: build
	./$(BIN_NAME) rules

.PHONY: clean
clean:
	$(RM) $(BIN_NAME)
	$(GO) clean -testcache

.PHONY: dashboard-build
dashboard-build:
	cd dashboard && npm ci && npm run build

.PHONY: dashboard-dev
dashboard-dev:
	cd dashboard && npm run dev

.PHONY: lint
lint: fmt-check vet

.PHONY: ci
ci: lint test-race build smoke

.PHONY: help
help:
	@echo "Available targets:"
	@echo "  build              Build the kdoctor binary"
	@echo "  test               Run the full Go test suite"
	@echo "  test-race          Run tests with the race detector (requires cgo)"
	@echo "  vet                Run go vet ./..."
	@echo "  fmt                Format all Go files with gofmt"
	@echo "  fmt-check          Fail if any Go file is not gofmt-formatted"
	@echo "  tidy               Run go mod tidy"
	@echo "  smoke              Run fixture smoke tests (requires detekt)"
	@echo "  e2e-rick-morty     Run kdoctor against RickMortyApp (requires detekt)"
	@echo "  doctor             Run kdoctor doctor"
	@echo "  rules              List the kdoctor rule catalog"
	@echo "  clean              Remove built binary and test cache"
	@echo "  dashboard-build    Build the HTML dashboard"
	@echo "  dashboard-dev      Start the dashboard dev server"
	@echo "  lint               Alias for fmt-check + vet"
	@echo "  ci                 Run lint, test, build, and smoke (CI gate)"
