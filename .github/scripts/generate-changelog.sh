#!/bin/bash
# Generates a changelog for a module from commits since last tag
# Usage: ./generate-changelog.sh <module-name> <last-tag>
# Output: Markdown formatted changelog

set -euo pipefail

MODULE="$1"
LAST_TAG="${2:-}"

if [ -z "$MODULE" ]; then
  echo "Error: Module name required" >&2
  exit 1
fi

echo "## Changes"
echo ""

# If there's a last tag, get commits since that tag
# Otherwise, get all commits for this module
if [ -n "$LAST_TAG" ]; then
  echo "Commits since \`$LAST_TAG\`:" >&2
  COMMIT_RANGE="${LAST_TAG}..HEAD"
else
  echo "All commits (first release):" >&2
  COMMIT_RANGE="HEAD"
fi

# Get commits that affected this module
# Format: - <message> (<short hash>)
COMMITS=$(git log --pretty=format:"- %s (\`%h\`)" --no-merges "$COMMIT_RANGE" -- "${MODULE}/" 2>/dev/null || true)

if [ -z "$COMMITS" ]; then
  echo "- Initial release"
  echo ""
  echo "No previous commit history." >&2
else
  echo "$COMMITS"
  echo ""

  # Count commits
  COMMIT_COUNT=$(echo "$COMMITS" | wc -l | tr -d ' ')
  echo "$COMMIT_COUNT commit(s) included in this release" >&2
fi

# Add module link
echo ""
echo "---"
echo ""
echo "**Module**: \`github.com/seb7887/gofw/$MODULE\`"
echo ""
echo "**Install**: \`go get github.com/seb7887/gofw/$MODULE@<version>\`"
