#!/usr/bin/env bash

set -euo pipefail

VERSION=""
SKIP_TAG=false
DRY_RUN=false

print_usage() {
    echo "Usage: $0 [-v VERSION] [-s] [-d] [-h]"
    echo "  -v VERSION  Specify the version to release (e.g., v1.2.3)"
    echo "  -s          Skip creating a new git tag"
    echo "  -d          Dry run (don't actually release)"
    echo "  -h          Display this help message"
}

while getopts "v:sdh" opt; do
    case ${opt} in
        v )
            VERSION=$OPTARG
            ;;
        s )
            SKIP_TAG=true
            ;;
        d )
            DRY_RUN=true
            ;;
        h )
            print_usage
            exit 0
            ;;
        \? )
            print_usage
            exit 1
            ;;
    esac
done

if [ -z "$VERSION" ]; then
    echo "Error: Version is required. Use -v to specify the version."
    print_usage
    exit 1
fi

# Ensure we're on the main branch
if [ "$(git rev-parse --abbrev-ref HEAD)" != "main" ]; then
    echo "Error: Not on main branch. Please switch to main before releasing."
    exit 1
fi

# Ensure the working directory is clean
if [ -n "$(git status --porcelain)" ]; then
    echo "Error: Working directory is not clean. Please commit or stash changes."
    exit 1
fi

# Create and push tag if not skipped
if [ "$SKIP_TAG" = false ]; then
    export GITHUB_TOKEN=$(gh auth token)
    echo "Creating git tag $VERSION..."
    git tag -a "$VERSION" -m "Release $VERSION"
    git push origin "$VERSION"
    goreleaser release --clean
fi

# Run goreleaser
if [ "$DRY_RUN" = true ]; then
    echo "Dry run: would execute 'GITHUB_TOKEN=$(gh auth token) goreleaser release --clean'"
else
    echo "Running goreleaser..."
    GITHUB_TOKEN=$(gh auth token) goreleaser release --clean
fi

echo "Release process completed successfully!"
