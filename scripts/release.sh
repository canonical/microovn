#!/bin/bash
set -e

DELETE_EXISTING=0
SKIP_TESTS=0

usage() {
    echo "Usage: $0 [-d] [-h] [-s] [VERSION]"
    echo ""
    echo "Automates the MicroOVN release process."
    echo ""
    echo "Options:"
    echo "  -d        Delete existing PR and stable branches before running"
    echo "  -h        Show this help message and exit"
    echo "  -s        Skip upgrade tests"
    echo ""
    echo "Arguments:"
    echo "  VERSION   Optional. The release version (e.g., 24.03)."
    echo "            Defaults to the current year with .03 (e.g., $(date +%y.03))."
}

# Create test file content
create_test_content() {
    ln -s upgrade.bats "$1"
}

# Parse options
while getopts "dhs" opt; do
  case $opt in
    d)
      DELETE_EXISTING=1
      ;;
    h)
      usage
      exit 0
      ;;
    s)
      SKIP_TESTS=1
      ;;
    \?)
      echo "Invalid option: -$OPTARG" >&2
      usage
      exit 1
      ;;
  esac
done

shift $((OPTIND - 1))

# Handle optional version argument
if [ -n "$1" ]; then
    VERSION="$1"
else
    VERSION="$(date +%y.03)"
fi

PR_BRANCH="prepare-$VERSION"
STABLE_BRANCH="branch-$VERSION"
YAML_FILE="snap/snapcraft.yaml"

# Ensure we have the target file
if [ ! -f "$YAML_FILE" ]; then
    echo "Error: $YAML_FILE not found! Please run this script from the repository root."
    exit 1
fi

if [ "$DELETE_EXISTING" -eq 1 ]; then
    echo "-> [-d] Removing existing branches if they exist..."
    git branch -D "$PR_BRANCH" 2>/dev/null || true
    git branch -D "$STABLE_BRANCH" 2>/dev/null || true
    echo
fi

if git branch | grep -qE "($STABLE_BRANCH|$PR_BRANCH)"; then
    if git branch | grep -q "$STABLE_BRANCH"; then
        echo "Error: $STABLE_BRANCH already exists"
    fi

    if git branch | grep -q "$PR_BRANCH"; then
        echo "Error: $PR_BRANCH already exists"
    fi
    exit 1
fi



echo "Setting up for MicroOVN release:"

# 1. Sync with main
echo "-> Checking out and updating 'main'..."
git fetch origin
git checkout --detach origin/main

# 2. Create the working branch for the PR
echo "-> Creating PR branch: $PR_BRANCH"
git checkout -b "$PR_BRANCH"

echo
echo "Stabilising the branch:"

# Delete any build-base and any grade statement
sed -i -E "/^build-base:.*/d" "$YAML_FILE"
sed -i -E "s/^grade: devel/grade: stable/" "$YAML_FILE"
echo "-> Auto-pinning git sources to their latest stable tags..."
# Find line numbers with "source-type: git", reverse order to avoid line shifting issues
GIT_LINES=$(grep -n "source-type: git" "$YAML_FILE" | cut -d: -f1 | sort -nr)

for LINE in $GIT_LINES; do
    # Search backwards (up to 5 lines) to find the corresponding source URL
    START=$((LINE - 5))
    [ $START -lt 1 ] && START=1
    URL=$(sed -n "${START},${LINE}p" "$YAML_FILE" | grep -E "^[[:space:]]+source:" | tail -n1 | awk '{print $2}')

    if [ -n "$URL" ]; then
        echo "   -> Fetching latest tag for: $URL"
        # Fetch tags from remote, ignore betas/RCs/peeled refs, sort by version, grab the latest
        LATEST_TAG=$(git ls-remote --tags --sort="v:refname" "$URL" | grep -vE 'rc|beta|alpha|dev|\^\{\}' | tail -n1 | awk -F/ '{print $3}')

        if [ -n "$LATEST_TAG" ]; then
            echo "      Pinned to: $LATEST_TAG"
            # Capture the exact indentation of the source-type line
            INDENT=$(sed -n "${LINE}p" "$YAML_FILE" | grep -o '^[[:space:]]*')

            # 1. Delete any existing source-tag in the immediate vicinity (next 5 lines)
            sed -i "${LINE},+5 { /^[[:space:]]*source-tag:/d }" "$YAML_FILE"

            # 2. Insert the new source-tag immediately after source-type: git
            sed -i "${LINE} s/$/\n${INDENT}source-tag: ${LATEST_TAG}/" "$YAML_FILE"
        else
            echo "      [!] WARNING: Could not determine stable tag for $URL. Skipping."
        fi
    fi
done

# Calculate versions
CURRENT_YEAR=$(date +%y)
PREV_YEAR=$((CURRENT_YEAR - 1))
PREV_VERSION="${PREV_YEAR}.03"

# Create upgrade tests
echo "-> Creating upgrade tests..."

# Create upgrade test from previous stable release
UPGRADE_TEST_FILE="tests/upgrade_${PREV_VERSION}.bats"
if [ -f "$UPGRADE_TEST_FILE" ]; then
    echo "   Skipping: $UPGRADE_TEST_FILE already exists"
else
    create_test_content "$UPGRADE_TEST_FILE"
    echo "   Created: $UPGRADE_TEST_FILE"
fi

# Create upgrade test from previous LTS release if current year is even
if [ $((CURRENT_YEAR % 2)) -eq 0 ]; then
    PREV_LTS_YEAR=$((CURRENT_YEAR - 2))
    PREV_LTS_VERSION="${PREV_LTS_YEAR}.03"
    LTS_UPGRADE_TEST_FILE="tests/upgrade_${PREV_LTS_VERSION}.bats"
    if [ -f "$LTS_UPGRADE_TEST_FILE" ]; then
        echo "   Skipping: $LTS_UPGRADE_TEST_FILE already exists"
    else
        create_test_content "$LTS_UPGRADE_TEST_FILE"
        echo "   Created: $LTS_UPGRADE_TEST_FILE"
    fi
else
    PREV_LTS_VERSION=""
fi

# Commit 1
git add "$YAML_FILE" "$UPGRADE_TEST_FILE"
if [ -n "$PREV_LTS_VERSION" ]; then
    git add "$LTS_UPGRADE_TEST_FILE"
fi
git commit -m "Prepare for $VERSION: Stabilise build-base and pin git sources" --signoff

# Save the commit hash of the first commit so we can branch off it later
STABLE_COMMIT=$(git rev-parse HEAD)

if [ "$SKIP_TESTS" -eq 0 ]; then
    echo
    echo "Running upgrade tests..."
    echo -n "-> Building and running upgrade tests from $PREV_VERSION"
    if [ -n "$PREV_LTS_VERSION" ]; then
        echo " and from $PREV_LTS_VERSION"
    fi

    # Run the upgrade tests using make
    echo "-> Running upgrade tests (this will build snap and test images)..."
    FAILED_TESTS=""
    for test_file in "tests/upgrade_${PREV_VERSION}.bats" $(if [ -n "$PREV_LTS_VERSION" ]; then echo "tests/upgrade_${PREV_LTS_VERSION}.bats"; fi); do
        echo "   Testing: $test_file"
        
        # Check if test file exists
        if [ ! -f "$test_file" ]; then
            echo "   [!] ERROR: Test file $test_file not found"
            FAILED_TESTS="$FAILED_TESTS $test_file"
            continue
        fi
        
        # Run test with make
        if make "$test_file" 2>&1; then
            echo "   ✓ PASSED: $test_file"
        else
            FAILED_TESTS="$FAILED_TESTS $test_file"
            echo "   [!] FAILED: $test_file"
            echo "   [!] Stopping test run."
            break
        fi
    done

    if [ -n "$FAILED_TESTS" ]; then
        echo
        echo "================================================="
        echo "UPGRADE TESTS FAILED"
        echo "================================================="
        echo
        echo "Please fix the upgrade issues before continuing."
        echo "You can:"
        echo "  1. Manually run: make $test_file"
        echo "  2. Debug and fix the issues"
        echo "  3. Run this script again with the -d flag to delete existing branches"
        echo
        echo "Do you want to continue anyway? (yes/no)"
        read -r response
        if [[ ! "$response" =~ ^([yY][eE][sS]|[yY])$ ]]; then
            echo "Aborting release process."
            exit 1
        fi
        echo "Continuing despite test failures..."
    fi
else
    echo
    echo "-> Skipping upgrade tests (-s flag provided)"
fi

echo
echo "Returning to edge:"

# Insert build-base and grade: devel right after the base statement
sed -i -E "/^base:.*/a build-base: devel" "$YAML_FILE"
sed -i -E "s/^grade: stable/grade: devel/" "$YAML_FILE"

echo "-> Unpinning git sources..."
# Recalculate git line numbers and remove tags
GIT_LINES=$(grep -n "source-type: git" "$YAML_FILE" | cut -d: -f1 | sort -nr)
for LINE in $GIT_LINES; do
    sed -i "${LINE},+5 { /^[[:space:]]*source-tag:/d }" "$YAML_FILE"
done

echo "-> Removing upgrade test files..."

# Remove upgrade test from previous stable release
if [ -f "$UPGRADE_TEST_FILE" ]; then
    git rm "$UPGRADE_TEST_FILE"
    echo "   Removed: $UPGRADE_TEST_FILE"
fi

# Remove upgrade test from previous LTS release if it exists
if [ -n "$PREV_LTS_VERSION" ]; then
    if [ -f "$LTS_UPGRADE_TEST_FILE" ]; then
        git rm "$LTS_UPGRADE_TEST_FILE"
        echo "   Removed: $LTS_UPGRADE_TEST_FILE"
    fi
fi

# Commit 2
git add "$YAML_FILE"
git commit -m "Prepare for $VERSION: Restore build-base and unpin git sources" --signoff

echo
echo "Locally branching:"
# Create the branch-YY.MM branch pointing exactly at Commit 1
git branch "$STABLE_BRANCH" "$STABLE_COMMIT"

echo
echo "================================================="
echo "Script Succeeded"
echo "================================================="
echo "Next steps to finalize the release:"
echo "  1. Push your branch to your fork:"
echo "       git push <remote> $PR_BRANCH"
echo "  2. Open a Pull Request on GitHub named 'Prepare for $VERSION'."
echo "       $PR_BRANCH -> main"
echo "  3. Once the PR is merged into main, push $STABLE_BRANCH to origin"
echo "       git push origin $STABLE_BRANCH"
echo "================================================="
