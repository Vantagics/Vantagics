#!/bin/bash

# RapidBI Build Script
# This script helps to build the RapidBI application using Wails.

set -e

# Project configuration
APP_NAME="RapidBI"
SRC_DIR="src"
BUILD_DIR="dist"

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
        rm -rf "$SRC_DIR/build/bin"
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

        # Default to darwin/universal if no platform specified
        if [ -z "$PLATFORM" ]; then
            PLATFORM="darwin/universal"
        fi

        BUILD_CMD="wails build"
        if [ "$COMMAND" == "debug" ]; then
            BUILD_CMD="$BUILD_CMD -debug"
        else
            BUILD_CMD="$BUILD_CMD -clean"
        fi

        if [ -n "$PLATFORM" ]; then
            BUILD_CMD="$BUILD_CMD -platform $PLATFORM"
        fi

        # Set macOS deployment target
        if [[ "$PLATFORM" == *"darwin"* ]]; then
            export MACOSX_DEPLOYMENT_TARGET=15.10
        fi

        echo "Starting $COMMAND for ${PLATFORM:-current platform}..."
        cd "$SRC_DIR"
        $BUILD_CMD

        # Ensure output directory exists and move files
        mkdir -p "../$BUILD_DIR"
        # Using cp -R followed by rm to handle potential cross-device moves if dist is a mount
        cp -R build/bin/* "../$BUILD_DIR/"

        # Generate Universal PKG
        if [ "$PLATFORM" == "darwin/universal" ]; then
            echo "Generating Universal PKG..."
            # Extract output filename from wails.json
            OUTPUT_NAME=$(grep '"outputfilename"' wails.json | awk -F'"' '{print $4}')
            APP_BUNDLE="../$BUILD_DIR/${OUTPUT_NAME}.app"
            PKG_OUTPUT="../$BUILD_DIR/${APP_NAME}-Universal.pkg"
            IDENTIFIER=$(grep -A 1 'CFBundleIdentifier' build/Info.plist | grep 'string' | sed 's/.*<string>\(.*\)<\/string>.*/\1/')
            
            # Prepare scripts directory for PKG
            SCRIPTS_DIR="build/pkg_scripts"
            mkdir -p "$SCRIPTS_DIR"
            
            # Create postinstall script to add app to Dock
            cat > "$SCRIPTS_DIR/postinstall" <<EOF
#!/bin/bash

APP_PATH="/Applications/${OUTPUT_NAME}.app"
LOGGED_IN_USER=\$(scutil <<< "show State:/Users/ConsoleUser" | awk '/Name :/ && ! /loginwindow/ { print \$3 }')

if [ -n "\$LOGGED_IN_USER" ]; then
    USER_HOME=\$(dscl . -read /Users/"\$LOGGED_IN_USER" NFSHomeDirectory | cut -d ' ' -f 2)
    
    # Check if App is already in Dock to avoid duplicates
    if ! sudo -u "\$LOGGED_IN_USER" defaults read com.apple.dock persistent-apps | grep -q "\$APP_PATH"; then
        # Add to Dock
        sudo -u "\$LOGGED_IN_USER" defaults write com.apple.dock persistent-apps -array-add "<dict><key>tile-data</key><dict><key>file-data</key><dict><key>_CFURLString</key><string>\$APP_PATH</string><key>_CFURLStringType</key><integer>0</integer></dict></dict></dict>"
        
        # Restart Dock
        sudo -u "\$LOGGED_IN_USER" killall Dock
    fi
fi
exit 0
EOF
            chmod +x "$SCRIPTS_DIR/postinstall"

            if [ -d "$APP_BUNDLE" ]; then
                pkgbuild --component "$APP_BUNDLE" \
                         --install-location "/Applications" \
                         --identifier "${IDENTIFIER:-com.rapidbi.app}" \
                         --scripts "$SCRIPTS_DIR" \
                         "$PKG_OUTPUT"
                echo "--------------------------------------------------"
                echo "Universal PKG created: $BUILD_DIR/$(basename "$PKG_OUTPUT")"
                echo "--------------------------------------------------"
            else
                echo "Error: Could not find app bundle at $APP_BUNDLE to create PKG."
                exit 1
            fi
            
            # Clean up scripts
            rm -rf "$SCRIPTS_DIR"
        fi
        
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
