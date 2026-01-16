#!/bin/bash
# setup-mcp.sh - Configure Claude Code MCP connection to Agent Mail
#
# Usage:
#   ./setup-mcp.sh              # Setup with defaults (localhost:8765)
#   ./setup-mcp.sh --global     # Setup globally for all projects
#   ./setup-mcp.sh --url URL    # Use custom Agent Mail URL

set -e

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

AGENT_MAIL_URL="http://localhost:8765"
SCOPE="project"

while [[ $# -gt 0 ]]; do
    case $1 in
        --global)
            SCOPE="user"
            shift
            ;;
        --url)
            AGENT_MAIL_URL="$2"
            shift 2
            ;;
        --help|-h)
            echo "Usage: $0 [options]"
            echo ""
            echo "Options:"
            echo "  --global     Configure globally (user scope)"
            echo "  --url URL    Agent Mail server URL (default: http://localhost:8765)"
            echo ""
            exit 0
            ;;
        *)
            shift
            ;;
    esac
done

echo -e "${BLUE}=== MCP Agent Mail Setup ===${NC}"
echo ""

# Check if claude CLI is available
if ! command -v claude &> /dev/null; then
    echo -e "${YELLOW}Warning: Claude Code CLI not found${NC}"
    echo "Install with: npm install -g @anthropic-ai/claude-code"
    echo ""
    echo "Manual setup:"
    echo "Add to .mcp.json:"
    echo '{'
    echo '  "mcpServers": {'
    echo '    "agent-mail": {'
    echo '      "type": "http",'
    echo "      \"url\": \"$AGENT_MAIL_URL\""
    echo '    }'
    echo '  }'
    echo '}'
    exit 0
fi

# Check if Agent Mail is running
echo -e "${YELLOW}Checking Agent Mail server...${NC}"
if curl -s "$AGENT_MAIL_URL/health" &> /dev/null 2>&1; then
    echo -e "${GREEN}Agent Mail server is running at $AGENT_MAIL_URL${NC}"
else
    echo -e "${YELLOW}Agent Mail server not responding at $AGENT_MAIL_URL${NC}"
    echo ""
    echo "To start Agent Mail:"
    echo '  curl -fsSL "https://raw.githubusercontent.com/Dicklesworthstone/mcp_agent_mail/main/scripts/install.sh" | bash -s -- --yes'
    echo "  am  # Start the server"
    echo ""
    read -p "Continue setup anyway? (y/N): " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        exit 0
    fi
fi

# Add MCP server
echo ""
echo -e "${YELLOW}Configuring Claude Code MCP...${NC}"

if [ "$SCOPE" = "user" ]; then
    echo "Adding agent-mail to user configuration..."
    claude mcp add --transport http --scope user agent-mail "$AGENT_MAIL_URL"
else
    echo "Adding agent-mail to project configuration..."
    claude mcp add --transport http --scope project agent-mail "$AGENT_MAIL_URL"
fi

echo ""
echo -e "${GREEN}Setup complete!${NC}"
echo ""
echo "Verify with:"
echo "  claude mcp list"
echo ""
echo "In Claude Code, check connection with:"
echo "  /mcp"
echo ""
echo "Available Agent Mail tools will include:"
echo "  - ensure_project      Create/register a project"
echo "  - register_agent      Register an agent identity"
echo "  - send_message        Post to a thread"
echo "  - file_reservation_*  Manage file reservations"
echo "  - resource://inbox/*  Check agent inbox"
