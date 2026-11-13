#!/usr/bin/env bash
# Pre-Commit Hook: runs detect-secrets audit + Terraform format/validation
# Install via: scripts/install-pre-commit-hook.sh

set -e

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

REPO_ROOT=$(git rev-parse --show-toplevel)
cd "$REPO_ROOT"

# ── 1. detect-secrets ────────────────────────────────────────────────────────
echo "🔍 Running detect-secrets audit..."
if [[ -f .secrets.baseline ]]; then
  detect-secrets scan --update .secrets.baseline

  UNAUDITED=$(python3 -c "
import json, sys
b = json.load(open('.secrets.baseline'))
u = [(f,s) for f,ss in b.get('results',{}).items() for s in ss if s.get('is_secret') is None]
print(len(u))
for f,s in u: print(f'  {f}:{s[\"line_number\"]} type={s[\"type\"]}', file=sys.stderr)
")
  if [[ "$UNAUDITED" -gt 0 ]]; then
    echo -e "${YELLOW}⚠ $UNAUDITED unaudited secret(s) found — run:${NC}"
    echo "    detect-secrets audit .secrets.baseline"
    echo "  then re-stage .secrets.baseline and commit again."
    exit 1
  fi
  git add .secrets.baseline
  echo -e "${GREEN}✓ detect-secrets: no unaudited secrets${NC}"
else
  echo -e "${YELLOW}⚠ No .secrets.baseline found — skipping detect-secrets${NC}"
fi

# ── 2. Terraform format + validation ─────────────────────────────────────────
if [[ -d "$REPO_ROOT/tests" ]]; then
  echo "🔍 Running Terraform pre-commit checks..."
  export TERRAFORM_DIR="$REPO_ROOT"
  export TF_VAR_ibmcloud_api_key="${TF_VAR_ibmcloud_api_key:-dummy-key-for-validation}"

  echo "📁 Repository: $REPO_ROOT"
  echo "🧪 Running tests..."

  cd "$REPO_ROOT/tests"
  if go test -v -timeout 5m -run "TestTerraformFormat|TestTerraformValidation"; then
    echo -e "${GREEN}✅ All pre-commit checks passed!${NC}"
  else
    echo -e "${RED}❌ Terraform checks failed!${NC}"
    exit 1
  fi
fi
