NAME = micro
GIT_COMMIT = $(shell git rev-parse --short HEAD)
GIT_TAG = $(shell git describe --abbrev=0 --tags --always --match "v*")
GIT_IMPORT = go-micro.dev/v5/cmd/micro
BUILD_DATE = $(shell date +%s)
LDFLAGS = -X $(GIT_IMPORT).BuildDate=$(BUILD_DATE) -X $(GIT_IMPORT).GitCommit=$(GIT_COMMIT) -X $(GIT_IMPORT).GitTag=$(GIT_TAG)

# GORELEASER_DOCKER_IMAGE = ghcr.io/goreleaser/goreleaser-cross:v1.25.7
GORELEASER_DOCKER_IMAGE = ghcr.io/goreleaser/goreleaser:latest

.PHONY: test test-race test-coverage harness inner-loop cli-wayfinding docs-wayfinding install-smoke provider-conformance-mock provider-conformance lint fmt install-tools proto clean help gorelease-dry-run gorelease-dry-run-docker

# Default target
help:
	@echo "Go Micro Development Tasks"
	@echo ""
	@echo "  make test          - Run tests"
	@echo "  make test-race     - Run tests with race detector"
	@echo "  make test-coverage - Run tests with coverage"
	@echo "  make lint          - Run linter"
	@echo "  make harness       - Run deterministic getting-started and end-to-end harnesses"
	@echo "  make inner-loop    - Verify scaffold → run/chat/inspect → deploy dry-run contract"
	@echo "  make cli-wayfinding - Verify installed first-agent CLI wayfinding commands"
	@echo "  make docs-wayfinding - Verify first-agent docs wayfinding links resolve locally"
	@echo "  make install-smoke - Verify the local install.sh and first-run CLI smoke path"
	@echo "  make provider-conformance-mock - Run cross-provider harness with deterministic mock provider"
	@echo "  make provider-conformance - Run harnesses against configured live providers"
	@echo "  make fmt           - Format code"
	@echo "  make install-tools - Install development tools"
	@echo "  make proto         - Generate protobuf code"
	@echo "  make clean         - Clean build artifacts"

$(NAME):
	CGO_ENABLED=0 go build -ldflags "-s -w ${LDFLAGS}" -o $(NAME) cmd/micro/main.go

# Run tests
test:
	go test -v ./...

# Run tests with race detector
test-race:
	go test -v -race ./...

# Run tests with coverage
test-coverage:
	go test -v -race -coverprofile=coverage.out -covermode=atomic ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

# Run the documented getting-started contracts plus the deterministic
# services → agents → workflows harnesses (mock LLM — no API key).
# This mirrors the default CI path so local dogfooding catches scaffold,
# run/chat/inspect, and 0→hero regressions before a PR is opened.
harness:
	$(MAKE) cli-wayfinding
	$(MAKE) inner-loop
	./internal/harness/zero-to-hero-ci/run.sh
	go run ./internal/harness/agent-flow
	$(MAKE) provider-conformance-mock

# Focused provider-free CLI inner-loop contract: scaffold a service, keep the
# run/chat/inspect commands discoverable, and prove deploy dry-run reaches the
# documented boundary without remote side effects. Use this when README/docs/CLI
# drift is the concern and the full runtime harness is more than you need.
inner-loop:
	go test ./cmd/micro/cli/new -run TestZeroToOne -count=1
	go test ./cmd/micro -run 'TestFirstAgentWalkthroughCLIBoundaries|TestZeroToHeroCLIBoundaries|TestZeroToHeroCommandPrintsMaintainedNoSecretPath' -count=1
	go test ./cmd/micro/cli/deploy -run TestDeployDryRun -count=1
	go test ./internal/harness/zero-to-hero-ci -run 'TestZeroToHeroDeployDryRunCommandSmoke|TestNoSecretFirstAgentDebuggingSmoke|TestYourFirstAgentTutorialSmoke' -count=1

# Verify the installed CLI keeps the first-agent on-ramp commands discoverable.
# This guards the no-secret commands README/docs recommend (`micro agent demo`,
# `micro examples`, and `micro zero-to-hero`) as a CI contract.
cli-wayfinding:
	go test ./cmd/micro -run 'TestFirstAgentWalkthroughCLIBoundaries|TestExamplesWayfindingIndexStaysLinked|TestExamplesCommandPointsAtWayfindingIndex|TestZeroToHeroCommandPrintsMaintainedNoSecretPath' -count=1
	$(MAKE) docs-wayfinding
	$(MAKE) install-smoke

# Verify the README and website first-agent/0→hero wayfinding links resolve to
# maintained local docs and examples. This is a focused no-network guard for the
# developer-adoption on-ramp.
docs-wayfinding:
	go test ./internal/harness/zero-to-hero-ci -run 'TestFirstAgentWayfindingDocs|TestFirstAgentWayfindingLinkTargetsResolve' -count=1

# Verify the documented install script and first-run CLI command boundaries without
# provider keys or network access.
install-smoke:
	./internal/harness/install-smoke/run.sh

# Run the shared provider conformance contract with the deterministic mock
# provider. This is the no-secret path used by CI and local dogfooding to keep
# provider-facing agent/tool semantics covered on every machine.
provider-conformance-mock:
	go run ./internal/harness/provider-conformance -providers mock

# Run the same harnesses against every configured live provider. Providers
# without API keys are skipped; configured providers must pass.
provider-conformance:
	go run ./internal/harness/provider-conformance

# Run linter
lint:
	golangci-lint run

# Format code
fmt:
	gofmt -s -w .
	goimports -w .

# Install development tools
install-tools:
	@echo "Installing development tools..."
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	go install golang.org/x/tools/cmd/goimports@latest
	go install github.com/kyoh86/richgo@latest
	go install go-micro.dev/v5/cmd/protoc-gen-micro@latest
	@echo "Tools installed successfully"

# Generate protobuf code
proto:
	@echo "Generating protobuf code..."
	find . -name "*.proto" -not -path "./vendor/*" -exec protoc --proto_path=. --micro_out=. --go_out=. {} \;

# Clean build artifacts
clean:
	rm -f coverage.out coverage.html
	find . -name "*.test" -type f -delete
	go clean -cache -testcache

# Try binary release
gorelease-dry-run:
	docker run \
		--rm \
		-e CGO_ENABLED=0 \
		-v $(CURDIR):/$(NAME) \
		-v /var/run/docker.sock:/var/run/docker.sock \
		-w /$(NAME) \
		$(GORELEASER_DOCKER_IMAGE) \
		--clean --verbose --skip=publish,validate --snapshot
