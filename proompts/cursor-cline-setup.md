# Cursor Cline Setup Prompt

## Prerequisites

This prompt should be executed AFTER `proompts/initial-prompt.md` has been completed and a `tasks.yaml` file exists in the project root.

## Agent Instructions

You are setting up a Cursor IDE environment with the Cline extension for optimal AI-assisted development on the Proompting project. Your goal is to configure the workspace for maximum productivity when working with task-based development.

## Setup Tasks

### 1. Verify Project Structure

First, verify the following files and directories exist:
- `/proompts/tasks/` directory
- `/docs/` directory with documentation files
- `tasks.yaml` in the project root
- Example task in `/proompts/tasks/example-todo-app.yaml`

### 2. Create Cursor Configuration

Create a `.cursor/` directory with the following configuration files:

#### .cursor/rules.md
```markdown
# Project-Specific Rules for Cursor

## Task Management
- Always reference the current task ID when making changes
- Update task status in tasks.yaml when starting and completing work
- Commit changes with task ID in commit message: "[task-id] Description"

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
3. Update status to 'in-progress'
4. Implement the task
5. Update status to 'completed'
6. Add update entry with timestamp

## AI Agent Guidelines
- Reference docs/agent-guidelines.md for detailed practices
- Use docs/prompt-templates.md for common scenarios
- Consult docs/task-format-guide.md for YAML structure
```

#### .cursor/prompts.md
```markdown
# Quick Reference Prompts for Cursor

## Find Next Task
"What is the next available task in tasks.yaml with all dependencies completed?"

## Start Task
"I want to start working on task [TASK_ID]. Please update its status to in-progress and summarize what needs to be done."

## Complete Task
"I've finished task [TASK_ID]. Please update its status to completed and add an update entry with today's date."

## Task Status Check
"Show me all in-progress tasks and their descriptions."

## Dependency Check
"Check if task [TASK_ID] has all its dependencies completed."

## Progress Report
"Generate a progress summary showing completed, in-progress, and pending tasks."
```

### 3. Create VSCode/Cursor Settings

Create or update `.vscode/settings.json`:

```json
{
  "files.associations": {
    "*.yaml": "yaml",
    "*.yml": "yaml"
  },
  "yaml.schemas": {
    "./docs/task-schema.json": "tasks.yaml"
  },
  "editor.formatOnSave": true,
  "editor.rulers": [80, 120],
  "files.exclude": {
    "**/.git": true,
    "**/.DS_Store": true,
    "**/node_modules": true,
    "**/dist": true,
    "**/build": true
  },
  "search.exclude": {
    "**/node_modules": true,
    "**/dist": true,
    "**/build": true,
    "**/.git": true
  },
  "cline.autocomplete.enabled": true,
  "cline.suggestions.enabled": true
}
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

### 5. Create Workspace Snippets

Create `.vscode/proompting.code-snippets`:

```json
{
  "New Task": {
    "prefix": "task",
    "body": [
      "- id: ${1:task-id}",
      "  name: \"${2:Task name}\"",
      "  description: \"${3:Detailed description}\"",
      "  priority: ${4|critical,high,medium,low|}",
      "  status: pending",
      "  dependencies: [${5}]"
    ],
    "description": "Create a new task entry"
  },
  "Task Update": {
    "prefix": "update",
    "body": [
      "- \"${1:$CURRENT_YEAR-$CURRENT_MONTH-$CURRENT_DATE}: ${2:Update description}\""
    ],
    "description": "Add a task update entry"
  },
  "External Dependency": {
    "prefix": "dep",
    "body": [
      "- project: \"${1:project-name}\"",
      "  location: \"${2:\\$GIT_DIRECTORY/}${3:project-path}\"",
      "  purpose: \"${4:Why this dependency exists}\"",
      "  tasks:",
      "    - \"${5:Required task}\""
    ],
    "description": "Add external dependency"
  }
}
```

### 6. Create Git Configuration

Create or update `.gitignore`:

```
# IDE
.cursor/
.vscode/
*.code-workspace

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

### 7. Create Quick Start Script

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

### 8. Create README for Cursor Users

Create `.cursor/README.md`:

```markdown
# Cursor + Cline Setup for Proompting

This directory contains Cursor-specific configuration for the Proompting project.

## Quick Start

1. Open tasks.yaml to see current project tasks
2. Use Cmd/Ctrl+K to ask Cline about next available tasks
3. Reference rules.md for project conventions
4. Use prompts.md for quick commands

## Useful Commands

- Find next task: "What's the next available task?"
- Start work: "Start task [ID]"
- Check progress: "Show project progress"
- Update status: "Mark task [ID] as completed"

## Key Files

- `/tasks.yaml` - Active project tasks
- `/proompts/tasks/` - Task file examples
- `/docs/` - Project documentation
- `/.cursor/rules.md` - Project rules for AI

## Tips

1. Always update task status when working
2. Reference task IDs in commits
3. Check dependencies before starting tasks
4. Use the example todo app as reference
5. Follow the agent guidelines in docs/
```

## Verification Steps

After setup, verify:

1. [ ] All configuration files are created
2. [ ] Task status script works: `./scripts/task-status.sh`
3. [ ] Cursor recognizes the .cursor directory
4. [ ] YAML validation works in tasks.yaml
5. [ ] Snippets are available in Cursor
6. [ ] Documentation is accessible

## Next Steps

1. Open `tasks.yaml` in Cursor
2. Use Cline to find the next available task
3. Begin working through tasks systematically
4. Refer to `.cursor/rules.md` for project conventions
5. Use documentation in `/docs/` as needed

## Tips for Optimal Workflow

1. **Use Cline Chat**: Ask about task dependencies and status
2. **Quick Commands**: Set up keyboard shortcuts for common task operations
3. **Split Views**: Keep tasks.yaml open in one pane while coding
4. **Terminal Integration**: Use the task-status script for quick checks
5. **Commit Often**: Make atomic commits with task ID references

---

Your Cursor + Cline environment is now optimized for task-based development. Happy coding!
