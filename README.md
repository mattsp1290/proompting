# proompting

Get up and vibe coding quickly with AI-powered task management.

## Overview

Proompting provides a framework for managing complex projects with AI-powered task execution. It uses **Beads** for git-backed task graphs with dependency tracking, and **MCP Agent Mail** for multi-agent coordination.

## Requirements

- Git
- Bash/Zsh shell
- [Beads CLI (bd)](https://github.com/steveyegge/beads) - Task graph management
- [Claude Code](https://github.com/anthropics/claude-code) - AI agent execution (optional but recommended)
- [Beads Viewer (bv)](https://github.com/Dicklesworthstone/beads_viewer) - AI task recommendations (optional)
- [MCP Agent Mail](https://github.com/Dicklesworthstone/mcp_agent_mail) - Multi-agent coordination (optional)

## Quick Start

```bash
# Set up a project with Beads task management
./get-the-vibes-going.sh /path/to/your-project

# Or with migration of existing tasks.yaml
./get-the-vibes-going.sh /path/to/your-project --migrate
```

## Workflow

The Beads workflow uses a git-backed task graph with dependency tracking, enabling smarter task prioritization and multi-agent coordination.

### Workflow Cycle

```
┌─────────────────────────────────────────────────────────────────┐
│                                                                 │
│  1. PLAN                                                        │
│     └── Use initial-prompt.md with AI agent               │
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
│         └── Posts review request via MCP Agent Mail             │
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
./get-the-vibes-going.sh /path/to/project
cd /path/to/project
```

**2. Create Task Graph**
- Open `proompts/initial-prompt.md`
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
- AI agent reserves files, creates branch, implements, commits

**5. Request Review**
- Use `proompts/request-review.md`
- Agent posts review request via MCP Agent Mail

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

## Beads Commands

```bash
bd init                    # Initialize beads in a project
bd create "Task" -p 1      # Create task (priority 0-3)
bd list                    # List all tasks
bd ready                   # Show unblocked tasks
bd update <id> --status closed  # Mark complete
bd dep add <child> <parent>     # Add dependency
bd graph --all             # Visualize dependencies
```

## Beads Viewer (AI Recommendations)

```bash
bv                         # Interactive TUI
bv --robot-triage          # Smart task recommendations
bv --robot-plan            # Parallel execution tracks
bv --robot-insights        # PageRank, critical path analysis
```

## MCP Agent Mail Integration

Agent Mail enables multi-agent coordination:

```bash
# Start the server
am

# Web UI
open http://localhost:8765
```

**Key features used in the workflow:**
- **File reservations** - Prevent conflicts when multiple agents work in parallel
- **Message threads** - Async communication for reviews and announcements
- **Agent inboxes** - Each agent has a mailbox for notifications

## Prompt Reference

| Prompt | Purpose |
|--------|---------|
| `initial-prompt.md` | Create task graph with Beads |
| `start-task.md` | Begin working on a selected task |
| `request-review.md` | Request code review via Agent Mail |
| `act-on-review.md` | Address review feedback |

### Setup Guides

| Guide | Purpose |
|-------|---------|
| `claude-code-setup.md` | Configure Claude Code |
| `cursor-cline-setup.md` | Configure Cursor/Cline |

## Installation

### Install Beads CLI

```bash
go install github.com/steveyegge/beads/cmd/bd@latest

# Add to PATH if needed
export PATH="$HOME/go/bin:$PATH"
```

### Install Beads Viewer

```bash
go install github.com/Dicklesworthstone/beads_viewer@latest
```

### Install Claude Code

```bash
npm install -g @anthropic-ai/claude-code
```

### Install MCP Agent Mail

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
│   ├── initial-prompt.md  # Task planning
│   ├── start-task.md            # Begin task
│   ├── request-review.md        # Request review
│   ├── act-on-review.md         # Address feedback
│   └── docs/                    # Documentation
├── .beads/                      # Beads task graph
│   └── beads.db                 # Task database
└── .gitignore                   # Includes proompts/, .beads/.cache/
```

## Migrating from tasks.yaml

If you have existing `tasks.yaml` files:

```bash
# Using Claude Code (handles any YAML structure)
./scripts/migrate-tasks-to-beads.sh path/to/tasks.yaml --verify

# During project setup
./get-the-vibes-going.sh /path/to/project --migrate
```

See [Migration Guide](proompts/docs/migration-guide.md) for details.

## Documentation

- [Migration Guide](proompts/docs/migration-guide.md) - Migrate tasks.yaml to Beads
- [Beads + Agent Mail Integration](proompts/docs/beads-agent-mail-integration.md) - Multi-agent setup
- [Agent Guidelines](proompts/docs/agent-guidelines.md) - Best practices for AI agents

## Notes

- The `proompts/` directory is gitignored to keep project-specific prompts local
- Beads data in `.beads/` is git-tracked (except `.beads/.cache/`)
