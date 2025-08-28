#!/bin/bash

set -e

# similarity-go installation script
# This script detects the platform and downloads the appropriate binary

REPO="paveg/similarity-go"
BINARY_NAME="similarity-go"
INSTALL_DIR="/usr/local/bin"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Print colored output
print_info() {
    printf "${BLUE}[INFO]${NC} %s\n" "$1"
}

print_success() {
    printf "${GREEN}[SUCCESS]${NC} %s\n" "$1"
}

print_warning() {
    printf "${YELLOW}[WARNING]${NC} %s\n" "$1"
}

print_error() {
    printf "${RED}[ERROR]${NC} %s\n" "$1"
}

# Detect OS and architecture
detect_platform() {
    local os=""
    local arch=""
    
    case "$(uname -s)" in
        Linux*)     os="Linux" ;;
        Darwin*)    os="Darwin" ;;
        CYGWIN*)    os="Windows" ;;
        MINGW*)     os="Windows" ;;
        MSYS*)      os="Windows" ;;
        *)          
            print_error "Unsupported OS: $(uname -s)"
            exit 1
            ;;
    esac
    
    case "$(uname -m)" in
        x86_64)     arch="x86_64" ;;
        amd64)      arch="x86_64" ;;
        arm64)      arch="arm64" ;;
        aarch64)    arch="arm64" ;;
        *)          
            print_error "Unsupported architecture: $(uname -m)"
            exit 1
            ;;
    esac
    
    echo "${os}_${arch}"
}

# Get the latest release version
get_latest_version() {
    local api_url="https://api.github.com/repos/${REPO}/releases/latest"
    
    if command -v curl >/dev/null 2>&1; then
        curl -s "$api_url" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/'
    elif command -v wget >/dev/null 2>&1; then
        wget -qO- "$api_url" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/'
    else
        print_error "Neither curl nor wget is available. Please install one of them."
        exit 1
    fi
}

# Download and install binary
install_binary() {
    local version="$1"
    local platform="$2"
    local download_url="https://github.com/${REPO}/releases/download/${version}/similarity-go_${platform}.tar.gz"
    local temp_dir=$(mktemp -d)
    local archive_file="${temp_dir}/similarity-go.tar.gz"
    
    print_info "Downloading ${BINARY_NAME} ${version} for ${platform}..."
    
    # Download the archive
    if command -v curl >/dev/null 2>&1; then
        if ! curl -L -o "$archive_file" "$download_url"; then
            print_error "Failed to download ${download_url}"
            rm -rf "$temp_dir"
            exit 1
        fi
    elif command -v wget >/dev/null 2>&1; then
        if ! wget -O "$archive_file" "$download_url"; then
            print_error "Failed to download ${download_url}"
            rm -rf "$temp_dir"
            exit 1
        fi
    fi
    
    # Extract the archive
    print_info "Extracting archive..."
    if ! tar -xzf "$archive_file" -C "$temp_dir"; then
        print_error "Failed to extract archive"
        rm -rf "$temp_dir"
        exit 1
    fi
    
    # Find the binary
    local binary_path=$(find "$temp_dir" -name "$BINARY_NAME" -type f)
    if [[ -z "$binary_path" ]]; then
        print_error "Binary not found in archive"
        rm -rf "$temp_dir"
        exit 1
    fi
    
    # Install the binary
    print_info "Installing to ${INSTALL_DIR}..."
    
    # Check if we need sudo
    if [[ ! -w "$INSTALL_DIR" ]]; then
        if command -v sudo >/dev/null 2>&1; then
            sudo cp "$binary_path" "$INSTALL_DIR/$BINARY_NAME"
            sudo chmod +x "$INSTALL_DIR/$BINARY_NAME"
        else
            print_error "No write permission to ${INSTALL_DIR} and sudo is not available"
            print_info "Please run: cp \"$binary_path\" \"$INSTALL_DIR/$BINARY_NAME\" && chmod +x \"$INSTALL_DIR/$BINARY_NAME\""
            rm -rf "$temp_dir"
            exit 1
        fi
    else
        cp "$binary_path" "$INSTALL_DIR/$BINARY_NAME"
        chmod +x "$INSTALL_DIR/$BINARY_NAME"
    fi
    
    # Cleanup
    rm -rf "$temp_dir"
}

# Verify installation
verify_installation() {
    if command -v "$BINARY_NAME" >/dev/null 2>&1; then
        local version_output=$($BINARY_NAME --version 2>/dev/null || echo "version command not available")
        print_success "Installation successful!"
        print_info "Installed: $version_output"
        print_info "Location: $(which $BINARY_NAME)"
    else
        print_error "Installation verification failed. ${BINARY_NAME} not found in PATH."
        print_info "Make sure ${INSTALL_DIR} is in your PATH environment variable."
        exit 1
    fi
}

# Show usage examples
show_usage() {
    cat << 'EOF'

Usage Examples:
  # Analyze specific files
  similarity-go main.go utils.go

  # Scan entire directory
  similarity-go ./src

  # Multiple directories with custom threshold
  similarity-go --threshold 0.7 ./cmd ./internal

  # Output in YAML format
  similarity-go --format yaml --output results.yaml ./project

For more information, visit: https://github.com/paveg/similarity-go
EOF
}

main() {
    print_info "Installing similarity-go..."
    
    # Check dependencies
    if ! command -v tar >/dev/null 2>&1; then
        print_error "tar is required but not installed."
        exit 1
    fi
    
    # Detect platform
    local platform=$(detect_platform)
    print_info "Detected platform: $platform"
    
    # Get latest version
    print_info "Fetching latest release information..."
    local version=$(get_latest_version)
    if [[ -z "$version" ]]; then
        print_error "Failed to get latest version information"
        exit 1
    fi
    print_info "Latest version: $version"
    
    # Install binary
    install_binary "$version" "$platform"
    
    # Verify installation
    verify_installation
    
    # Show usage examples
    show_usage
}

# Handle command line arguments
case "${1:-}" in
    --help|-h)
        echo "similarity-go installation script"
        echo ""
        echo "Usage: $0 [OPTIONS]"
        echo ""
        echo "Options:"
        echo "  -h, --help     Show this help message"
        echo "  --uninstall    Remove similarity-go binary"
        echo ""
        echo "Environment variables:"
        echo "  INSTALL_DIR    Installation directory (default: /usr/local/bin)"
        echo ""
        echo "Example:"
        echo "  curl -sfL https://raw.githubusercontent.com/paveg/similarity-go/main/install.sh | sh"
        exit 0
        ;;
    --uninstall)
        print_info "Uninstalling similarity-go..."
        if [[ -f "$INSTALL_DIR/$BINARY_NAME" ]]; then
            if [[ -w "$INSTALL_DIR" ]]; then
                rm "$INSTALL_DIR/$BINARY_NAME"
            else
                sudo rm "$INSTALL_DIR/$BINARY_NAME"
            fi
            print_success "Uninstalled successfully"
        else
            print_warning "similarity-go not found in $INSTALL_DIR"
        fi
        exit 0
        ;;
    "")
        # No arguments, proceed with installation
        main
        ;;
    *)
        print_error "Unknown option: $1"
        print_info "Use --help for usage information"
        exit 1
        ;;
esac