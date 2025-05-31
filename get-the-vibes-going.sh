#!/bin/bash

# Script: get-the-vibes-going.sh
# Purpose: Set up proompts directory in a git project by copying from the proompting repository
# Usage: ./get-the-vibes-going.sh <target_directory>

set -e  # Exit on error

# Color codes for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Function to print colored messages
print_error() {
    echo -e "${RED}Error: $1${NC}" >&2
}

print_success() {
    echo -e "${GREEN}✓ $1${NC}"
}

print_info() {
    echo -e "${YELLOW}→ $1${NC}"
}

# Check if directory argument is provided
if [ $# -eq 0 ]; then
    print_error "No directory provided"
    echo "Usage: $0 <target_directory>"
    exit 1
fi

TARGET_DIR="$1"

# Get the directory where this script is located
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SOURCE_PROOMPTS_DIR="${SCRIPT_DIR}/proompts"

# Validate the target directory exists
if [ ! -d "$TARGET_DIR" ]; then
    print_error "Directory '$TARGET_DIR' does not exist"
    exit 1
fi

# Convert to absolute path
TARGET_DIR="$(cd "$TARGET_DIR" && pwd)"

# Check if the target directory is a git repository
if [ ! -d "$TARGET_DIR/.git" ]; then
    print_error "Directory '$TARGET_DIR' is not a git repository"
    exit 1
fi

# Check if source proompts directory exists
if [ ! -d "$SOURCE_PROOMPTS_DIR" ]; then
    print_error "Source proompts directory not found at '$SOURCE_PROOMPTS_DIR'"
    exit 1
fi

print_info "Setting up proompts in: $TARGET_DIR"

# Create the proompts directory in the target
TARGET_PROOMPTS_DIR="$TARGET_DIR/proompts"

if [ -d "$TARGET_PROOMPTS_DIR" ]; then
    print_info "Proompts directory already exists at '$TARGET_PROOMPTS_DIR'"
    read -p "Do you want to overwrite existing files? (y/N): " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        print_info "Operation cancelled"
        exit 0
    fi
else
    mkdir -p "$TARGET_PROOMPTS_DIR"
    print_success "Created proompts directory"
fi

# Copy all files from source proompts to target proompts
print_info "Copying proompts files..."
cp -r "$SOURCE_PROOMPTS_DIR"/* "$TARGET_PROOMPTS_DIR"/ 2>/dev/null || {
    if [ -z "$(ls -A "$SOURCE_PROOMPTS_DIR")" ]; then
        print_info "No files to copy from source proompts directory"
    else
        print_error "Failed to copy proompts files"
        exit 1
    fi
}

# Count copied files
FILE_COUNT=$(find "$TARGET_PROOMPTS_DIR" -type f | wc -l | tr -d ' ')
if [ "$FILE_COUNT" -gt 0 ]; then
    print_success "Copied $FILE_COUNT file(s) to proompts directory"
fi

# Handle .gitignore
GITIGNORE_PATH="$TARGET_DIR/.gitignore"

if [ ! -f "$GITIGNORE_PATH" ]; then
    # Create .gitignore if it doesn't exist
    touch "$GITIGNORE_PATH"
    print_success "Created .gitignore file"
fi

# Check if proompts is already in .gitignore
if grep -q "^proompts/?$\|^proompts$\|^/proompts/?$\|^/proompts$" "$GITIGNORE_PATH"; then
    print_info "proompts is already in .gitignore"
else
    # Add proompts to .gitignore
    echo "proompts/" >> "$GITIGNORE_PATH"
    print_success "Added proompts/ to .gitignore"
fi

print_success "Setup complete! The vibes are going in '$TARGET_DIR' 🚀"
print_info "The proompts directory has been added to .gitignore and won't be tracked by git"
