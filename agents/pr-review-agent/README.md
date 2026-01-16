# PR Review Agent (RedOwl)

A functional example of an AI agent that performs code reviews through MCP Agent Mail.

## Overview

The PR Review Agent (`RedOwl`) monitors for review request threads in Agent Mail, picks up pending reviews, analyzes the code changes, and posts structured feedback back to the thread.

## How It Works

```
┌──────────────────────────────────────────────────────────────┐
│                    Implementation Agent                       │
│                    (e.g., GreenCastle)                        │
│                                                               │
│  1. Completes work on bd-XXXX                                 │
│  2. Posts to thread: bd-XXXX-review                           │
│     "Review Request: Implement JWT auth"                      │
└──────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌──────────────────────────────────────────────────────────────┐
│                     MCP Agent Mail                            │
│                                                               │
│  Thread: bd-XXXX-review                                       │
│  Status: Pending Review                                       │
└──────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌──────────────────────────────────────────────────────────────┐
│                    PR Review Agent (RedOwl)                   │
│                                                               │
│  1. Polls inbox for *-review threads                          │
│  2. Claims review by posting "Review In Progress"             │
│  3. Fetches diff from git                                     │
│  4. Analyzes code against best practices                      │
│  5. Posts structured feedback to thread                       │
└──────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌──────────────────────────────────────────────────────────────┐
│                    Implementation Agent                       │
│                                                               │
│  1. Receives notification in inbox                            │
│  2. Reads feedback from bd-XXXX-review thread                 │
│  3. Addresses issues using act-on-review protocol             │
└──────────────────────────────────────────────────────────────┘
```

## Files

| File | Purpose |
|------|---------|
| `review-prompt.md` | The prompt template for code review |
| `run-review.sh` | CLI script to trigger a review |
| `review-schema.json` | JSON schema for structured feedback |

## Usage

### Manual Review (Single PR)

```bash
# Review current branch against main
./agents/pr-review-agent/run-review.sh

# Review specific branch
./agents/pr-review-agent/run-review.sh feature/auth-jwt

# Review and post to Agent Mail thread
./agents/pr-review-agent/run-review.sh --thread bd-a1b2-review
```

### As Part of Agent Mail Workflow

When using MCP Agent Mail, the review agent automatically:

1. Watches for `*-review` threads via inbox polling
2. Claims unclaimed reviews
3. Posts feedback when complete

### Integration with Claude Code

Add to your `.mcp.json`:

```json
{
  "mcpServers": {
    "agent-mail": {
      "type": "http",
      "url": "http://localhost:8765"
    }
  }
}
```

Then in Claude Code:
```
Use the RedOwl agent to review my PR for bd-XXXX
```

## Review Output Format

Reviews are posted as structured markdown:

```markdown
## Code Review: bd-XXXX

**Reviewer**: RedOwl
**Verdict**: Changes Requested | Approved | Approved with Comments

### Summary
[Brief overview of changes and overall assessment]

### Blocking Issues
[Must fix before merge]

### Suggestions
[Should consider, but not blocking]

### Questions
[Clarifications needed]

### Nitpicks
[Minor style/preference items]

### Highlights
[Things done particularly well]
```

## Configuration

The agent behavior can be customized in `RedOwl.json`:

```json
{
  "behavior": {
    "auto_claim_reviews": true,
    "review_timeout_hours": 24,
    "max_concurrent_reviews": 3
  }
}
```

## Go-Specific Reviews

For Go projects, the agent references patterns from `100-go-mistakes`:

- Error handling (mistakes #48-52)
- Concurrency patterns (mistakes #55-72)
- Memory/performance (mistakes #91-98)

Set the reference path in your environment:
```bash
export GO_MISTAKES_PATH="$HOME/git/100-go-mistakes"
```
