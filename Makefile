# THDWB - The Hotdog Web Browser build system.
#
# All targets are portable (Linux/macOS, native or WSL) and build the
# pure-Go engine with CGO disabled. Tests and lint run per module because
# the repository is a go.work workspace, not a single module.

git_revision := $(shell git log -1 --pretty="%H" 2>/dev/null || echo none)
git_branch   := $(shell git log -1 --pretty="%d" 2>/dev/null || echo none)
host_info    := $(shell uname -s -r -p 2>/dev/null || echo unknown)
build_date   := $(shell date 2>/dev/null || echo unknown)

ldflags := -X 'main.gitRevision=$(git_revision)' \
           -X 'main.gitBranch=$(git_branch)' \
           -X 'main.buildTime=$(build_date)' \
           -X 'main.hostInfo=$(host_info)'

# Modules that contain testable or lintable Go code.
MODULES := sauce mayo bun gg hotdog mustard profiler ketchup platform webview assets pages browser

.PHONY: build test clean lint coverage help run

build: clean
	@echo "Building THDWB - hotdog web browser (CGO_ENABLED=0)"
	CGO_ENABLED=0 go build -o bin/thdwb -ldflags "$(ldflags) -s -w" ./browser/
	@echo "Built bin/thdwb"

test:
	@echo "Testing all modules (CGO_ENABLED=0)..."
	@status=0; \
	for m in $(MODULES); do \
		echo "=== $$m ==="; \
		( cd $$m && CGO_ENABLED=0 go test ./... ) || status=1; \
	done; \
	exit $$status

clean:
	@rm -rf bin thdwb coverage.out coverage.html coverage-*.out coverage-*.html

lint:
	@echo "Linting all modules with golangci-lint..."
	@status=0; \
	for m in $(MODULES); do \
		echo "=== $$m ==="; \
		( cd $$m && golangci-lint run ./... ) || status=1; \
	done; \
	exit $$status

coverage:
	@echo "Generating per-module coverage (CGO_ENABLED=0)..."
	@CGO_ENABLED=0 go test -coverprofile=coverage.out ./sauce/... ./mayo/... ./bun/... ./gg/... ./hotdog/... ./mustard/... ./profiler/... ./ketchup/... ./platform/... ./webview/... ./assets/... ./pages/... ./browser/...
	@echo "Module coverage:"
	@for m in $(MODULES); do \
		( cd $$m && CGO_ENABLED=0 go test -cover ./... 2>/dev/null | grep -v 'no test files' | sed 's/^/  $$m: /' ); \
	done
	@echo "Combined coverage:"
	@go tool cover -func=coverage.out | tail -1
	@echo "Per-module HTML reports: coverage-<module>.html"
	@for m in $(MODULES); do \
		( cd $$m && CGO_ENABLED=0 go test -coverprofile=../coverage-$$m.out ./... >/dev/null 2>&1 \
			&& go tool cover -html=../coverage-$$m.out -o ../coverage-$$m.html 2>/dev/null \
			&& rm -f ../coverage-$$m.out ); \
	done

run:
	CGO_ENABLED=0 go run ./browser/ --settings=./settings.json

help:
	@echo "Available targets:"
	@echo "  make build    - Build the browser engine (bin/thdwb, CGO_ENABLED=0)"
	@echo "  make test     - Run all module test suites"
	@echo "  make clean    - Remove build artifacts"
	@echo "  make lint     - Run golangci-lint on every module"
	@echo "  make coverage - Generate coverage reports (combined + per module)"
	@echo "  make run      - Run the browser with ./settings.json"