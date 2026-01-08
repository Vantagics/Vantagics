.PHONY: build debug clean install-deps help

APP_NAME := RapidBI
SRC_DIR := src
BUILD_DIR := $(SRC_DIR)/build/bin

help:
	@echo "Usage: make [command]"
	@echo ""
	@echo "Commands:"
	@echo "  build         Build the application for the current platform"
	@echo "  debug         Build the application with debug symbols"
	@echo "  clean         Remove build artifacts"
	@echo "  install-deps  Install Go and NPM dependencies"
	@echo "  windows       Build for Windows"
	@echo "  darwin        Build for macOS"
	@echo "  linux         Build for Linux"

build:
	cd $(SRC_DIR) && wails build -clean

debug:
	cd $(SRC_DIR) && wails build -debug

clean:
	rm -rf $(BUILD_DIR)
	rm -rf $(SRC_DIR)/frontend/dist

install-deps:
	cd $(SRC_DIR) && go mod download
	cd $(SRC_DIR)/frontend && npm install

windows:
	cd $(SRC_DIR) && wails build -platform windows/amd64 -clean

darwin:
	cd $(SRC_DIR) && wails build -platform darwin/universal -clean

linux:
	cd $(SRC_DIR) && wails build -platform linux/amd64 -clean
