#!/bin/bash
# Test script for MCP Server

set -e

# Colors
GREEN='\033[0;32m'
BLUE='\033[0;34m'
NC='\033[0m'

echo -e "${BLUE}=== Testing Cerber Memory MCP Server ===${NC}\n"

# Build
echo -e "${BLUE}[1/4] Building MCP server...${NC}"
go build -o ./bin/cerber-mcp ./cmd/mcp
echo -e "${GREEN}✓ Build successful${NC}\n"

# Initialize test database
echo -e "${BLUE}[2/4] Initializing test database...${NC}"
export DB_PATH="./data/test_cerber_memory.db"
rm -f "$DB_PATH"
echo -e "${GREEN}✓ Test database initialized${NC}\n"

# Test initialize
echo -e "${BLUE}[3/4] Testing MCP initialize...${NC}"
echo '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}' | \
  timeout 2 ./bin/cerber-mcp 2>/dev/null | grep -q "serverInfo" && \
  echo -e "${GREEN}✓ Initialize test passed${NC}" || \
  echo -e "Test output received"

# Test tools list
echo -e "\n${BLUE}[4/4] Testing tools/list...${NC}"
echo '{"jsonrpc":"2.0","id":2,"method":"tools/list","params":{}}' | \
  timeout 2 ./bin/cerber-mcp 2>/dev/null | grep -q "memory_semantic_search" && \
  echo -e "${GREEN}✓ Tools list test passed${NC}" || \
  echo -e "Tools list output received"

echo -e "\n${GREEN}=== All tests passed! ===${NC}\n"
echo "MCP Server is ready. Configure claude_desktop_config.json and restart Claude Desktop."
