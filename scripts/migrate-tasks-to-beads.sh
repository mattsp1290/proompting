#!/bin/bash
# migrate-tasks-to-beads.sh - Migrate any tasks YAML to Beads using Claude Code
#
# This is the primary migration method. Claude Code can understand any YAML structure
# and convert it to Beads format reliably.
#
# Usage:
#   ./migrate-tasks-to-beads.sh tasks.yaml
#   ./migrate-tasks-to-beads.sh tasks.yaml --verify
#   ./migrate-tasks-to-beads.sh *.yaml              # Multiple files

set -e

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

VERIFY_MODE=false
YAML_FILES=()

# Parse arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        --verify)
            VERIFY_MODE=true
            shift
            ;;
        --help|-h)
            echo "Usage: $0 <tasks.yaml> [tasks2.yaml ...] [--verify]"
            echo ""
            echo "Migrates task YAML files to Beads format using Claude Code."
            echo "Supports any YAML structure - Claude interprets and converts."
            echo ""
            echo "Options:"
            echo "  --verify    Run verification pass to ensure all tasks migrated"
            echo ""
            exit 0
            ;;
        *)
            if [ -f "$1" ]; then
                YAML_FILES+=("$1")
            else
                echo -e "${RED}Error: File not found: $1${NC}"
                exit 1
            fi
            shift
            ;;
    esac
done

if [ ${#YAML_FILES[@]} -eq 0 ]; then
    echo -e "${RED}Error: No YAML files provided${NC}"
    echo "Usage: $0 <tasks.yaml> [--verify]"
    exit 1
fi

# Check for Claude Code CLI
if ! command -v claude &> /dev/null; then
    echo -e "${RED}Error: Claude Code CLI not found${NC}"
    echo "Install with: npm install -g @anthropic-ai/claude-code"
    exit 1
fi

# Check for bd CLI
if ! command -v bd &> /dev/null; then
    echo -e "${RED}Error: Beads CLI (bd) not found${NC}"
    echo "Install with: npm install -g @beads/bd"
    exit 1
fi

# Initialize beads if needed
if [ ! -d ".beads" ]; then
    echo -e "${YELLOW}Initializing Beads...${NC}"
    bd init
fi

# Create temp file for the combined prompt
PROMPT_FILE=$(mktemp)
trap "rm -f $PROMPT_FILE" EXIT

# Build the migration prompt
cat > "$PROMPT_FILE" << 'PROMPT_HEADER'
You are migrating tasks from YAML files to Beads format.

## Your Goal
Convert ALL tasks from the YAML file(s) below into Beads using the `bd` CLI.

## Instructions

1. **Analyze the YAML structure** - each project may have a different format
2. **Extract all tasks** - look for anything that represents a task/todo item
3. **Create beads** using: `bd create "Task title" -p PRIORITY --label LABEL`
   - Priority: 0=critical, 1=high, 2=medium, 3=low
   - Label: use phase/category name, slugified (e.g., "project-setup")
4. **Add dependencies** using: `bd dep add CHILD_ID PARENT_ID`
5. **Mark completed tasks** using: `bd update ID --status closed`

## Important Rules
- Run `bd` commands directly - execute them, don't just output a script
- After creating tasks, run `bd list --json | jq length` to verify count
- Continue until ALL tasks are migrated
- If a task has subtasks/steps, create them as separate beads with dependencies

## YAML Content to Migrate:

PROMPT_HEADER

# Add each YAML file to the prompt
for yaml_file in "${YAML_FILES[@]}"; do
    echo "" >> "$PROMPT_FILE"
    echo "### File: $(basename "$yaml_file")" >> "$PROMPT_FILE"
    echo '```yaml' >> "$PROMPT_FILE"
    cat "$yaml_file" >> "$PROMPT_FILE"
    echo '```' >> "$PROMPT_FILE"
done

# Add final instructions
cat >> "$PROMPT_FILE" << 'PROMPT_FOOTER'

## Now migrate all tasks above to Beads.

Start by analyzing the structure, then create all tasks, add dependencies, and mark completed ones.
Report your progress and final count when done.
PROMPT_FOOTER

# Count approximate tasks for reference
APPROX_COUNT=0
for yaml_file in "${YAML_FILES[@]}"; do
    file_count=$(grep -cE "^\s*-?\s*(id:|name:|description:)" "$yaml_file" 2>/dev/null || echo "0")
    APPROX_COUNT=$((APPROX_COUNT + file_count / 2))
done
echo -e "${BLUE}Approximate tasks to migrate: $APPROX_COUNT${NC}"
echo ""

echo -e "${YELLOW}Starting Claude Code migration...${NC}"
echo -e "${YELLOW}Files: ${YAML_FILES[*]}${NC}"
echo ""

# Run Claude Code with the migration prompt
claude -p "$(cat "$PROMPT_FILE")" --allowedTools "Bash(bd *),Bash(jq *)"

# Get current count
BEAD_COUNT=$(bd list --json 2>/dev/null | jq length 2>/dev/null || echo "0")
echo ""
echo -e "${BLUE}=== Migration Result ===${NC}"
echo -e "Beads created: ${GREEN}$BEAD_COUNT${NC}"

# Verification pass if requested
if [ "$VERIFY_MODE" = true ]; then
    echo ""
    echo -e "${YELLOW}Running verification pass...${NC}"

    VERIFY_FILE=$(mktemp)
    cat > "$VERIFY_FILE" << 'VERIFY_HEADER'
Verify that ALL tasks from the YAML files have been migrated to Beads.

## Instructions
1. Run `bd list --json` to see all current beads
2. Compare against the YAML content below
3. If ANY tasks are missing, create them now
4. When done, say "VERIFICATION COMPLETE: X tasks total"

## YAML Content:
VERIFY_HEADER

    for yaml_file in "${YAML_FILES[@]}"; do
        echo "" >> "$VERIFY_FILE"
        echo "### $(basename "$yaml_file")" >> "$VERIFY_FILE"
        echo '```yaml' >> "$VERIFY_FILE"
        cat "$yaml_file" >> "$VERIFY_FILE"
        echo '```' >> "$VERIFY_FILE"
    done

    claude -p "$(cat "$VERIFY_FILE")" --allowedTools "Bash(bd *),Bash(jq *)"
    rm -f "$VERIFY_FILE"

    FINAL_COUNT=$(bd list --json 2>/dev/null | jq length 2>/dev/null || echo "0")
    echo ""
    echo -e "${GREEN}Final bead count: $FINAL_COUNT${NC}"
fi

echo ""
echo -e "${GREEN}=== Migration Complete ===${NC}"
echo ""
echo "View your task graph:"
echo "  bv                    # Interactive TUI"
echo "  bv --robot-triage     # AI recommendations"
echo "  bd ready              # List unblocked tasks"
