#!/bin/bash
# run-review.sh - Trigger a PR review using the RedOwl agent
#
# Usage:
#   ./run-review.sh                    # Review current branch vs main
#   ./run-review.sh feature/auth       # Review specific branch
#   ./run-review.sh --thread bd-a1b2   # Post to Agent Mail thread
#   ./run-review.sh --output review.md # Write to file

set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# Defaults
BRANCH=""
BASE_BRANCH="main"
THREAD_ID=""
OUTPUT_FILE=""
PROJECT_KEY=""
AGENT_NAME="RedOwl"

# Parse arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        --thread)
            THREAD_ID="$2"
            shift 2
            ;;
        --base)
            BASE_BRANCH="$2"
            shift 2
            ;;
        --output|-o)
            OUTPUT_FILE="$2"
            shift 2
            ;;
        --project)
            PROJECT_KEY="$2"
            shift 2
            ;;
        --help|-h)
            echo "Usage: $0 [branch] [options]"
            echo ""
            echo "Options:"
            echo "  --thread ID    Post review to Agent Mail thread"
            echo "  --base BRANCH  Base branch for comparison (default: main)"
            echo "  --output FILE  Write review to file"
            echo "  --project KEY  Agent Mail project key"
            echo ""
            exit 0
            ;;
        *)
            BRANCH="$1"
            shift
            ;;
    esac
done

# Get current branch if not specified
if [ -z "$BRANCH" ]; then
    BRANCH=$(git branch --show-current)
fi

# Validate we're in a git repo
if ! git rev-parse --git-dir > /dev/null 2>&1; then
    echo -e "${RED}Error: Not in a git repository${NC}"
    exit 1
fi

# Check if branch exists
if ! git rev-parse --verify "$BRANCH" > /dev/null 2>&1; then
    echo -e "${RED}Error: Branch '$BRANCH' not found${NC}"
    exit 1
fi

echo -e "${BLUE}=== PR Review: $BRANCH → $BASE_BRANCH ===${NC}"
echo ""

# Gather context
echo -e "${YELLOW}Gathering context...${NC}"

# Get diff stats
DIFF_STAT=$(git diff "$BASE_BRANCH..$BRANCH" --stat 2>/dev/null || git diff "$BASE_BRANCH...$BRANCH" --stat)
DIFF_LINES=$(echo "$DIFF_STAT" | tail -1)

# Get commit messages
COMMITS=$(git log "$BASE_BRANCH..$BRANCH" --oneline 2>/dev/null || git log "$BASE_BRANCH...$BRANCH" --oneline)
COMMIT_COUNT=$(echo "$COMMITS" | wc -l | tr -d ' ')

# Get changed files
CHANGED_FILES=$(git diff "$BASE_BRANCH..$BRANCH" --name-only 2>/dev/null || git diff "$BASE_BRANCH...$BRANCH" --name-only)
FILE_COUNT=$(echo "$CHANGED_FILES" | wc -l | tr -d ' ')

echo "  Branch: $BRANCH"
echo "  Base: $BASE_BRANCH"
echo "  Commits: $COMMIT_COUNT"
echo "  Files changed: $FILE_COUNT"
echo "  $DIFF_LINES"
echo ""

# Create review context file
CONTEXT_FILE=$(mktemp)
cat > "$CONTEXT_FILE" << EOF
# PR Review Context

## Branch Information
- **Feature Branch**: $BRANCH
- **Base Branch**: $BASE_BRANCH
- **Commits**: $COMMIT_COUNT
- **Files Changed**: $FILE_COUNT

## Commits
\`\`\`
$COMMITS
\`\`\`

## Changed Files
\`\`\`
$CHANGED_FILES
\`\`\`

## Diff Statistics
\`\`\`
$DIFF_STAT
\`\`\`

## Full Diff
\`\`\`diff
$(git diff "$BASE_BRANCH..$BRANCH" 2>/dev/null || git diff "$BASE_BRANCH...$BRANCH")
\`\`\`
EOF

echo -e "${YELLOW}Context written to: $CONTEXT_FILE${NC}"
echo ""

# Determine output destination
if [ -n "$OUTPUT_FILE" ]; then
    echo -e "${GREEN}Review will be written to: $OUTPUT_FILE${NC}"
elif [ -n "$THREAD_ID" ]; then
    echo -e "${GREEN}Review will be posted to thread: $THREAD_ID${NC}"
else
    echo -e "${YELLOW}Review will be printed to stdout${NC}"
fi

echo ""
echo -e "${BLUE}=== Review Prompt ===${NC}"
echo ""
echo "To perform the review, use this context with Claude Code or your preferred AI:"
echo ""
echo "  1. Read the review prompt:"
echo "     cat $(dirname "$0")/review-prompt.md"
echo ""
echo "  2. Read the context file:"
echo "     cat $CONTEXT_FILE"
echo ""
echo "  3. If using Agent Mail, post the review:"
echo "     Thread: $THREAD_ID"
echo "     Agent: $AGENT_NAME"
echo ""

# If Claude Code is available, offer to run the review
if command -v claude &> /dev/null; then
    echo -e "${GREEN}Claude Code detected. To run review:${NC}"
    echo ""
    echo "  claude -p \"$(cat "$(dirname "$0")/review-prompt.md")\" < $CONTEXT_FILE"
    echo ""
fi

# Keep the context file around for the user
echo -e "${YELLOW}Context file preserved at: $CONTEXT_FILE${NC}"
echo "Delete when done: rm $CONTEXT_FILE"
