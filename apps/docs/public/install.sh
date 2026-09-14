#!/usr/bin/env bash
set -euo pipefail

# ANSI terminal colors
CYAN='\033[0;36m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
RED='\033[0;31m'
BOLD='\033[1m'
NC='\033[0m'

info()    { printf "${CYAN}info:${NC} %s\n" "$*"; }
success() { printf "${GREEN}ok:${NC} %s\n" "$*"; }
warn()    { printf "${YELLOW}warn:${NC} %s\n" "$*"; }
error()   { printf "${RED}error:${NC} %s\n" "$*" >&2; exit 1; }

# 1. Dependency checks
command -v curl >/dev/null 2>&1 || error "'curl' is required but not installed."
command -v tar >/dev/null 2>&1  || error "'tar' is required but not installed."

# 2. Operating system detection
OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
case "$OS" in
  linux*)  TARGET_OS="linux" ;;
  darwin*) TARGET_OS="darwin" ;;
  *)       error "Unsupported operating system: $OS" ;;
esac

# 3. CPU architecture detection
ARCH="$(uname -m)"
case "$ARCH" in
  x86_64|amd64)  TARGET_ARCH="amd64" ;;
  arm64|aarch64) TARGET_ARCH="arm64" ;;
  *)             error "Unsupported architecture: $ARCH" ;;
esac

OWNER="budment"
REPO="budment"
BIN_NAME="budment"

printf "\n${BOLD}Installing ${BIN_NAME} CLI...${NC}\n\n"

# 4. Fetch latest release version tag
LATEST_TAG=$(curl -fsSL "https://api.github.com/repos/${OWNER}/${REPO}/releases/latest" 2>/dev/null \
  | grep '"tag_name":' \
  | sed -E 's/.*"([^"]+)".*/\1/' || true)

[ -n "$LATEST_TAG" ] || error "Failed to fetch latest release metadata from GitHub."

VERSION="${LATEST_TAG#v}"
ARCHIVE_NAME="budment_${VERSION}_${TARGET_OS}_${TARGET_ARCH}.tar.gz"
DOWNLOAD_URL="https://github.com/${OWNER}/${REPO}/releases/download/${LATEST_TAG}/${ARCHIVE_NAME}"
CHECKSUMS_URL="https://github.com/${OWNER}/${REPO}/releases/download/${LATEST_TAG}/checksums.txt"

# 5. Prepare sandbox temporary directory with auto-cleanup
TEMP_DIR=$(mktemp -d)
trap 'rm -rf "$TEMP_DIR"' EXIT

# 6. Download binary archive and checksums
info "Downloading ${LATEST_TAG} (${ARCHIVE_NAME})..."
curl -fsSL "$DOWNLOAD_URL" -o "${TEMP_DIR}/${ARCHIVE_NAME}"

if curl -fsSL -I "$CHECKSUMS_URL" >/dev/null 2>&1; then
  info "Verifying SHA-256 checksum..."
  curl -fsSL "$CHECKSUMS_URL" -o "${TEMP_DIR}/checksums.txt"
  (
    cd "$TEMP_DIR"
    if command -v sha256sum >/dev/null 2>&1; then
      grep "${ARCHIVE_NAME}" checksums.txt | sha256sum -c - >/dev/null 2>&1 || error "Checksum verification failed!"
    elif command -v shasum >/dev/null 2>&1; then
      grep "${ARCHIVE_NAME}" checksums.txt | shasum -a 256 -c - >/dev/null 2>&1 || error "Checksum verification failed!"
    fi
  )
fi

# 7. Unpack archive
tar -xzf "${TEMP_DIR}/${ARCHIVE_NAME}" -C "$TEMP_DIR"
[ -f "${TEMP_DIR}/${BIN_NAME}" ] || error "Extracted archive does not contain '${BIN_NAME}' binary."

# 8. Install to primary directory
INSTALL_DIR="${BUDMENT_INSTALL_DIR:-$HOME/.budment/bin}"
mkdir -p "$INSTALL_DIR"
mv "${TEMP_DIR}/${BIN_NAME}" "${INSTALL_DIR}/${BIN_NAME}"
chmod +x "${INSTALL_DIR}/${BIN_NAME}"

# 9. Smart Symlink: If ~/.local/bin is already in PATH, link it there for instant execution
SYSTEM_LINKED=0
if [[ ":$PATH:" == *":$HOME/.local/bin:"* ]] && [ -d "$HOME/.local/bin" ] && [ -w "$HOME/.local/bin" ]; then
  ln -sf "${INSTALL_DIR}/${BIN_NAME}" "$HOME/.local/bin/${BIN_NAME}"
  SYSTEM_LINKED=1
fi

# 10. Update Shell configuration profiles (Bash, Zsh, Fish)
CONFIG_UPDATED=0
PATH_EXPORT='export PATH="$HOME/.budment/bin:$PATH"'

# Bash
if [ -f "$HOME/.bashrc" ] && ! grep -qs "budment/bin" "$HOME/.bashrc"; then
  printf "\n# Budment CLI\n%s\n" "$PATH_EXPORT" >> "$HOME/.bashrc"
  CONFIG_UPDATED=1
fi
if [ -f "$HOME/.bash_profile" ] && ! grep -qs "budment/bin" "$HOME/.bash_profile"; then
  printf "\n# Budment CLI\n%s\n" "$PATH_EXPORT" >> "$HOME/.bash_profile"
  CONFIG_UPDATED=1
fi

# Zsh
if [ -f "$HOME/.zshrc" ] && ! grep -qs "budment/bin" "$HOME/.zshrc"; then
  printf "\n# Budment CLI\n%s\n" "$PATH_EXPORT" >> "$HOME/.zshrc"
  CONFIG_UPDATED=1
fi

# Fish Shell
FISH_CONF_DIR="$HOME/.config/fish/conf.d"
if [ -d "$HOME/.config/fish" ]; then
  mkdir -p "$FISH_CONF_DIR"
  if [ ! -f "$FISH_CONF_DIR/budment.fish" ]; then
    echo 'fish_add_path "$HOME/.budment/bin"' > "$FISH_CONF_DIR/budment.fish"
    CONFIG_UPDATED=1
  fi
fi

# 11. Completion Banner
success "Budment CLI ${LATEST_TAG} installed to ${INSTALL_DIR}/${BIN_NAME}"

printf "\n"
if [ $SYSTEM_LINKED -eq 1 ]; then
  printf "${BOLD}Ready to use!${NC} Run: ${CYAN}budment --help${NC}\n\n"
else
  printf "${BOLD}Next step:${NC} Reload your current shell environment to activate the command:\n"
  printf "  ${CYAN}exec \"\$SHELL\"${NC}\n"
  printf "  or:\n"
  printf "  ${CYAN}export PATH=\"\$HOME/.budment/bin:\$PATH\"${NC}\n\n"
fi