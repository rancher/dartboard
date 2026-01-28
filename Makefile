DARTBOARD_BIN_NAME := dartboard
REPORTER_BIN_NAME := qasereporter-k6
LDFLAGS := -w -s
TOFU := ./internal/vendored/bin/tofu
TOFU_MAIN_DIRS := k3d aws azure harvester

# =============================================================================
# Build targets
# =============================================================================

.PHONY: build
build: internal/vendored/bin qasereporter-k6/${REPORTER_BIN_NAME}
	CGO_ENABLED=0 go build -ldflags '$(LDFLAGS)' -o ${DARTBOARD_BIN_NAME} cmd/dartboard/*.go

internal/vendored/bin:
	sh download-vendored-bin.sh

qasereporter-k6/${REPORTER_BIN_NAME}: qasereporter-k6/*.go
	CGO_ENABLED=0 go build -ldflags '$(LDFLAGS)' -o $@ qasereporter-k6/*.go

.PHONY: clean
clean:
	rm -rfv .bin
	rm -rfv internal/vendored/bin
	rm -fv ${DARTBOARD_BIN_NAME}
	rm -fv qasereporter-k6/${REPORTER_BIN_NAME}

# =============================================================================
# Go module verification
# =============================================================================

.PHONY: go-mod-verify
go-mod-verify:
	go mod tidy
	go mod verify
	@if [ -n "$$(git status --porcelain go.mod go.sum 2>/dev/null)" ]; then \
		echo "Go mod isn't up to date. Please run 'go mod tidy'."; \
		echo "The following files differ after tidying:"; \
		git status --porcelain go.mod go.sum; \
		exit 1; \
	fi

# =============================================================================
# Linting
# =============================================================================

.PHONY: lint
lint: internal/vendored/bin
	golangci-lint run

# =============================================================================
# OpenTofu validation
# =============================================================================

.PHONY: tofu-fmt-check
tofu-fmt-check: internal/vendored/bin
	@for dir in $(TOFU_MAIN_DIRS); do \
		$(TOFU) -chdir=./tofu/main/$$dir fmt -check -diff -recursive || exit 1; \
	done
	$(TOFU) -chdir=./tofu/modules fmt -check -diff -recursive

.PHONY: tofu-fmt
tofu-fmt: internal/vendored/bin
	@for dir in $(TOFU_MAIN_DIRS); do \
		$(TOFU) -chdir=./tofu/main/$$dir fmt -recursive; \
	done
	$(TOFU) -chdir=./tofu/modules fmt -recursive

.PHONY: tofu-validate
tofu-validate: internal/vendored/bin
	@for dir in $(TOFU_MAIN_DIRS); do \
		echo "Validating $$dir..."; \
		$(TOFU) -chdir=./tofu/main/$$dir init -backend=false || exit 1; \
		$(TOFU) -chdir=./tofu/main/$$dir validate || exit 1; \
	done

# =============================================================================
# Combined verification targets
# =============================================================================

.PHONY: verify
verify: go-mod-verify build lint

.PHONY: verify-tofu
verify-tofu: tofu-fmt-check tofu-validate

.PHONY: verify-all
verify-all: verify verify-tofu
