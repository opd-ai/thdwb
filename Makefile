git_revision := $(shell git log -1 --pretty="%H")
git_branch := $(shell git log -1 --pretty="%d")
host_info := $(shell uname -s -r -p)
build_date := $(shell date)

ldflags := -X 'main.gitRevision=$(git_revision)' -X 'main.gitBranch=$(git_branch)' -X 'main.buildTime=$(build_date)' -X 'main.hostInfo=$(host_info)'

.PHONY: build test clean lint coverage help run

build: clean
	@echo "Building THDWB - 🌭"
	CGO_ENABLED=0 go build -o thdwb -ldflags "$(ldflags) -s -w" ./browser/
	@chmod 755 thdwb
	@mkdir -p bin; mv thdwb bin/

test:
	@echo -e "Testing all modules...\n"
	CGO_ENABLED=0 go test -v ./sauce/... ./mayo/... ./bun/... ./gg/... ./hotdog/... ./mustard/... ./profiler/... ./ketchup/... ./platform/... ./webview/... ./assets/... ./pages/... ./browser/... | sed ''/PASS/s//$$(printf "\033[32mPASS\033[0m")/'' | sed ''/FAIL/s//$$(printf "\033[31mFAIL\033[0m")/'' | sed ''/FAIL/s//$$(printf "\033[31mFAIL\033[0m")/'' | GREP_COLOR="01;33" egrep --color=always '\s*[a-zA-Z0-9\-_.]+[:][0-9]+[:]|^'
	@echo -e "\n"

clean:
	@rm -rf bin thdwb coverage.out coverage.html

lint:
	@for m in sauce mayo bun gg hotdog mustard profiler ketchup platform webview assets pages browser; do \
		echo "=== Linting $$m ==="; \
		( cd $$m && golangci-lint run ./... ) || exit 1; \
	done

coverage:
	CGO_ENABLED=0 go test -cover ./sauce/... ./mayo/... ./bun/... ./gg/... ./hotdog/... ./mustard/... ./profiler/... -coverprofile=coverage.out
	go tool cover -html=coverage.out

run:
	CGO_ENABLED=0 go run ./browser/ --settings="./settings.json"

help:
	@echo "Available targets:"
	@echo "  make build    - Build the browser engine"
	@echo "  make test     - Run all tests"
	@echo "  make clean    - Remove build artifacts"
	@echo "  make lint     - Run linter"
	@echo "  make coverage - Generate coverage report"
	@echo "  make run      - Run the application"