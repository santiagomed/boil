#!/bin/bash

set -e

INSTALL_DIR="/usr/local/bin"
BINARY_NAME="boil"

echo "Uninstalling boil..."

# Check if boil is installed
if [ ! -f "$INSTALL_DIR/$BINARY_NAME" ]; then
    echo "boil is not installed in $INSTALL_DIR. Nothing to uninstall."
    exit 0
fi

# Remove the binary
sudo rm "$INSTALL_DIR/$BINARY_NAME"
echo "Removed $INSTALL_DIR/$BINARY_NAME"

# Check for any leftover files (this step is optional and depends on your app structure)
LEFTOVER_FILES=$(find "$INSTALL_DIR" -name "boil*")
if [ ! -z "$LEFTOVER_FILES" ]; then
    echo "Found additional files related to boil:"
    echo "$LEFTOVER_FILES"
    read -p "Do you want to remove these files? (y/N) " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        echo "$LEFTOVER_FILES" | xargs sudo rm -rf
        echo "Removed additional files."
    fi
fi

echo "boil has been uninstalled."

# Remind user about PATH
echo "Note: If you manually added boil to your PATH, you may want to remove that entry from your shell configuration file (.bashrc, .bash_profile, .zshrc, etc.)."

# Optional: Remind about configuration files
echo "If boil created any configuration files in your home directory, you may want to remove them manually."