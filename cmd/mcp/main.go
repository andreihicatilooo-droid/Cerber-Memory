package main

import (
	"log"
	"os"

	"cerber-memory/internal/mcp"
	"cerber-memory/internal/mvc/models"
	"github.com/joho/godotenv"
)

func main() {
	// Load .env file
	_ = godotenv.Load()

	// Initialize database
	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = "./data/cerber_memory.db"
	}

	models.InitDB(dbPath)
	defer models.CloseDB()

	// Create and start MCP server
	server := mcp.NewMCPServer()
	log.SetOutput(os.Stderr) // Log to stderr so stdout remains clean for MCP protocol
	log.Println("Cerber Memory MCP Server starting...")
	server.Start()
}
