# Agent Identity System

This directory contains agent configurations for use with MCP Agent Mail.

## Identity Hierarchy

```
~/.config/agent-mail/
├── global-agents.json      # Global agents (available across all repos)
└── identity-pool.json      # Reserved identity names

<repo>/
└── .agent-mail/
    └── agents/
        └── <AgentName>.json  # Per-repo agent configurations
```

## Global vs Per-Repo Agents

### Global Agents
Global agents are shared across all projects. Useful for:
- Review agents that operate across codebases
- Research/documentation agents
- Your personal "main" coding agent

Location: `~/.config/agent-mail/global-agents.json`

### Per-Repo Agents
Project-specific agents that only exist within a single repo. Useful for:
- Specialized agents for specific tech stacks
- Feature-focused agents
- Test/QA agents for specific projects

Location: `<repo>/.agent-mail/agents/<AgentName>.json`

## Agent Identity Format

```json
{
  "name": "RedOwl",
  "type": "review",
  "scope": "global",
  "contact_policy": "auto",
  "profile": {
    "description": "Code review specialist focusing on Go best practices",
    "capabilities": ["code-review", "security-audit", "performance-review"],
    "preferred_languages": ["go", "typescript", "python"],
    "review_style": "thorough"
  },
  "created_at": "2026-01-16T00:00:00Z",
  "created_by": "human"
}
```

## Standard Agent Types

| Type | Purpose | Typical Scope |
|------|---------|---------------|
| `implementation` | Feature development, bug fixes | per-repo |
| `review` | Code review, PR feedback | global |
| `research` | Documentation, exploration | global |
| `testing` | Test writing, QA | per-repo |
| `planning` | Task breakdown, architecture | global |

## Contact Policies

| Policy | Behavior |
|--------|----------|
| `open` | Accept messages from any agent |
| `auto` | Allow messages from same thread/project |
| `contacts_only` | Require prior approval |
| `block_all` | Human-controlled only |

## Identity Naming Convention

Agent names use the adjective+noun format:
- `RedOwl`, `BlueArchitect`, `GreenCastle`
- Must be unique within scope (global or per-repo)
- Generated via Agent Mail or chosen manually

## Registration

### Via CLI
```bash
# Global agent
./scripts/register-agent.sh --global --name RedOwl --type review

# Per-repo agent
./scripts/register-agent.sh --name GreenBuilder --type implementation
```

### Via MCP
```
register_agent(
    project_key="myproject",
    agent_name="GreenBuilder",
    profile={"type": "implementation", "scope": "repo"}
)
```

## Example Global Agents

This repo includes example configurations:

| Agent | Type | Purpose |
|-------|------|---------|
| `RedOwl` | review | PR review with Go best practices |
| `BlueArchitect` | planning | Task breakdown and architecture |
| `SilverFox` | research | Documentation and exploration |
