BUILD_DIR := bin
BIN_NAME := scli
LDFLAGS := -w -s

.PHONY: build
build:
	CGO_ENABLED=0 go build -ldflags '$(LDFLAGS)' -o $(BUILD_DIR)/${BIN_NAME} src/cmd/*.go

.PHONY: clean
clean:
	rm -fv ${BUILD_DIR}/${BIN_NAME}

