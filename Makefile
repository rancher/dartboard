DARTBOARD_BIN_NAME := dartboard
REPORTER_BIN_NAME := qasereporter-k6
LDFLAGS := -w -s

.PHONY: build
build: internal/vendored/bin qasereporter-k6/${REPORTER_BIN_NAME}
	CGO_ENABLED=0 go build -ldflags '$(LDFLAGS)' -o ${DARTBOARD_BIN_NAME} cmd/dartboard/*.go

internal/vendored/bin:
	sh download-vendored-bin.sh

qasereporter-k6/${REPORTER_BIN_NAME}: qasereporter-k6/*.go
	CGO_ENABLED=0 go build -ldflags '$(LDFLAGS)' -o $@ qasereporter-k6/*.go

.PHONY: clean
clean:
	rm -fv ${DARTBOARD_BIN_NAME}
	rm -fv qasereporter-k6/${REPORTER_BIN_NAME}
