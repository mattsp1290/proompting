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
MAX_RETRIES=3

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

TASKS_CONTENT=$(cat "$TASKS_YAML")

# Count tasks in YAML (rough estimate)
YAML_TASK_COUNT=$(grep -c "^\s*- id:" "$TASKS_YAML" 2>/dev/null || grep -c "^\s*id:" "$TASKS_YAML" 2>/dev/null || echo "unknown")
echo -e "${BLUE}Estimated tasks in YAML: $YAML_TASK_COUNT${NC}"

# Migration prompt
MIGRATION_PROMPT="You are migrating tasks from a tasks.yaml file to Beads format.

## Instructions

1. Read the tasks.yaml content below
2. For each task, run the appropriate bd commands:
   - \`bd create \"Task title\" -p PRIORITY --label LABEL\` to create tasks
   - Priority: 0=critical, 1=high, 2=medium, 3=low
   - Label should be the phase name (slugified, e.g., 'project-setup')
3. After creating all tasks, add dependencies with \`bd dep add CHILD PARENT\`
4. Mark completed tasks with \`bd update ID --status closed\`

## Important
- Run bd commands directly - do not just output a script
- After each batch of commands, run \`bd list --json | jq length\` to verify count
- Continue until ALL tasks from the YAML are migrated
- The YAML has approximately $YAML_TASK_COUNT tasks

## tasks.yaml content:

\`\`\`yaml
$TASKS_CONTENT
\`\`\`

Start by creating all the tasks, then add dependencies, then mark completed ones.
Report progress as you go."

echo -e "${YELLOW}Starting Claude Code migration...${NC}"
echo ""

# Run Claude Code with the migration prompt
claude -p "$MIGRATION_PROMPT" --allowedTools "Bash(bd *),Bash(jq *)"

# Verify migration
echo ""
echo -e "${BLUE}=== Verifying Migration ===${NC}"

BEAD_COUNT=$(bd list --json 2>/dev/null | jq length 2>/dev/null || echo "0")
echo -e "Beads created: ${GREEN}$BEAD_COUNT${NC}"
echo -e "Expected (approx): ${YELLOW}$YAML_TASK_COUNT${NC}"

if [ "$VERIFY_MODE" = true ]; then
    echo ""
    echo -e "${YELLOW}Running verification with Claude Code...${NC}"

    VERIFY_PROMPT="Verify that all tasks from this tasks.yaml have been migrated to Beads.

## Instructions
1. Run \`bd list --json\` to see current beads
2. Compare against the tasks.yaml below
3. If any tasks are missing, create them
4. Report which tasks were missing (if any)

## tasks.yaml:
\`\`\`yaml
$TASKS_CONTENT
\`\`\`

Check and report status."

    claude -p "$VERIFY_PROMPT" --allowedTools "Bash(bd *),Bash(jq *)"
fi

echo ""
echo -e "${GREEN}=== Migration Complete ===${NC}"
echo ""
echo "View your task graph:"
echo "  bv                    # Interactive TUI"
echo "  bv --robot-triage     # AI recommendations"
echo "  bd ready              # List unblocked tasks"
