GO           := go
LD_FLAGS     := -s -w
BUILD_DIR    := ./build
BUILD_TIME   := $(shell date '+%Y-%m-%d %H:%M:%S')
BUILD_FLAGS  := -trimpath -ldflags "$(LD_FLAGS) -X 'main.BUILD_TIME=$(BUILD_TIME)'"

.PHONY: all build build-linux-amd64 build-linux-arm64 build-darwin-arm64 build-windows-amd64

all: build

build: build-linux-amd64 build-linux-arm64 build-darwin-arm64 build-windows-amd64

ensure-path:
	@mkdir -p $(BUILD_DIR)/linux-amd64/
	@mkdir -p $(BUILD_DIR)/linux-arm64/
	@mkdir -p $(BUILD_DIR)/darwin-arm64/
	@mkdir -p $(BUILD_DIR)/windows-amd64/

build-linux-amd64: ensure-path
	GOOS=linux GOARCH=amd64 $(GO) build $(BUILD_FLAGS) -o $(BUILD_DIR)/linux-amd64/
	cp config.yaml $(BUILD_DIR)/linux-amd64/

build-linux-arm64: ensure-path
	GOOS=linux GOARCH=arm64 $(GO) build $(BUILD_FLAGS) -o $(BUILD_DIR)/linux-arm64/
	cp config.yaml $(BUILD_DIR)/linux-arm64/

build-darwin-arm64: ensure-path
	GOOS=darwin GOARCH=arm64 $(GO) build $(BUILD_FLAGS) -o $(BUILD_DIR)/darwin-arm64/
	cp config.yaml $(BUILD_DIR)/darwin-arm64/

build-windows-amd64: ensure-path
	GOOS=windows GOARCH=amd64 $(GO) build $(BUILD_FLAGS) -o $(BUILD_DIR)/windows-amd64/
	cp config.yaml $(BUILD_DIR)/windows-amd64/
