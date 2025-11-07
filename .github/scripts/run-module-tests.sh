#!/bin/bash
# Runs tests for a specific Go module
# Usage: ./run-module-tests.sh <module-name>
# Exit code: 0 if tests pass, 1 if tests fail or error

set -euo pipefail

MODULE="$1"

if [ -z "$MODULE" ]; then
  echo "Error: Module name required" >&2
  exit 1
fi

# Check if module directory exists
if [ ! -d "$MODULE" ]; then
  echo "Error: Module directory '$MODULE' not found" >&2
  exit 1
fi

# Change to module directory
cd "$MODULE"

echo "================================"
echo "Running tests for module: $MODULE"
echo "================================"
echo ""

# Check if there are any test files
TEST_FILES=$(find . -name "*_test.go" | wc -l | tr -d ' ')

if [ "$TEST_FILES" -eq 0 ]; then
  echo "⚠️  No test files found in $MODULE - skipping tests"
  echo ""
  echo "✅ Tests: SKIPPED (no tests)"
  exit 0
fi

echo "Found $TEST_FILES test file(s)"
echo ""

# Run tests with coverage
if go test -v ./... -cover 2>&1; then
  echo ""
  echo "✅ Tests: PASSED"
  exit 0
else
  echo ""
  echo "❌ Tests: FAILED"
  exit 1
fi
