#!/bin/bash

# RapidBI Build Script
# This script helps to build the RapidBI application using Wails.

set -e

# Project configuration
APP_NAME="RapidBI"
SRC_DIR="src"
BUILD_DIR="$SRC_DIR/build/bin"

# Help message
show_help() {
    echo "Usage: ./build.sh [command] [options]"
    echo ""
    echo "Commands:"
    echo "  build (default)   Build the application for the current platform"
    echo "  debug             Build the application with debug symbols and developer tools"
    echo "  clean             Remove build artifacts"
    echo "  install-deps      Install Go and NPM dependencies"
    echo "  help              Show this help message"
    echo ""
    echo "Platforms (for build command):"
    echo "  --darwin          Build for macOS"
    echo "  --windows         Build for Windows"
    echo "  --linux           Build for Linux"
    echo "  --universal       Build for macOS Universal (Intel + Apple Silicon)"
    echo ""
    echo "Example:"
    echo "  ./build.sh debug"
    echo "  ./build.sh build --windows"
}

# Check for dependencies
check_deps() {
    if ! command -v go &> /dev/null; then
        echo "Error: Go is not installed. Please install Go (https://golang.org/)."
        exit 1
    fi

    if ! command -v npm &> /dev/null; then
        echo "Error: NPM is not installed. Please install Node.js (https://nodejs.org/)."
        exit 1
    fi

    if ! command -v wails &> /dev/null; then
        echo "Wails CLI not found. Installing latest Wails v2..."
        go install github.com/wailsapp/wails/v2/cmd/wails@latest
        
        # Add Go bin to PATH for this session
        GOBIN=$(go env GOBIN)
        if [ -z "$GOBIN" ]; then
            GOBIN="$(go env GOPATH)/bin"
        fi
        export PATH=$PATH:$GOBIN
    fi
}

# Main logic
COMMAND=${1:-"build"}
PLATFORM=""

case $COMMAND in
    "help"|"-h"|"--help")
        show_help
        exit 0
        ;;
    "clean")
        echo "Cleaning build artifacts..."
        rm -rf "$BUILD_DIR"
        rm -rf "$SRC_DIR/frontend/dist"
        echo "Done."
        exit 0
        ;;
    "install-deps")
        check_deps
        echo "Installing Go dependencies..."
        cd "$SRC_DIR" && go mod download
        echo "Installing NPM dependencies..."
        cd "frontend" && npm install
        echo "Dependencies installed."
        exit 0
        ;;
    "build"|"debug")
        check_deps
        
        # Parse platform flags
        shift || true
        while [[ "$#" -gt 0 ]]; do
            case $1 in
                --darwin) 
                    if [ "$PLATFORM" != "darwin/universal" ]; then
                        PLATFORM="darwin"
                    fi
                    ;;
                --windows) PLATFORM="windows" ;;
                --linux) PLATFORM="linux" ;;
                --universal) PLATFORM="darwin/universal" ;;
                *) echo "Unknown option: $1"; show_help; exit 1 ;;
            esac
            shift
        done

        BUILD_CMD="wails build"
        if [ "$COMMAND" == "debug" ]; then
            BUILD_CMD="$BUILD_CMD -debug"
        else
            BUILD_CMD="$BUILD_CMD -clean"
        fi

        if [ -n "$PLATFORM" ]; then
            BUILD_CMD="$BUILD_CMD -platform $PLATFORM"
        fi

        echo "Starting $COMMAND for ${PLATFORM:-current platform}..."
        cd "$SRC_DIR"
        $BUILD_CMD
        
        echo ""
        echo "$APP_NAME build finished successfully!"
        echo "Output directory: $BUILD_DIR"
        ;;
    *)
        echo "Unknown command: $COMMAND"
        show_help
        exit 1
        ;;
esac
