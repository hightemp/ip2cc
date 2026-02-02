#!/bin/bash
# ip2cc installer for Unix systems (Linux/macOS)
# Usage: curl -fsSL https://raw.githubusercontent.com/hightemp/ip2cc/main/scripts/install.sh | bash

set -euo pipefail

REPO="hightemp/ip2cc"
BINARY_NAME="ip2cc"
INSTALL_DIR_SYSTEM="/usr/local/bin"
INSTALL_DIR_USER="$HOME/.local/bin"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Detect OS
detect_os() {
    local os
    os=$(uname -s | tr '[:upper:]' '[:lower:]')
    case "$os" in
        linux*)  echo "linux" ;;
        darwin*) echo "darwin" ;;
        *)       log_error "Unsupported OS: $os"; exit 1 ;;
    esac
}

# Detect architecture
detect_arch() {
    local arch
    arch=$(uname -m)
    case "$arch" in
        x86_64|amd64)  echo "amd64" ;;
        aarch64|arm64) echo "arm64" ;;
        *)             log_error "Unsupported architecture: $arch"; exit 1 ;;
    esac
}

# Get latest version from GitHub releases
get_latest_version() {
    local latest
    latest=$(curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" | grep '"tag_name"' | sed -E 's/.*"([^"]+)".*/\1/')
    if [[ -z "$latest" ]]; then
        log_error "Failed to get latest version"
        exit 1
    fi
    echo "$latest"
}

# Download file
download_file() {
    local url=$1
    local output=$2
    
    if command -v curl &> /dev/null; then
        curl -fsSL "$url" -o "$output"
    elif command -v wget &> /dev/null; then
        wget -q "$url" -O "$output"
    else
        log_error "Neither curl nor wget found. Please install one of them."
        exit 1
    fi
}

# Verify SHA256 checksum
verify_checksum() {
    local file=$1
    local expected=$2
    local actual
    
    if command -v sha256sum &> /dev/null; then
        actual=$(sha256sum "$file" | awk '{print $1}')
    elif command -v shasum &> /dev/null; then
        actual=$(shasum -a 256 "$file" | awk '{print $1}')
    else
        log_warn "No SHA256 tool found, skipping verification"
        return 0
    fi
    
    if [[ "$actual" != "$expected" ]]; then
        log_error "Checksum verification failed!"
        log_error "Expected: $expected"
        log_error "Actual:   $actual"
        return 1
    fi
    
    log_info "Checksum verified"
    return 0
}

# Determine install directory
get_install_dir() {
    if [[ -w "$INSTALL_DIR_SYSTEM" ]]; then
        echo "$INSTALL_DIR_SYSTEM"
    else
        mkdir -p "$INSTALL_DIR_USER"
        echo "$INSTALL_DIR_USER"
    fi
}

# Main installation
main() {
    log_info "Installing ip2cc..."
    
    local os arch version install_dir
    os=$(detect_os)
    arch=$(detect_arch)
    version=$(get_latest_version)
    install_dir=$(get_install_dir)
    
    log_info "Detected: OS=$os, Arch=$arch"
    log_info "Version: $version"
    log_info "Install directory: $install_dir"
    
    local binary_name="${BINARY_NAME}_${os}_${arch}"
    if [[ "$os" == "windows" ]]; then
        binary_name="${binary_name}.exe"
    fi
    
    local download_url="https://github.com/${REPO}/releases/download/${version}/${binary_name}"
    local checksums_url="https://github.com/${REPO}/releases/download/${version}/checksums.txt"
    
    local tmp_dir
    tmp_dir=$(mktemp -d)
    trap "rm -rf $tmp_dir" EXIT
    
    local binary_path="${tmp_dir}/${binary_name}"
    local checksums_path="${tmp_dir}/checksums.txt"
    
    # Download binary
    log_info "Downloading ${binary_name}..."
    download_file "$download_url" "$binary_path"
    
    # Download and verify checksum
    log_info "Downloading checksums..."
    download_file "$checksums_url" "$checksums_path"
    
    local expected_checksum
    expected_checksum=$(grep "${binary_name}" "$checksums_path" | awk '{print $1}')
    if [[ -n "$expected_checksum" ]]; then
        verify_checksum "$binary_path" "$expected_checksum"
    else
        log_warn "Checksum not found for ${binary_name}, skipping verification"
    fi
    
    # Install binary
    log_info "Installing to ${install_dir}/${BINARY_NAME}..."
    chmod +x "$binary_path"
    mv "$binary_path" "${install_dir}/${BINARY_NAME}"
    
    # Verify installation
    if command -v "$BINARY_NAME" &> /dev/null; then
        log_info "Installation successful!"
        "$BINARY_NAME" version
    else
        log_warn "Installation complete, but '${BINARY_NAME}' is not in PATH"
        log_warn "Add ${install_dir} to your PATH:"
        log_warn "  export PATH=\"${install_dir}:\$PATH\""
        log_warn "Or add this line to your shell profile (~/.bashrc, ~/.zshrc, etc.)"
    fi
    
    echo ""
    log_info "Quick start:"
    log_info "  1. Download data: ${BINARY_NAME} update"
    log_info "  2. Lookup an IP:  ${BINARY_NAME} 8.8.8.8"
}

main "$@"
