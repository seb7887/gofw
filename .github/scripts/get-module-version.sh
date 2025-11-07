#!/bin/bash
# Gets the latest version tag for a module and increments patch version
# Usage: ./get-module-version.sh <module-name>
# Output: <module>/v<major>.<minor>.<patch+1>

set -euo pipefail

MODULE="$1"

if [ -z "$MODULE" ]; then
  echo "Error: Module name required" >&2
  exit 1
fi

# Get all tags for this module (format: <module>/v*.*.*)
LATEST_TAG=$(git tag --list "${MODULE}/v*.*.*" --sort=-v:refname | head -n 1)

if [ -z "$LATEST_TAG" ]; then
  # No previous tag - this is the first release
  NEW_VERSION="${MODULE}/v0.1.0"
  echo "No previous version found. Using initial version: $NEW_VERSION" >&2
else
  echo "Latest tag found: $LATEST_TAG" >&2

  # Extract version numbers (remove module prefix and 'v')
  VERSION="${LATEST_TAG#${MODULE}/v}"

  # Split into major.minor.patch
  IFS='.' read -r MAJOR MINOR PATCH <<< "$VERSION"

  # Increment patch version
  PATCH=$((PATCH + 1))

  NEW_VERSION="${MODULE}/v${MAJOR}.${MINOR}.${PATCH}"
  echo "New version: $NEW_VERSION" >&2
fi

# Output the new version tag
echo "$NEW_VERSION"
