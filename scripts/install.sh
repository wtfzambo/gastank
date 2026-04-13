#!/usr/bin/env bash

set -euo pipefail

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

REPO="donnaknows/gastank"

log_info() {
  printf "%b==>%b %s\n" "$BLUE" "$NC" "$1"
}

log_success() {
  printf "%b==>%b %s\n" "$GREEN" "$NC" "$1"
}

log_warning() {
  printf "%b==>%b %s\n" "$YELLOW" "$NC" "$1"
}

log_error() {
  printf "%bError:%b %s\n" "$RED" "$NC" "$1" >&2
}

require_cmd() {
  if ! command -v "$1" >/dev/null 2>&1; then
    log_error "Required command not found: $1"
    exit 1
  fi
}

download_file() {
  local url=$1
  local out=$2

  if command -v curl >/dev/null 2>&1; then
    curl -fsSL -o "$out" "$url"
    return
  fi

  wget -qO "$out" "$url"
}

latest_tag() {
  local url="https://api.github.com/repos/${REPO}/releases/latest"

  if command -v curl >/dev/null 2>&1; then
    curl -fsSL "$url" | sed -n 's/.*"tag_name": *"\([^"]*\)".*/\1/p' | head -n 1
    return
  fi

  wget -qO- "$url" | sed -n 's/.*"tag_name": *"\([^"]*\)".*/\1/p' | head -n 1
}

detect_platform() {
  case "$(uname -s)" in
    Darwin)
      printf 'macos\n'
      ;;
    Linux)
      printf 'linux\n'
      ;;
    *)
      log_error "Unsupported operating system: $(uname -s)"
      exit 1
      ;;
  esac
}

install_macos() {
  local version=$1
  local tmp_dir archive url

  require_cmd unzip

  tmp_dir=$(mktemp -d)
  trap 'rm -rf "$tmp_dir"' EXIT
  archive="gastank-macos-universal.zip"
  url="https://github.com/${REPO}/releases/download/${version}/${archive}"

  log_info "Downloading ${archive}"
  download_file "$url" "$tmp_dir/$archive"

  unzip -q "$tmp_dir/$archive" -d "$tmp_dir"

  if [ ! -d "$tmp_dir/gastank.app" ]; then
    log_error "Expected gastank.app in the downloaded archive"
    exit 1
  fi

  rm -rf /Applications/gastank.app
  mv "$tmp_dir/gastank.app" /Applications/gastank.app

  log_success "Installed /Applications/gastank.app"
  log_warning "Unsigned builds may need: xattr -cr /Applications/gastank.app"
}

install_linux() {
  local version=$1
  local tmp_dir appimage_url archive_name target

  tmp_dir=$(mktemp -d)
  trap 'rm -rf "$tmp_dir"' EXIT

  log_info "Resolving AppImage from GitHub API for ${version}"

  local api_url="https://api.github.com/repos/${REPO}/releases/tags/${version}"
  local release_json

  if command -v curl >/dev/null 2>&1; then
    release_json=$(curl -fsSL "$api_url")
  else
    release_json=$(wget -qO- "$api_url")
  fi

  # Extract the AppImage browser_download_url from the JSON response.
  # Uses sed to avoid a hard dependency on jq.
  appimage_url=$(printf '%s' "$release_json" \
    | sed -n 's/.*"browser_download_url": *"\([^"]*\.AppImage\)".*/\1/p' \
    | head -n 1)

  if [ -z "$appimage_url" ]; then
    log_error "Could not find an AppImage asset in the GitHub release"
    exit 1
  fi

  archive_name=$(basename "$appimage_url")
  log_info "Downloading ${archive_name}"
  download_file "$appimage_url" "$tmp_dir/$archive_name"

  target="$HOME/.local/bin/gastank-app"
  mkdir -p "$(dirname "$target")"
  install -m 0755 "$tmp_dir/$archive_name" "$target"

  log_success "Installed AppImage launcher at ${target}"

  if [[ ":$PATH:" != *":$HOME/.local/bin:"* ]]; then
    log_warning "$HOME/.local/bin is not in your PATH"
    printf 'Add this to your shell profile:\n  export PATH="$PATH:$HOME/.local/bin"\n'
  fi
}

main() {
  local version platform
  version=$(latest_tag)
  if [ -z "$version" ]; then
    log_error "Could not determine latest release"
    exit 1
  fi

  platform=$(detect_platform)
  log_info "Latest release: ${version}"

  case "$platform" in
    macos)
      install_macos "$version"
      ;;
    linux)
      install_linux "$version"
      ;;
  esac
}

main "$@"
