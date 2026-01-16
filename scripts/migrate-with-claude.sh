#!/bin/bash
# migrate-with-claude.sh - Migrate tasks.yaml to Beads using Claude Code
#
# This script handles migrations that fail Python parsing by using Claude Code
# to interpret the YAML and generate bd commands.
#
# Usage:
#   ./migrate-with-claude.sh tasks.yaml
#   ./migrate-with-claude.sh tasks.yaml --verify  # Verify all tasks migrated

set -e

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

TASKS_YAML="$1"
VERIFY_MODE=false
MAX_VERIFY_RETRIES=3

# Parse arguments
shift || true
while [[ $# -gt 0 ]]; do
    case $1 in
        --verify)
            VERIFY_MODE=true
            shift
            ;;
        *)
            shift
            ;;
    esac
done

if [ -z "$TASKS_YAML" ] || [ ! -f "$TASKS_YAML" ]; then
    echo -e "${RED}Error: tasks.yaml file required${NC}"
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

# Count tasks in YAML (rough estimate)
YAML_TASK_COUNT=$(grep -c "^\s*- id:" "$TASKS_YAML" 2>/dev/null || grep -c "^\s*id:" "$TASKS_YAML" 2>/dev/null || echo "unknown")
echo -e "${BLUE}Estimated tasks in YAML: $YAML_TASK_COUNT${NC}"

# Create temp file for the prompt (handles large YAML files safely)
PROMPT_FILE=$(mktemp)
trap "rm -f $PROMPT_FILE" EXIT

cat > "$PROMPT_FILE" << 'PROMPT_HEADER'
You are migrating tasks from a tasks.yaml file to Beads format.

## Instructions

1. Read the tasks.yaml content below
2. For each task, run the appropriate bd commands:
   - `bd create "Task title" -p PRIORITY --label LABEL` to create tasks
   - Priority: 0=critical, 1=high, 2=medium, 3=low
   - Label should be the phase name (slugified, e.g., 'project-setup')
3. After creating all tasks, add dependencies with `bd dep add CHILD PARENT`
4. Mark completed tasks with `bd update ID --status closed`

## Important
- Run bd commands directly - do not just output a script
- After each batch of commands, run `bd list --json | jq length` to verify count
- Continue until ALL tasks from the YAML are migrated
PROMPT_HEADER

echo "- The YAML has approximately $YAML_TASK_COUNT tasks" >> "$PROMPT_FILE"
echo "" >> "$PROMPT_FILE"
echo "## tasks.yaml content:" >> "$PROMPT_FILE"
echo "" >> "$PROMPT_FILE"
echo '```yaml' >> "$PROMPT_FILE"
cat "$TASKS_YAML" >> "$PROMPT_FILE"
echo '```' >> "$PROMPT_FILE"
echo "" >> "$PROMPT_FILE"
echo "Start by creating all the tasks, then add dependencies, then mark completed ones." >> "$PROMPT_FILE"
echo "Report progress as you go." >> "$PROMPT_FILE"

echo -e "${YELLOW}Starting Claude Code migration...${NC}"
echo ""

# Run Claude Code with the migration prompt
claude -p "$(cat "$PROMPT_FILE")" --allowedTools "Bash(bd *),Bash(jq *)"

# Verify migration
echo ""
echo -e "${BLUE}=== Verifying Migration ===${NC}"

BEAD_COUNT=$(bd list --json 2>/dev/null | jq length 2>/dev/null || echo "0")
echo -e "Beads created: ${GREEN}$BEAD_COUNT${NC}"
echo -e "Expected (approx): ${YELLOW}$YAML_TASK_COUNT${NC}"

if [ "$VERIFY_MODE" = true ]; then
    RETRY_COUNT=0

    while [ $RETRY_COUNT -lt $MAX_VERIFY_RETRIES ]; do
        echo ""
        echo -e "${YELLOW}Running verification (attempt $((RETRY_COUNT + 1))/$MAX_VERIFY_RETRIES)...${NC}"

        # Create verification prompt in temp file
        VERIFY_FILE=$(mktemp)
        cat > "$VERIFY_FILE" << 'VERIFY_HEADER'
Verify that all tasks from this tasks.yaml have been migrated to Beads.

## Instructions
1. Run `bd list --json` to see current beads
2. Compare against the tasks.yaml below
3. If any tasks are missing, create them with `bd create`
4. If all tasks are present, respond with "VERIFICATION COMPLETE: All tasks migrated"
5. Report which tasks were missing (if any)

## tasks.yaml:
```yaml
VERIFY_HEADER
        cat "$TASKS_YAML" >> "$VERIFY_FILE"
        echo '```' >> "$VERIFY_FILE"
        echo "" >> "$VERIFY_FILE"
        echo "Check and report status. Say 'VERIFICATION COMPLETE' when done." >> "$VERIFY_FILE"

        VERIFY_OUTPUT=$(claude -p "$(cat "$VERIFY_FILE")" --allowedTools "Bash(bd *),Bash(jq *)" 2>&1)
        rm -f "$VERIFY_FILE"

        echo "$VERIFY_OUTPUT"

        # Check if verification is complete
        if echo "$VERIFY_OUTPUT" | grep -qi "VERIFICATION COMPLETE"; then
            echo ""
            echo -e "${GREEN}Verification successful!${NC}"
            break
        fi

        RETRY_COUNT=$((RETRY_COUNT + 1))

        if [ $RETRY_COUNT -lt $MAX_VERIFY_RETRIES ]; then
            echo -e "${YELLOW}Some tasks may be missing, retrying verification...${NC}"
        fi
    done

    if [ $RETRY_COUNT -eq $MAX_VERIFY_RETRIES ]; then
        echo -e "${YELLOW}Max verification retries reached. Please manually verify with: bd list${NC}"
    fi
fi

# Final count
FINAL_COUNT=$(bd list --json 2>/dev/null | jq length 2>/dev/null || echo "0")

echo ""
echo -e "${GREEN}=== Migration Complete ===${NC}"
echo -e "Final bead count: ${GREEN}$FINAL_COUNT${NC}"
echo ""
echo "View your task graph:"
echo "  bv                    # Interactive TUI"
echo "  bv --robot-triage     # AI recommendations"
echo "  bd ready              # List unblocked tasks"
