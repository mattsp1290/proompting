# Claude Code Setup Prompt

## Prerequisites

This prompt should be executed AFTER `proompts/initial-prompt.md` has been completed and a `tasks.yaml` file exists in the project root.

## Agent Instructions

You are setting up a Claude Code environment for optimal AI-assisted development on the Proompting project. Your goal is to configure the workspace for maximum productivity when working with task-based development using Claude Code CLI.

**Research Capabilities**: Claude Code has built-in `WebSearch` and `WebFetch` tools for real-time web research. Use these tools when you need current information about technologies, best practices, or solutions that might not be in your training data.

## Setup Tasks

### 1. Verify Project Structure

First, verify the following files and directories exist:
- `/proompts/tasks/` directory
- `/docs/` directory with documentation files
- `tasks.yaml` in the project root
- Example task in `/proompts/tasks/example-todo-app.yaml`

### 2. Create Claude Code Configuration

Create a `CLAUDE.md` file in the project root with project-specific instructions:

#### CLAUDE.md
```markdown
# Project-Specific Instructions for Claude Code

## Task Management
- Always reference the current task ID when making changes
- Update task status in tasks.yaml when starting and completing work
- Commit changes with task ID in commit message: "[task-id] Description"

## Research and Information Gathering
- Use WebSearch for real-time research when needed
- Use WebFetch to retrieve specific documentation pages
- Research current best practices before implementing new technologies
- Verify compatibility and versions of tools/libraries before use
- Document research findings in task updates or separate docs

## File Organization
- Task files go in /proompts/tasks/
- Documentation goes in /docs/
- Follow existing project structure patterns

## Code Standards
- Use TypeScript for type safety when applicable
- Follow existing code style in the project
- Write tests for new features
- Document complex logic with comments

## Working with Tasks
1. Read tasks.yaml to find next pending task
2. Check all dependencies are completed
3. Research any unfamiliar technologies or requirements using WebSearch
4. Update status to 'in-progress'
5. Implement the task
6. Update status to 'completed'
7. Add update entry with timestamp

## AI Agent Guidelines
- Reference docs/agent-guidelines.md for detailed practices
- Use docs/prompt-templates.md for common scenarios
- Consult docs/task-format-guide.md for YAML structure
- Leverage WebSearch for up-to-date information and verification

## Claude Code Workflow
- Use the TodoWrite tool to track multi-step tasks
- Use the Task tool for complex exploration and research
- Prefer Edit over Write for modifying existing files
- Make atomic commits with descriptive messages
```

### 3. Create Quick Reference Commands

Create `.claude/commands.md` for common Claude Code workflows:

```markdown
# Quick Reference Commands for Claude Code

## Task Management

"What is the next available task in tasks.yaml with all dependencies completed?"

"I want to start working on task [TASK_ID]. Please update its status to in-progress and summarize what needs to be done."

"I've finished task [TASK_ID]. Please update its status to completed and add an update entry with today's date."

"Show me all in-progress tasks and their descriptions."

"Check if task [TASK_ID] has all its dependencies completed."

"Generate a progress summary showing completed, in-progress, and pending tasks."

## Research and Development

"Research the current best practices for [TECHNOLOGY/APPROACH] using web search before we implement this."

"Fetch the documentation page at [URL] and summarize the key points."

"Search for recent documentation or tutorials on [TOPIC] to ensure we're using the latest approach."

"Find current compatibility information between [TOOL_A] and [TOOL_B]."

"Look up the latest version and installation instructions for [PACKAGE/TOOL]."

"Research common issues and solutions for [SPECIFIC_PROBLEM] before troubleshooting."

## Problem Solving

"Before implementing [FEATURE], research if there are existing solutions or libraries we should consider."

"Search for current examples of [PATTERN/IMPLEMENTATION] to guide our approach."

## Claude Code Specific

"Use the Explore agent to find all files related to [FEATURE]."

"Create a todo list to track the implementation of [FEATURE]."

"Run the tests and fix any failures."

"Commit these changes with an appropriate message referencing task [TASK_ID]."
```

### 4. Create Task Schema (Optional but Helpful)

Create `docs/task-schema.json` for YAML validation:

```json
{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "type": "object",
  "required": ["metadata", "phases", "dependencies", "notes", "updates"],
  "properties": {
    "metadata": {
      "type": "object",
      "required": ["project", "description", "tech_stack"],
      "properties": {
        "project": { "type": "string" },
        "description": { "type": "string" },
        "tech_stack": {
          "type": "array",
          "items": { "type": "string" }
        }
      }
    },
    "phases": {
      "type": "array",
      "items": {
        "type": "object",
        "required": ["name", "tasks"],
        "properties": {
          "name": { "type": "string" },
          "tasks": {
            "type": "array",
            "items": {
              "type": "object",
              "required": ["id", "name", "description", "priority", "status"],
              "properties": {
                "id": { "type": "string" },
                "name": { "type": "string" },
                "description": { "type": "string" },
                "priority": {
                  "type": "string",
                  "enum": ["critical", "high", "medium", "low"]
                },
                "status": {
                  "type": "string",
                  "enum": ["pending", "in-progress", "completed", "blocked"]
                },
                "dependencies": {
                  "type": "array",
                  "items": { "type": "string" }
                },
                "references": {
                  "type": "array",
                  "items": { "type": "string" }
                }
              }
            }
          }
        }
      }
    },
    "dependencies": {
      "type": "object",
      "properties": {
        "external": {
          "type": "array",
          "items": {
            "type": "object",
            "required": ["project", "tasks"],
            "properties": {
              "project": { "type": "string" },
              "location": { "type": "string" },
              "purpose": { "type": "string" },
              "tasks": {
                "type": "array",
                "items": { "type": "string" }
              }
            }
          }
        }
      }
    },
    "notes": {
      "type": "array",
      "items": { "type": "string" }
    },
    "updates": {
      "type": "array",
      "items": { "type": "string" }
    }
  }
}
```

### 5. Create Git Configuration

Create or update `.gitignore`:

```
# OS
.DS_Store
Thumbs.db

# Dependencies
node_modules/
venv/
__pycache__/

# Build outputs
dist/
build/
*.egg-info/

# Logs
*.log
logs/

# Environment
.env
.env.local

# Testing
coverage/
.coverage
.pytest_cache/

# Temporary files
*.tmp
*.temp
.cache/
```

### 6. Create Quick Start Script

Create `scripts/task-status.sh`:

```bash
#!/bin/bash
# Quick task status checker

echo "=== Task Status Summary ==="
echo ""

# Count tasks by status
echo "Task Counts:"
echo "- Completed: $(grep -c "status: completed" tasks.yaml)"
echo "- In Progress: $(grep -c "status: in-progress" tasks.yaml)"
echo "- Pending: $(grep -c "status: pending" tasks.yaml)"
echo "- Blocked: $(grep -c "status: blocked" tasks.yaml)"

echo ""
echo "=== In-Progress Tasks ==="
awk '/- id:/{id=$3} /name:/{name=$0} /status: in-progress/{print id, name}' tasks.yaml

echo ""
echo "=== Next Available Tasks (pending with completed dependencies) ==="
# This is simplified - a more complex script would actually check dependencies
awk '/- id:/{id=$3} /name:/{name=$0} /status: pending/{if (!deps) print id, name} /dependencies:/{deps=1} /^ *- id:/{deps=0}' tasks.yaml | head -5
```

Make it executable:
```bash
chmod +x scripts/task-status.sh
```

### 7. Create Claude Code Tools Reference

Create `.claude/tools.md`:

```markdown
# Claude Code Tools Available in This Workspace

## Built-in Research Tools

### WebSearch
Real-time web search capability for research and information gathering.

#### When to Use
- Researching current best practices
- Finding up-to-date documentation
- Checking compatibility between tools
- Looking up recent tutorials or examples
- Verifying current package versions
- Troubleshooting with recent solutions

### WebFetch
Fetch and analyze content from specific URLs.

#### When to Use
- Reading specific documentation pages
- Analyzing API references
- Reviewing tutorials or guides

## Task and Exploration Tools

### Task (with Explore agent)
Use for codebase exploration and complex searches.

#### When to Use
- Finding files by patterns
- Searching for keywords across the codebase
- Understanding codebase structure

### TodoWrite
Track multi-step tasks and show progress.

#### When to Use
- Planning complex implementations
- Breaking down features into steps
- Tracking progress on multi-file changes

## Integration with Tasks
- Use before starting complex tasks to gather current information
- Research dependencies and their compatibility
- Find examples of similar implementations
- Verify that planned approaches are still current
```

### 8. Create README for Claude Code Users

Create `.claude/README.md`:

```markdown
# Claude Code Setup for Proompting

This directory contains Claude Code-specific configuration for the Proompting project.

## Quick Start

1. Open the project with `claude` command in terminal
2. Ask "What's the next available task?"
3. Reference CLAUDE.md for project conventions
4. Use commands.md for quick reference prompts

## Useful Commands

- Find next task: "What's the next available task?"
- Start work: "Start task [ID]"
- Check progress: "Show project progress"
- Update status: "Mark task [ID] as completed"
- Explore: "Use the Explore agent to find [PATTERN]"

## Key Files

- `/tasks.yaml` - Active project tasks
- `/proompts/tasks/` - Task file examples
- `/docs/` - Project documentation
- `/CLAUDE.md` - Project instructions for Claude Code
- `/.claude/tools.md` - Available tools and usage

## Tips

1. Always update task status when working
2. Reference task IDs in commits
3. Check dependencies before starting tasks
4. **Use WebSearch for research** before implementing unfamiliar tech
5. Use the TodoWrite tool for multi-step tasks
6. Use the Explore agent for codebase understanding
7. Research current best practices using web search when needed
```

## Verification Steps

After setup, verify:

1. [ ] CLAUDE.md exists in project root
2. [ ] .claude/ directory with configuration files
3. [ ] Task status script works: `./scripts/task-status.sh`
4. [ ] Documentation is accessible in /docs/

## Next Steps

1. Open the project with `claude` in terminal
2. Ask Claude Code to find the next available task
3. Begin working through tasks systematically
4. Refer to `CLAUDE.md` for project conventions
5. Use documentation in `/docs/` as needed

## Tips for Optimal Workflow

1. **Use TodoWrite**: Track complex tasks with the built-in todo list
2. **Research First**: Use WebSearch to research unfamiliar technologies
3. **Explore Agent**: Use the Task tool with Explore agent for codebase navigation
4. **Atomic Commits**: Make atomic commits with task ID references
5. **Stay Current**: Regularly research best practices for technologies you're using
6. **Plan Mode**: Use EnterPlanMode for non-trivial implementations

## Claude Code Capabilities

Claude Code provides powerful built-in capabilities:

- **WebSearch**: Real-time web search for documentation and solutions
- **WebFetch**: Fetch specific web pages for analysis
- **TodoWrite**: Track multi-step task progress
- **Task (Explore)**: Comprehensive codebase exploration
- **Git Integration**: Commit, create PRs, and manage version control
- **File Operations**: Read, Write, Edit with intelligent context

---

Your Claude Code environment is now optimized for task-based development with enhanced research capabilities. Happy coding!
