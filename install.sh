#!/bin/sh
# MOCHI installer script
# Usage: curl -fsSL https://raw.githubusercontent.com/thisguymartin/Mochi/main/install.sh | sh

set -e

REPO="thisguymartin/Mochi"
BINARY_NAME="mochi"
INSTALL_DIR="/usr/local/bin"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

info() { printf "${CYAN}[info]${NC} %s\n" "$1"; }
success() { printf "${GREEN}[ok]${NC} %s\n" "$1"; }
warn() { printf "${YELLOW}[warn]${NC} %s\n" "$1"; }
error() { printf "${RED}[error]${NC} %s\n" "$1" >&2; exit 1; }

# Detect OS
detect_os() {
    case "$(uname -s)" in
        Linux*)  echo "linux" ;;
        Darwin*) echo "darwin" ;;
        *)       error "Unsupported OS: $(uname -s). Only Linux and macOS are supported." ;;
    esac
}

# Detect architecture
detect_arch() {
    case "$(uname -m)" in
        x86_64|amd64)  echo "amd64" ;;
        arm64|aarch64) echo "arm64" ;;
        *)             error "Unsupported architecture: $(uname -m). Only amd64 and arm64 are supported." ;;
    esac
}

# Check for required tools
check_dependencies() {
    if ! command -v git >/dev/null 2>&1; then
        error "git is required but not installed. Please install git first."
    fi
}

# Get latest release tag from GitHub
get_latest_version() {
    if command -v curl >/dev/null 2>&1; then
        curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" 2>/dev/null | grep '"tag_name"' | sed -E 's/.*"tag_name": *"([^"]+)".*/\1/'
    elif command -v wget >/dev/null 2>&1; then
        wget -qO- "https://api.github.com/repos/${REPO}/releases/latest" 2>/dev/null | grep '"tag_name"' | sed -E 's/.*"tag_name": *"([^"]+)".*/\1/'
    else
        return 1
    fi
}

# Download binary from GitHub releases
download_binary() {
    local os="$1"
    local arch="$2"
    local version="$3"
    local url="https://github.com/${REPO}/releases/download/${version}/${BINARY_NAME}_${os}_${arch}.tar.gz"
    local tmp_dir

    tmp_dir=$(mktemp -d)
    trap 'rm -rf "$tmp_dir"' EXIT

    info "Downloading ${BINARY_NAME} ${version} for ${os}/${arch}..."

    if command -v curl >/dev/null 2>&1; then
        curl -fsSL "$url" -o "${tmp_dir}/${BINARY_NAME}.tar.gz"
    elif command -v wget >/dev/null 2>&1; then
        wget -q "$url" -O "${tmp_dir}/${BINARY_NAME}.tar.gz"
    else
        error "Neither curl nor wget found. Please install one of them."
    fi

    tar -xzf "${tmp_dir}/${BINARY_NAME}.tar.gz" -C "$tmp_dir"

    if [ ! -f "${tmp_dir}/${BINARY_NAME}" ]; then
        error "Binary not found in archive. The release format may have changed."
    fi

    install_binary "${tmp_dir}/${BINARY_NAME}"
}

# Build from source
build_from_source() {
    if ! command -v go >/dev/null 2>&1; then
        error "Go is required to build from source but not found. Install Go 1.22+ from https://go.dev/dl/"
    fi

    local go_version
    go_version=$(go version | grep -oE 'go[0-9]+\.[0-9]+' | sed 's/go//')
    local major minor
    major=$(echo "$go_version" | cut -d. -f1)
    minor=$(echo "$go_version" | cut -d. -f2)

    if [ "$major" -lt 1 ] || { [ "$major" -eq 1 ] && [ "$minor" -lt 22 ]; }; then
        error "Go 1.22+ is required, but found Go ${go_version}."
    fi

    info "Building ${BINARY_NAME} from source..."

    local tmp_dir
    tmp_dir=$(mktemp -d)
    trap 'rm -rf "$tmp_dir"' EXIT

    git clone --depth 1 "https://github.com/${REPO}.git" "${tmp_dir}/mochi"
    cd "${tmp_dir}/mochi"
    go build -o "${tmp_dir}/${BINARY_NAME}" .

    install_binary "${tmp_dir}/${BINARY_NAME}"
}

# Install binary to system path
install_binary() {
    local src="$1"

    chmod +x "$src"

    if [ -w "$INSTALL_DIR" ]; then
        mv "$src" "${INSTALL_DIR}/${BINARY_NAME}"
    else
        info "Elevated permissions required to install to ${INSTALL_DIR}"
        sudo mv "$src" "${INSTALL_DIR}/${BINARY_NAME}"
    fi

    success "Installed ${BINARY_NAME} to ${INSTALL_DIR}/${BINARY_NAME}"
}

# Main
main() {
    echo ""
    echo "  MOCHI — Multi-Task AI Coding Orchestrator"
    echo "  https://github.com/${REPO}"
    echo ""

    check_dependencies

    local os arch version
    os=$(detect_os)
    arch=$(detect_arch)

    info "Detected platform: ${os}/${arch}"

    # Try to download a pre-built binary first
    version=$(get_latest_version)

    if [ -n "$version" ]; then
        info "Latest release: ${version}"
        download_binary "$os" "$arch" "$version" 2>/dev/null && {
            echo ""
            success "Installation complete! Run 'mochi --help' to get started."
            return 0
        }
        warn "Pre-built binary not available for ${os}/${arch}. Falling back to build from source..."
    else
        warn "No releases found. Building from source..."
    fi

    build_from_source

    echo ""
    success "Installation complete! Run 'mochi --help' to get started."
}

main "$@"
