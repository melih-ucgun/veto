#!/bin/bash

set -e

# Colors
GREEN='\033[0;32m'
RED='\033[0;31m'
NC='\033[0m'

echo "🚀 Starting Integration Tests..."

# Initialize Veto (Create dirs)
veto init --yes

# ==========================================
# Scenario 1: Happy Path (Install 'nano')
# ==========================================
echo -n "Test 1: Installing 'nano' (Happy Path)... "

# Ensure nano is NOT installed (it comes with base-devel but let's remove it first to be sure)
pacman -Rns --noconfirm nano > /dev/null 2>&1 || true

# Create config
cat <<EOF > success.yaml
resources:
  - id: install-nano
    type: pkg
    name: nano
    state: present
EOF

# Apply
OUTPUT=$(veto apply success.yaml 2>&1)
echo "$OUTPUT"

if echo "$OUTPUT" | grep -q "Successfully present package nano"; then
    echo -e "${GREEN}PASS${NC}"
else
    echo -e "${RED}FAIL${NC}"
    exit 1
fi

# Verify installation
if ! pacman -Qi nano > /dev/null 2>&1; then
    echo -e "${RED}FAIL (Package not found)${NC}"
    exit 1
fi

# ==========================================
# Scenario 2: Idempotency (Run again)
# ==========================================
echo -n "Test 2: Idempotency (Run again)... "

if veto apply success.yaml | grep -q "already in desired state"; then
    echo -e "${GREEN}PASS${NC}"
else
    echo -e "${RED}FAIL${NC}"
    exit 1
fi

# ==========================================
# Scenario 3: Rollback (Atomic Failure)
# ==========================================
echo -n "Test 3: Automatic Rollback... "

# Install 'vim' (valid) but fail on 'invalid-pkg'
# We remove vim first
pacman -Rns --noconfirm vim > /dev/null 2>&1 || true

cat <<EOF > fail.yaml
resources:
  - id: install-vim
    type: pkg
    name: vim
    state: present
  - id: install-invalid
    type: pkg
    name: this-package-does-not-exist-12345
    state: present
EOF

# Run Apply (Should fail)
set +e # Disable exit on error temporarily
OUTPUT=$(veto apply fail.yaml 2>&1)
EXIT_CODE=$?
set -e

if [ $EXIT_CODE -eq 0 ]; then
    echo -e "${RED}FAIL (Should have failed)${NC}"
    exit 1
fi

# Verify Rollback (Vim should be gone)
if pacman -Qi vim > /dev/null 2>&1; then
    echo -e "${RED}FAIL (Rollback failed - vim is still installed)${NC}"
    exit 1
else
    echo -e "${GREEN}PASS (Vim correctly reverted)${NC}"
fi

echo ""
echo -e "✅ All Integration Tests Passed!"
