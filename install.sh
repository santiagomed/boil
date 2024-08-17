#!/bin/bash

set -e

# Function to detect OS and architecture
detect_os_arch() {
    OS=$(uname -s | tr '[:upper:]' '[:lower:]')
    ARCH=$(uname -m)
    case $ARCH in
        x86_64) ARCH="x86_64" ;;
        aarch64 | arm64) ARCH="arm64" ;;
        i386 | i686) ARCH="i386" ;;
        *)
            echo "Unsupported architecture: $ARCH"
            exit 1
            ;;
    esac
}

# Function to download and install
install_boil() {
    DOWNLOAD_URL="https://github.com/santiagomed/boil/releases/latest/download/boil_${OS}_${ARCH}.tar.gz"
    INSTALL_DIR="/usr/local/bin"
    TMP_DIR=$(mktemp -d)

    echo "Downloading boil for $OS $ARCH..."
    curl -L "$DOWNLOAD_URL" -o "$TMP_DIR/boil.tar.gz"

    echo "Extracting..."
    tar -xzf "$TMP_DIR/boil.tar.gz" -C "$TMP_DIR"

    echo "Installing to $INSTALL_DIR..."
    sudo mv "$TMP_DIR/boil" "$INSTALL_DIR/"
    sudo chmod +x "$INSTALL_DIR/boil"

    echo "Cleaning up..."
    rm -rf "$TMP_DIR"

    echo "boil has been installed to $INSTALL_DIR/boil"

    # Check if the installation directory is in PATH
    if [[ ":$PATH:" != *":$INSTALL_DIR:"* ]]; then
        echo "NOTE: $INSTALL_DIR is not in your PATH."
        echo "To add it, run these commands:"
        echo "  echo 'export PATH=\$PATH:$INSTALL_DIR' >> ~/.bash_profile"
        echo "  source ~/.bash_profile"
        echo "Or add the following line to your shell's config file (.bashrc, .zshrc, etc.):"
        echo "  export PATH=\$PATH:$INSTALL_DIR"
    fi
}

# Main script
if [[ "$OSTYPE" == "darwin"* ]] || [[ "$OSTYPE" == "linux"* ]]; then
    detect_os_arch
    install_boil
else
    echo "This script is for macOS and Linux only. For Windows, please use the PowerShell script."
    exit 1
fi

echo "Installation complete. You can now use boil by running 'boil' in your terminal."
echo "If 'boil' is not recognized, please restart your terminal or source your shell's config file."