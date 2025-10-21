BIN_NAME := dartboard
LDFLAGS := -w -s

.PHONY: build
build: internal/vendored/bin qasereporter-k6
	CGO_ENABLED=0 go build -ldflags '$(LDFLAGS)' -o ${BIN_NAME} cmd/dartboard/*.go

internal/vendored/bin:
	sh download-vendored-bin.sh

.PHONY: qasereporter-k6
qasereporter-k6:
	@echo "Building k6 Qase reporter..."
	@cd qasereporter-k6 && go mod tidy && go build -ldflags '$(LDFLAGS)' -o qasereporter-k6 .
	@echo "Build complete: qasereporter-k6/qasereporter-k6"

.PHONY: clean
clean:
	rm -fv ${BIN_NAME}
	rm -fv qasereporter-k6/qasereporter-k6
