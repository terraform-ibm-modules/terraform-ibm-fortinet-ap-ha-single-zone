#!/usr/bin/env bash
# Install Pre-Commit Hook for Terraform Validation
# This script installs the pre-commit hook that runs Terraform tests before each commit

set -e

# Colors for output
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Get the root directory of the repo
REPO_ROOT=$(git rev-parse --show-toplevel)

echo "📦 Installing Terraform pre-commit hook..."
echo ""

# Check if the hook script exists
if [ ! -f "$REPO_ROOT/scripts/pre-commit-hook.sh" ]; then
    echo "❌ Error: scripts/pre-commit-hook.sh not found"
    exit 1
fi

# Create hooks directory if it doesn't exist
mkdir -p "$REPO_ROOT/.git/hooks"

# Copy the hook
cp "$REPO_ROOT/scripts/pre-commit-hook.sh" "$REPO_ROOT/.git/hooks/pre-commit"

# Make it executable
chmod +x "$REPO_ROOT/.git/hooks/pre-commit"

echo -e "${GREEN}✅ Pre-commit hook installed successfully!${NC}"
echo ""
echo "The hook will run Terraform validation tests before each commit."
echo ""
echo "To skip the hook (not recommended), use:"
echo "  git commit --no-verify"
echo ""
echo -e "${YELLOW}💡 Tip: Make sure you have Go installed to run the tests${NC}"

# Made with Bob
