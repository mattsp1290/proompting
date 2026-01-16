# Project Planning with Beads

## Agent Instructions

You are a project manager at a FAANG level company, one project away from a life-changing promotion. This task breakdown will be executed by a team of AI agents working in parallel, coordinated through MCP Agent Mail with file reservations to prevent conflicts.

## Project Information

### Links to Relevant Documentation
{PLEASE FILL THIS OUT}

### Project Description
{PLEASE FILL THIS OUT}

### Technical Stack
{PLEASE FILL THIS OUT}

### Specific Requirements
{PLEASE FILL THIS OUT}

---

## Your Task

Analyze this project and create a comprehensive **Beads task graph** using the `bd` CLI. Beads provides dependency-aware, conflict-free task management for multi-agent execution.

---

## Output Format

Generate a shell script that creates the full task graph. The script should:

1. **Initialize Beads** (if not already initialized)
2. **Create all beads** with appropriate priorities
3. **Establish dependencies** between beads
4. **Add labels** for phase grouping

### Example Output

```bash
#!/bin/bash
# Project: {PROJECT_NAME}
# Generated: {DATE}

set -e

# Initialize beads if needed
if [ ! -d ".beads" ]; then
    bd init
fi

echo "Creating project beads..."

# ========================================
# Phase 1: Project Setup & Infrastructure
# ========================================

bd create "Initialize project with Vite + React + TypeScript" -p 0 --label setup
SETUP_VITE=$(bd list --json | jq -r '.[-1].id')

bd create "Configure ESLint, Prettier, and TypeScript strict mode" -p 1 --label setup
SETUP_LINT=$(bd list --json | jq -r '.[-1].id')
bd dep add $SETUP_LINT $SETUP_VITE

bd create "Set up Tailwind CSS with design system tokens" -p 1 --label setup
SETUP_TAILWIND=$(bd list --json | jq -r '.[-1].id')
bd dep add $SETUP_TAILWIND $SETUP_VITE

bd create "Configure testing framework (Vitest + Testing Library)" -p 1 --label setup
SETUP_TESTING=$(bd list --json | jq -r '.[-1].id')
bd dep add $SETUP_TESTING $SETUP_LINT

# ========================================
# Phase 2: Core Architecture
# ========================================

bd create "Implement API client with error handling and retries" -p 0 --label core
API_CLIENT=$(bd list --json | jq -r '.[-1].id')
bd dep add $API_CLIENT $SETUP_VITE

bd create "Set up global state management (Zustand/Jotai)" -p 0 --label core
STATE_MGMT=$(bd list --json | jq -r '.[-1].id')
bd dep add $STATE_MGMT $SETUP_VITE

bd create "Create authentication context and hooks" -p 0 --label core
AUTH_CONTEXT=$(bd list --json | jq -r '.[-1].id')
bd dep add $AUTH_CONTEXT $STATE_MGMT
bd dep add $AUTH_CONTEXT $API_CLIENT

# ... continue for all phases ...

echo ""
echo "Bead graph created! View with:"
echo "  bv                    # Interactive TUI"
echo "  bv --robot-triage     # AI-friendly recommendations"
echo "  bd ready              # List unblocked tasks"
```

---

## Bead Creation Guidelines

### Priority Levels
- `-p 0` = Critical (blocking other work)
- `-p 1` = High (important but not blocking)
- `-p 2` = Medium (standard work)
- `-p 3` = Low (nice to have)

### Labels (Phase Grouping)
Use `--label` to group beads by phase:
- `setup` - Project initialization
- `core` - Core architecture
- `auth` - Authentication/authorization
- `ui` - UI components
- `feature-{name}` - Feature-specific work
- `testing` - Test coverage
- `docs` - Documentation
- `deploy` - Deployment/CI

### Dependency Rules
1. Never create cycles
2. Every bead should have a clear dependency chain back to setup tasks
3. Use `bd dep add CHILD PARENT` (child depends on parent completing first)
4. Parallel work should share a common ancestor, not depend on each other

### Task Granularity
- Each bead should be completable in **under 750 lines of code**
- Tasks should be atomic enough for one agent to complete without coordination
- If a task requires multiple file areas, consider splitting by file area

---

## File Reservation Planning

For each major work area, note the file patterns that will need exclusive reservation:

```bash
# Example reservation notes (add as bead descriptions)
# Auth work: src/auth/**, tests/auth/**, src/hooks/useAuth*
# API client: src/api/**, src/lib/fetch*, tests/api/**
# UI components: src/components/{ComponentName}/**, tests/components/{ComponentName}/**
```

This helps agents claim appropriate file surfaces when they start work.

---

## Agent Mail Thread Convention

When beads are created, threads will be automatically created with pattern `bd-{id}`:

| Thread Pattern | Purpose |
|----------------|---------|
| `bd-{id}` | Main work discussion |
| `bd-{id}-review` | Code review |
| `bd-{id}-blocked` | Blocker escalation |

---

## Context Documentation

Place any important context in `proompts/docs/` for agents to reference. This includes:
- Architecture decisions
- API documentation
- Design system specs
- External service integration guides

---

## Verification Steps

After generating the script:

1. **Run it**: `chmod +x setup-beads.sh && ./setup-beads.sh`
2. **Verify graph**: `bv --robot-insights` to check for cycles
3. **Check ready work**: `bd ready` should show initial setup tasks
4. **Review plan**: `bv --robot-plan` to see parallel tracks

---

## Final Notes

This task graph will orchestrate multiple AI agents working simultaneously. Each agent will:
1. Query `bv --robot-triage` for recommendations
2. Claim a bead with `bd update ID --status in_progress`
3. Reserve files via MCP Agent Mail
4. Post progress to the bead's thread
5. Complete and release reservations

Make the graph comprehensive enough that agents can work independently, but with clear dependencies so nothing falls through the cracks.

The promotion depends on this. Make it count.
