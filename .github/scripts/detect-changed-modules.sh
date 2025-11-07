#!/bin/bash
# Detects which Go modules have changed between HEAD and HEAD~1
# Outputs a JSON array of changed module names

set -euo pipefail

# List of all modules in the repository
MODULES=(
  "cfgmng"
  "eventbus"
  "ginsrv"
  "httpx"
  "idgen"
  "sietch"
  "wp"
)

# Get changed files between HEAD and HEAD~1
# If this is the first commit, compare with empty tree
if git rev-parse HEAD~1 >/dev/null 2>&1; then
  CHANGED_FILES=$(git diff --name-only HEAD~1 HEAD)
else
  # First commit - get all files
  CHANGED_FILES=$(git ls-files)
fi

# Array to store changed modules
CHANGED_MODULES=()

# Check each module for changes
for module in "${MODULES[@]}"; do
  # Check if any Go files or go.mod in this module changed
  if echo "$CHANGED_FILES" | grep -q "^${module}/.*\.go$\|^${module}/go\.mod$"; then
    CHANGED_MODULES+=("$module")
    echo "✓ Detected changes in module: $module" >&2
  fi
done

# Output as JSON array for GitHub Actions matrix
if [ ${#CHANGED_MODULES[@]} -eq 0 ]; then
  echo "[]"
  echo "No module changes detected" >&2
else
  # Convert bash array to JSON array
  printf '%s\n' "${CHANGED_MODULES[@]}" | jq -R . | jq -s .
fi
