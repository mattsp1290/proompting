#!/bin/bash

# Script: get-the-vibes-going-v2.sh
# Purpose: Set up proompts + Beads + MCP Agent Mail in a git project
# Usage: ./get-the-vibes-going-v2.sh <target_directory> [--migrate]

set -e

# Color codes
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

print_error() { echo -e "${RED}Error: $1${NC}" >&2; }
print_success() { echo -e "${GREEN}$1${NC}"; }
print_info() { echo -e "${YELLOW}$1${NC}"; }
print_header() { echo -e "${BLUE}=== $1 ===${NC}"; }

# Flags
MIGRATE_TASKS=false
SKIP_PROOMPTS=false

# Parse arguments
TARGET_DIR=""
while [[ $# -gt 0 ]]; do
    case $1 in
        --migrate)
            MIGRATE_TASKS=true
            shift
            ;;
        --skip-proompts)
            SKIP_PROOMPTS=true
            shift
            ;;
        --help|-h)
            echo "Usage: $0 <target_directory> [options]"
            echo ""
            echo "Options:"
            echo "  --migrate        Migrate existing tasks.yaml to Beads"
            echo "  --skip-proompts  Don't copy proompts directory"
            echo ""
            exit 0
            ;;
        *)
            TARGET_DIR="$1"
            shift
            ;;
    esac
done

# Check arguments
if [ -z "$TARGET_DIR" ]; then
    print_error "No directory provided"
    echo "Usage: $0 <target_directory> [--migrate]"
    exit 1
fi
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SOURCE_PROOMPTS_DIR="${SCRIPT_DIR}/proompts"

# Validate target
if [ ! -d "$TARGET_DIR" ]; then
    print_error "Directory '$TARGET_DIR' does not exist"
    exit 1
fi

TARGET_DIR="$(cd "$TARGET_DIR" && pwd)"

if [ ! -d "$TARGET_DIR/.git" ]; then
    print_error "Directory '$TARGET_DIR' is not a git repository"
    exit 1
fi

if [ ! -d "$SOURCE_PROOMPTS_DIR" ]; then
    print_error "Source proompts directory not found at '$SOURCE_PROOMPTS_DIR'"
    exit 1
fi

print_header "Setting up AI Agent Infrastructure"
print_info "Target: $TARGET_DIR"
echo ""

# ==========================================
# Step 1: Copy proompts directory
# ==========================================
print_header "Step 1: Proompts Directory"

TARGET_PROOMPTS_DIR="$TARGET_DIR/proompts"

if [ "$SKIP_PROOMPTS" = true ]; then
    print_info "Skipping proompts copy (--skip-proompts)"
elif [ -d "$TARGET_PROOMPTS_DIR" ]; then
    print_info "Proompts directory already exists"
    read -p "Overwrite existing files? (y/N): " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        print_info "Keeping existing proompts"
    else
        cp -r "$SOURCE_PROOMPTS_DIR"/* "$TARGET_PROOMPTS_DIR"/
        print_success "Updated proompts directory"
    fi
else
    mkdir -p "$TARGET_PROOMPTS_DIR"
    cp -r "$SOURCE_PROOMPTS_DIR"/* "$TARGET_PROOMPTS_DIR"/
    print_success "Created proompts directory"
fi

# ==========================================
# Step 2: Initialize Beads
# ==========================================
print_header "Step 2: Beads Task Graph"

if [ -d "$TARGET_DIR/.beads" ]; then
    print_info "Beads already initialized"
else
    if command -v bd &> /dev/null; then
        cd "$TARGET_DIR" && bd init
        print_success "Initialized Beads (.beads/)"
    else
        print_info "Beads CLI (bd) not found"
        echo "  Install with: npm install -g @beads/bd"
        echo "  Or: go install github.com/steveyegge/beads/cmd/bd@latest"
        echo ""
        echo "  After installing, run: cd $TARGET_DIR && bd init"
    fi
fi

# ==========================================
# Step 2b: Migrate tasks.yaml to Beads (if requested)
# ==========================================
if [ "$MIGRATE_TASKS" = true ]; then
    print_header "Step 2b: Migrate tasks.yaml to Beads"

    # Look for tasks.yaml in common locations
    TASKS_YAML=""
    if [ -f "$TARGET_DIR/tasks.yaml" ]; then
        TASKS_YAML="$TARGET_DIR/tasks.yaml"
    elif [ -f "$TARGET_DIR/proompts/tasks.yaml" ]; then
        TASKS_YAML="$TARGET_DIR/proompts/tasks.yaml"
    fi

    if [ -n "$TASKS_YAML" ]; then
        print_info "Found tasks.yaml at: $TASKS_YAML"

        # Primary migration method: Claude Code (handles any YAML structure)
        MIGRATE_SCRIPT="$SCRIPT_DIR/scripts/migrate-tasks-to-beads.sh"

        if [ -f "$MIGRATE_SCRIPT" ]; then
            if command -v claude &> /dev/null; then
                print_info "Using Claude Code to migrate tasks..."
                echo ""
                read -p "Run migration now? (y/N): " -n 1 -r
                echo
                if [[ $REPLY =~ ^[Yy]$ ]]; then
                    cd "$TARGET_DIR" && bash "$MIGRATE_SCRIPT" "$TASKS_YAML" --verify
                    print_success "Migration complete!"
                else
                    print_info "Run later with: bash $MIGRATE_SCRIPT $TASKS_YAML --verify"
                fi
            else
                print_info "Claude Code CLI not found"
                echo "  Install with: npm install -g @anthropic-ai/claude-code"
                echo ""
                echo "  Then run: bash $MIGRATE_SCRIPT $TASKS_YAML --verify"
            fi
        else
            print_info "Migration script not found at $MIGRATE_SCRIPT"
        fi
    else
        print_info "No tasks.yaml found to migrate"
        echo "  Looked in: $TARGET_DIR/tasks.yaml"
        echo "             $TARGET_DIR/proompts/tasks.yaml"
    fi
fi

# ==========================================
# Step 3: Check MCP Agent Mail
# ==========================================
print_header "Step 3: MCP Agent Mail"

if curl -s --max-time 2 http://localhost:8765/health &> /dev/null 2>&1; then
    print_success "Agent Mail server is running on :8765"
else
    print_info "Agent Mail server not detected"
    echo "  Install with:"
    echo '  curl -fsSL "https://raw.githubusercontent.com/Dicklesworthstone/mcp_agent_mail/main/scripts/install.sh" | bash -s -- --yes'
    echo ""
    echo "  Start server with: am"
    echo "  Web UI: http://localhost:8765"
fi

# ==========================================
# Step 4: Check Beads Viewer
# ==========================================
print_header "Step 4: Beads Viewer (bv)"

if command -v bv &> /dev/null; then
    print_success "Beads Viewer (bv) is installed"
else
    print_info "Beads Viewer (bv) not found"
    echo "  Install with: go install github.com/Dicklesworthstone/beads_viewer@latest"
    echo ""
    echo "  This provides robot flags for AI agents:"
    echo "    bv --robot-triage    # Intelligent task recommendations"
    echo "    bv --robot-plan      # Parallel execution tracks"
    echo "    bv --robot-insights  # PageRank, critical path"
fi

# ==========================================
# Step 5: Set up .gitignore
# ==========================================
print_header "Step 5: Git Configuration"

GITIGNORE_PATH="$TARGET_DIR/.gitignore"

if [ ! -f "$GITIGNORE_PATH" ]; then
    touch "$GITIGNORE_PATH"
    print_success "Created .gitignore"
fi

# Add entries if not present
add_gitignore() {
    local entry="$1"
    if ! grep -q "^${entry}$" "$GITIGNORE_PATH" 2>/dev/null; then
        echo "$entry" >> "$GITIGNORE_PATH"
        print_success "Added $entry to .gitignore"
    fi
}

add_gitignore "proompts/"
add_gitignore ".beads/.cache/"

# ==========================================
# Step 6: Create pre-commit hook (optional)
# ==========================================
print_header "Step 6: Pre-commit Hook"

HOOK_PATH="$TARGET_DIR/.git/hooks/pre-commit"

read -p "Install file reservation check hook? (y/N): " -n 1 -r
echo
if [[ $REPLY =~ ^[Yy]$ ]]; then
    cat > "$HOOK_PATH" << 'HOOKEOF'
#!/bin/bash
# Pre-commit hook: Check for file reservation conflicts
# Part of Beads + MCP Agent Mail integration

# Skip if agent mail server isn't running
if ! curl -s http://localhost:8765/health &> /dev/null 2>&1; then
    exit 0
fi

# Get staged files
STAGED_FILES=$(git diff --cached --name-only)

if [ -z "$STAGED_FILES" ]; then
    exit 0
fi

# Check for conflicts (implement based on your MCP integration)
# This is a placeholder - actual implementation would query Agent Mail API
# echo "Checking file reservations for: $STAGED_FILES"

# For now, just pass
exit 0
HOOKEOF
    chmod +x "$HOOK_PATH"
    print_success "Installed pre-commit hook"
else
    print_info "Skipped pre-commit hook"
fi

# ==========================================
# Summary
# ==========================================
echo ""
print_header "Setup Complete"
echo ""
echo "Directory structure:"
echo "  $TARGET_DIR/"
echo "  ├── proompts/              # Prompts and documentation"
echo "  │   ├── initial-prompt-beads.md"
echo "  │   ├── start-task.md"
echo "  │   ├── request-review.md"
echo "  │   ├── act-on-review.md"
echo "  │   └── docs/"
echo "  ├── .beads/                # Beads task graph (if initialized)"
echo "  └── .gitignore             # Updated"
echo ""
echo "Quick Start:"
echo "  1. Create task graph:  Use proompts/initial-prompt-beads.md"
echo "     OR migrate existing: $0 $TARGET_DIR --migrate"
echo "  2. Start working:      bv --robot-triage && bd ready"
echo "  3. Get next task:      Use proompts/start-task.md"
echo "  4. Request review:     Use proompts/request-review.md"
echo "  5. Act on feedback:    Use proompts/act-on-review.md"
echo ""
echo "Web UI (when Agent Mail running): http://localhost:8765"
echo ""
print_success "The vibes are going! Good luck with the project."
