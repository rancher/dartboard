BIN_NAME := dartboard
LDFLAGS := -w -s

.PHONY: build
build: internal/vendored/bin
	CGO_ENABLED=0 go build -ldflags '$(LDFLAGS)' -o ${BIN_NAME} cmd/dartboard/*.go

internal/vendored/bin:
	sh download-vendored-bin.sh

.PHONY: clean
clean:
	rm -fv ${BIN_NAME}
