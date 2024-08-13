BIN_NAME := dartboard
LDFLAGS := -w -s

.PHONY: build
build:
	CGO_ENABLED=0 go build -ldflags '$(LDFLAGS)' -o ${BIN_NAME} cmd/dartboard/*.go

.PHONY: clean
clean:
	rm -fv ${BIN_NAME}
