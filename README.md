# proompting

Get up and vibe coding quickly with AI-powered task management.

## Overview

Proompting provides a framework for managing complex projects with AI-powered task execution. It includes prompt templates that structure project planning and task management for maximum efficiency.

**Two workflows are available:**
- **Beads Workflow** (Recommended) - Git-backed task graph with dependency tracking
- **Legacy Workflow** - Static tasks.yaml file management

## Requirements

- Git
- Bash/Zsh shell

**For Beads Workflow (recommended):**
- [Beads CLI (bd)](https://github.com/steveyegge/beads) - Task graph management
- [Claude Code](https://github.com/anthropics/claude-code) - AI agent execution
- [Beads Viewer (bv)](https://github.com/Dicklesworthstone/beads_viewer) - Optional, for AI task recommendations

## Quick Start

### Option 1: New Project with Beads (Recommended)

```bash
# Set up a project with Beads task management
./get-the-vibes-going-v2.sh /path/to/your-project

# Or with migration of existing tasks.yaml
./get-the-vibes-going-v2.sh /path/to/your-project --migrate
```

### Option 2: Legacy Setup (tasks.yaml)

```bash
# Set up with classic tasks.yaml workflow
./get-the-vibes-going.sh /path/to/your-project
```

## Beads Workflow (Recommended)

The Beads workflow uses a git-backed task graph with dependency tracking, enabling smarter task prioritization and multi-agent coordination.

### Workflow Cycle

```
┌─────────────────────────────────────────────────────────────────┐
│                                                                 │
│  1. PLAN                                                        │
│     └── Use initial-prompt-beads.md with AI agent               │
│         └── Creates task graph in .beads/                       │
│                                                                 │
│  2. SELECT TASK                                                 │
│     └── bv --robot-triage (or bd ready)                         │
│         └── Use start-task.md to begin work                     │
│                                                                 │
│  3. IMPLEMENT                                                   │
│     └── AI agent works on the task                              │
│         └── Creates branch, writes code, commits                │
│                                                                 │
│  4. REQUEST REVIEW                                              │
│     └── Use request-review.md                                   │
│         └── Posts review request (MCP Agent Mail or PR)         │
│                                                                 │
│  5. ACT ON REVIEW                                               │
│     └── Use act-on-review.md                                    │
│         └── Address feedback, update code                       │
│                                                                 │
│  6. COMPLETE                                                    │
│     └── bd update <id> --status closed                          │
│         └── Return to step 2 for next task                      │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

### Step-by-Step

**1. Initialize Project**
```bash
./get-the-vibes-going-v2.sh /path/to/project
cd /path/to/project
```

**2. Create Task Graph**
- Open `proompts/initial-prompt-beads.md`
- Customize for your project requirements
- Pass to an AI agent (Claude Code, Cursor, etc.)
- The agent creates beads using `bd create` commands

**3. Get Next Task**
```bash
# AI-optimized recommendations
bv --robot-triage

# Or simple list of ready tasks
bd ready
```

**4. Start Working**
- Copy output from `bv --robot-triage`
- Use `proompts/start-task.md` with the task details
- AI agent creates branch, implements, commits

**5. Request Review**
- Use `proompts/request-review.md`
- Agent posts review request via MCP Agent Mail or creates PR

**6. Address Feedback**
- Use `proompts/act-on-review.md`
- Agent addresses review comments
- Repeat review cycle if needed

**7. Complete and Continue**
```bash
# Mark task complete
bd update <bead-id> --status closed

# Get next task
bv --robot-triage
```

### Beads Commands

```bash
bd init                    # Initialize beads in a project
bd create "Task" -p 1      # Create task (priority 0-3)
bd list                    # List all tasks
bd ready                   # Show unblocked tasks
bd update <id> --status closed  # Mark complete
bd dep add <child> <parent>     # Add dependency
bd graph --all             # Visualize dependencies
```

### Beads Viewer (AI Recommendations)

```bash
bv                         # Interactive TUI
bv --robot-triage          # Smart task recommendations
bv --robot-plan            # Parallel execution tracks
bv --robot-insights        # PageRank, critical path analysis
```

## Legacy Workflow (tasks.yaml)

The original workflow using static YAML files for task management.

### Workflow Cycle

```
┌─────────────────────────────────────────────────────────────────┐
│                                                                 │
│  1. PLAN                                                        │
│     └── Update initial-prompt.md for your project               │
│         └── Pass to AI agent                                    │
│             └── Agent creates tasks.yaml                        │
│                                                                 │
│  2. GET NEXT TASK                                               │
│     └── Use get-next-task.md with AI agent                      │
│         └── Agent reads tasks.yaml                              │
│             └── Produces task execution prompt                  │
│                                                                 │
│  3. EXECUTE TASK                                                │
│     └── Use the generated task prompt                           │
│         └── AI agent implements the task                        │
│                                                                 │
│  4. REVIEW                                                      │
│     └── Use pr-review.md                                        │
│         └── Agent reviews changes                               │
│                                                                 │
│  5. ADDRESS FEEDBACK                                            │
│     └── AI agent addresses review comments                      │
│         └── Return to step 4 if more feedback                   │
│             └── Otherwise return to step 2                      │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

### Legacy Prompts

| Prompt | Purpose |
|--------|---------|
| `initial-prompt.md` | Customize and pass to AI to generate tasks.yaml |
| `get-next-task.md` | AI reads tasks.yaml and produces task prompt |
| `pr-review.md` | AI reviews changes and provides feedback |

## Migrating from tasks.yaml to Beads

If you have existing `tasks.yaml` files, migrate them to Beads:

```bash
# Using Claude Code (handles any YAML structure)
./scripts/migrate-tasks-to-beads.sh path/to/tasks.yaml --verify

# During project setup
./get-the-vibes-going-v2.sh /path/to/project --migrate
```

See [Migration Guide](proompts/docs/migration-guide.md) for detailed instructions.

## Prompt Reference

### Beads Workflow Prompts

| Prompt | Purpose |
|--------|---------|
| `initial-prompt-beads.md` | Create task graph with Beads |
| `start-task.md` | Begin working on a selected task |
| `request-review.md` | Request code review |
| `act-on-review.md` | Address review feedback |

### Legacy Prompts

| Prompt | Purpose |
|--------|---------|
| `initial-prompt.md` | Generate tasks.yaml |
| `get-next-task.md` | Get next task from tasks.yaml |
| `pr-review.md` | Review changes |

### Setup Guides

| Guide | Purpose |
|-------|---------|
| `claude-code-setup.md` | Configure Claude Code |
| `cursor-cline-setup.md` | Configure Cursor/Cline |

## Installation

### Install Beads CLI

```bash
# Via Go
go install github.com/steveyegge/beads/cmd/bd@latest

# Add to PATH if needed
export PATH="$HOME/go/bin:$PATH"
```

### Install Beads Viewer (Optional)

```bash
go install github.com/Dicklesworthstone/beads_viewer@latest
```

### Install Claude Code (Optional)

```bash
npm install -g @anthropic-ai/claude-code
```

### Install MCP Agent Mail (Optional)

For multi-agent coordination:

```bash
curl -fsSL "https://raw.githubusercontent.com/Dicklesworthstone/mcp_agent_mail/main/scripts/install.sh" | bash -s -- --yes

# Start server
am

# Web UI at http://localhost:8765
```

## Project Structure

```
your-project/
├── proompts/                    # Prompt templates (gitignored)
│   ├── initial-prompt-beads.md  # Task planning (Beads)
│   ├── start-task.md            # Begin task
│   ├── request-review.md        # Request review
│   ├── act-on-review.md         # Address feedback
│   ├── initial-prompt.md        # Task planning (legacy)
│   ├── get-next-task.md         # Get next task (legacy)
│   ├── pr-review.md             # Review (legacy)
│   └── docs/                    # Documentation
├── .beads/                      # Beads task graph
│   └── beads.db                 # Task database
└── .gitignore                   # Includes proompts/, .beads/.cache/
```

## Documentation

- [Migration Guide](proompts/docs/migration-guide.md) - Migrate tasks.yaml to Beads
- [Beads + Agent Mail Integration](proompts/docs/beads-agent-mail-integration.md) - Multi-agent setup
- [Agent Guidelines](proompts/docs/agent-guidelines.md) - Best practices for AI agents
- [Prompt Templates](proompts/docs/prompt-templates.md) - Template reference

## Notes

- The `proompts/` directory is gitignored to keep project-specific prompts local
- Beads data in `.beads/` is git-tracked (except `.beads/.cache/`)
- Both workflows can coexist - use whichever fits your project
