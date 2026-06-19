#!/bin/bash
set -e

cd "$(dirname "$0")/.."

echo "Building MCP server..."
go build -o ./bin/cerber-mcp ./cmd/mcp

echo "Starting Cerber Memory MCP Server..."
./bin/cerber-mcp
