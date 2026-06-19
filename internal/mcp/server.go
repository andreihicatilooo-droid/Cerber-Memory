package mcp

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sync"

	"cerber-memory/internal/mvc/models"
	"cerber-memory/internal/pipeline"
	"cerber-memory/internal/vector"
)

// JSONRPCRequest represents an incoming JSON-RPC 2.0 method call.
type JSONRPCRequest struct {
	Jsonrpc string            `json:"jsonrpc"`
	ID      interface{}       `json:"id"`
	Method  string            `json:"method"`
	Params  json.RawMessage   `json:"params"`
}

// JSONRPCResponse represents a JSON-RPC 2.0 response or error.
type JSONRPCResponse struct {
	Jsonrpc string        `json:"jsonrpc"`
	ID      interface{}   `json:"id"`
	Result  interface{}   `json:"result,omitempty"`
	Error   *JSONRPCError `json:"error,omitempty"`
}

// JSONRPCError represents a JSON-RPC 2.0 error response.
type JSONRPCError struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// MCPServer implements the Model Context Protocol over JSON-RPC 2.0.
type MCPServer struct {
	handlers map[string]Handler
	mu       sync.RWMutex
}

// Handler is a function that processes an MCP method call and returns a result or error.
type Handler func(params json.RawMessage) (interface{}, error)

// NewMCPServer creates and initializes a new MCP server with all handlers registered.
func NewMCPServer() *MCPServer {
	server := &MCPServer{
		handlers: make(map[string]Handler),
	}
	server.registerHandlers()
	return server
}

// registerHandlers maps MCP method names to their handler functions.
func (s *MCPServer) registerHandlers() {
	s.handlers["initialize"] = s.handleInitialize
	s.handlers["tools/list"] = s.handleToolsList
	s.handlers["tools/call"] = s.handleToolsCall
	s.handlers["memory_semantic_search"] = s.handleSemanticSearch
	s.handlers["memory_save"] = s.handleSaveMemory
	s.handlers["memory_query_structured"] = s.handleQueryStructured
	s.handlers["credentials_get"] = s.handleCredentialsGet
	s.handlers["credentials_set"] = s.handleCredentialsSet
	s.handlers["memory_get_graph"] = s.handleGetGraph
}

// Start begins the MCP server, reading JSON-RPC requests from stdin and writing responses to stdout.
func (s *MCPServer) Start() {
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		var req JSONRPCRequest
		if err := json.Unmarshal([]byte(line), &req); err != nil {
			resp := &JSONRPCResponse{
				Jsonrpc: "2.0",
				ID:      nil,
				Error: &JSONRPCError{
					Code:    -32700,
					Message: "Parse error",
					Data:    err.Error(),
				},
			}
			if data, marshalErr := json.Marshal(resp); marshalErr == nil {
				fmt.Println(string(data))
			}
			continue
		}

		// Only send response if this is a request (has ID), not a notification
		if isRequest(&req) {
			response := s.handleRequest(&req)
			if response != nil {
				if data, err := json.Marshal(response); err == nil {
					fmt.Println(string(data))
				}
			}
		} else {
			// Still dispatch notification, but don't send response per JSON-RPC spec
			s.handleRequest(&req)
		}
	}
	if err := scanner.Err(); err != nil {
		log.Printf("stdin scanner error: %v", err)
	}
}

// handleRequest dispatches a JSON-RPC request to its registered handler and returns the response.
func (s *MCPServer) handleRequest(req *JSONRPCRequest) *JSONRPCResponse {
	s.mu.RLock()
	handler, exists := s.handlers[req.Method]
	s.mu.RUnlock()

	if !exists {
		return &JSONRPCResponse{
			Jsonrpc: "2.0",
			ID:      req.ID,
			Error: &JSONRPCError{
				Code:    -32601,
				Message: "Method not found",
			},
		}
	}

	result, err := handler(req.Params)
	if err != nil {
		return &JSONRPCResponse{
			Jsonrpc: "2.0",
			ID:      req.ID,
			Error: &JSONRPCError{
				Code:    -32603,
				Message: err.Error(),
			},
		}
	}

	return &JSONRPCResponse{
		Jsonrpc: "2.0",
		ID:      req.ID,
		Result:  result,
	}
}

// handleInitialize returns the MCP protocol version and server information.
func (s *MCPServer) handleInitialize(params json.RawMessage) (interface{}, error) {
	return map[string]interface{}{
		"protocolVersion": "2024-11-05",
		"capabilities": map[string]interface{}{
			"tools": map[string]interface{}{},
		},
		"serverInfo": map[string]interface{}{
			"name":    "cerber-memory-mcp",
			"version": "1.0.0",
		},
	}, nil
}

// handleToolsList returns the schema and descriptions for all available MCP tools.
func (s *MCPServer) handleToolsList(params json.RawMessage) (interface{}, error) {
	return map[string]interface{}{
		"tools": []map[string]interface{}{
			{
				"name":        "memory_semantic_search",
				"description": "Semantic search across all memory (tasks, ideas, documents, notes)",
				"inputSchema": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"query": map[string]interface{}{
							"type":        "string",
							"description": "Search query",
						},
						"limit": map[string]interface{}{
							"type":        "integer",
							"description": "Maximum results (default: 5)",
							"default":     5,
						},
					},
					"required": []string{"query"},
				},
			},
			{
				"name":        "memory_save",
				"description": "Save new memory item (task, idea, note, document, project)",
				"inputSchema": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"type": map[string]interface{}{
							"type":        "string",
							"enum":        []string{"task", "idea", "note", "document", "project"},
							"description": "Type of memory item",
						},
						"content": map[string]interface{}{
							"type":        "string",
							"description": "Main content/description",
						},
						"title": map[string]interface{}{
							"type":        "string",
							"description": "Title (optional)",
						},
						"category": map[string]interface{}{
							"type":        "string",
							"description": "Category (for tasks/ideas)",
						},
						"priority": map[string]interface{}{
							"type":        "string",
							"enum":        []string{"low", "medium", "high"},
							"description": "Priority level",
						},
						"tags": map[string]interface{}{
							"type":        "array",
							"items":       map[string]interface{}{"type": "string"},
							"description": "Tags for organization",
						},
					},
					"required": []string{"type", "content"},
				},
			},
			{
				"name":        "memory_query_structured",
				"description": "Query memory with filters (status, category, priority)",
				"inputSchema": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"entity_type": map[string]interface{}{
							"type":        "string",
							"enum":        []string{"task", "idea", "note", "document", "project"},
							"description": "Type of entity to query",
						},
						"status": map[string]interface{}{
							"type":        "string",
							"description": "Filter by status (pending, active, completed, archived)",
						},
						"category": map[string]interface{}{
							"type":        "string",
							"description": "Filter by category",
						},
						"priority": map[string]interface{}{
							"type":        "string",
							"enum":        []string{"low", "medium", "high"},
							"description": "Filter by priority",
						},
						"limit": map[string]interface{}{
							"type":        "integer",
							"description": "Maximum results (default: 10)",
							"default":     10,
						},
					},
					"required": []string{"entity_type"},
				},
			},
			{
				"name":        "credentials_get",
				"description": "Retrieve a stored credential securely",
				"inputSchema": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"key": map[string]interface{}{
							"type":        "string",
							"description": "Credential key name",
						},
					},
					"required": []string{"key"},
				},
			},
			{
				"name":        "credentials_set",
				"description": "Store a credential securely (encrypted)",
				"inputSchema": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"key": map[string]interface{}{
							"type":        "string",
							"description": "Credential key name",
						},
						"value": map[string]interface{}{
							"type":        "string",
							"description": "Credential value",
						},
						"category": map[string]interface{}{
							"type":        "string",
							"enum":        []string{"api_key", "password", "token", "database"},
							"description": "Credential category",
						},
					},
					"required": []string{"key", "value"},
				},
			},
			{
				"name":        "memory_get_graph",
				"description": "Get knowledge graph of all memory connections",
				"inputSchema": map[string]interface{}{
					"type": "object",
				},
			},
		},
	}, nil
}

// handleToolsCall invokes a named tool with the provided arguments and wraps result in CallToolResult.
func (s *MCPServer) handleToolsCall(params json.RawMessage) (interface{}, error) {
	var req struct {
		Name      string          `json:"name"`
		Arguments json.RawMessage `json:"arguments"`
	}
	if err := json.Unmarshal(params, &req); err != nil {
		return nil, fmt.Errorf("invalid parameters: %v", err)
	}

	// Prevent tools/call from being called recursively to avoid infinite loops
	if req.Name == "tools/call" {
		return nil, fmt.Errorf("tools/call cannot be called recursively")
	}

	s.mu.RLock()
	handler, exists := s.handlers[req.Name]
	s.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("tool not found: %s", req.Name)
	}

	result, err := handler(req.Arguments)
	if err != nil {
		return nil, err
	}

	// Wrap result in MCP CallToolResult structure
	resultJSON, _ := json.Marshal(result)
	return map[string]interface{}{
		"content": []map[string]interface{}{
			{
				"type": "text",
				"text": string(resultJSON),
			},
		},
	}, nil
}

// handleSemanticSearch performs vector-based semantic search across all indexed memory.
func (s *MCPServer) handleSemanticSearch(params json.RawMessage) (interface{}, error) {
	var req struct {
		Query string `json:"query"`
		Limit int    `json:"limit"`
	}
	if err := json.Unmarshal(params, &req); err != nil {
		return nil, fmt.Errorf("invalid parameters: %v", err)
	}

	if req.Limit <= 0 {
		req.Limit = 5
	}
	if req.Limit > 20 {
		req.Limit = 20
	}

	// Generate embedding for query
	embedding, err := pipeline.GenerateEmbedding(req.Query)
	if err != nil {
		return nil, fmt.Errorf("failed to generate embedding: %v", err)
	}

	// Search in vector DB
	results, err := vector.SearchSimilarVectors(embedding, req.Limit)
	if err != nil {
		return nil, fmt.Errorf("vector search failed: %v", err)
	}

	// Enrich results with full data from SQLite
	var searchResults []map[string]interface{}
	for _, result := range results {
		enriched := enrichSearchResult(result)
		if enriched != nil {
			searchResults = append(searchResults, enriched)
		}
	}

	return map[string]interface{}{
		"query":   req.Query,
		"count":   len(searchResults),
		"results": searchResults,
	}, nil
}

// handleSaveMemory creates a new memory item and queues it for vector indexing.
func (s *MCPServer) handleSaveMemory(params json.RawMessage) (interface{}, error) {
	var req struct {
		Type     string   `json:"type"`
		Content  string   `json:"content"`
		Title    string   `json:"title"`
		Category string   `json:"category"`
		Priority string   `json:"priority"`
		Tags     []string `json:"tags"`
	}
	if err := json.Unmarshal(params, &req); err != nil {
		return nil, fmt.Errorf("invalid parameters: %v", err)
	}

	if req.Content == "" {
		return nil, fmt.Errorf("content is required")
	}

	var entityID int64
	var entityType string

	switch req.Type {
	case "task":
		id, err := models.CreateTask(req.Title, req.Content, req.Priority, req.Category)
		if err != nil {
			return nil, err
		}
		entityID = id
		entityType = "task"

	case "idea":
		id, err := models.CreateIdea(req.Title, req.Content, req.Category, req.Priority)
		if err != nil {
			return nil, err
		}
		entityID = id
		entityType = "idea"

	case "note":
		id, err := models.CreateNote(req.Title, req.Content)
		if err != nil {
			return nil, err
		}
		entityID = id
		entityType = "note"

	case "document":
		id, err := models.CreateDocument(req.Title, req.Content)
		if err != nil {
			return nil, err
		}
		entityID = id
		entityType = "document"

	case "project":
		if req.Title == "" {
			return nil, fmt.Errorf("project title is required")
		}
		id, err := models.EnsureProject(req.Title, req.Content)
		if err != nil {
			return nil, err
		}
		entityID = id
		entityType = "project"

	default:
		return nil, fmt.Errorf("unknown type: %s", req.Type)
	}

	// Add tags
	if len(req.Tags) > 0 {
		if err := models.AddTagsToEntity(entityType, entityID, req.Tags); err != nil {
			log.Printf("Warning: failed to add tags: %v", err)
		}
	}

	// Queue for vector indexing
	queuedIndex := true
	if err := models.EnqueueVectorIndex(entityType, entityID); err != nil {
		log.Printf("Warning: failed to queue vector index: %v", err)
		queuedIndex = false
	}

	return map[string]interface{}{
		"id":           entityID,
		"type":         entityType,
		"status":       "created",
		"queued_index": queuedIndex,
	}, nil
}

// handleQueryStructured returns memory items filtered by entity type, status, category, and priority.
func (s *MCPServer) handleQueryStructured(params json.RawMessage) (interface{}, error) {
	var req struct {
		EntityType string `json:"entity_type"`
		Status     string `json:"status"`
		Category   string `json:"category"`
		Priority   string `json:"priority"`
		Limit      int    `json:"limit"`
	}
	if err := json.Unmarshal(params, &req); err != nil {
		return nil, fmt.Errorf("invalid parameters: %v", err)
	}

	if req.Limit <= 0 {
		req.Limit = 10
	}
	if req.Limit > 50 {
		req.Limit = 50
	}

	var results interface{}
	var err error

	switch req.EntityType {
	case "task":
		results, err = models.QueryTasks(req.Status, req.Priority, req.Category, req.Limit)
	case "idea":
		results, err = models.QueryIdeas(req.Status, req.Category, req.Priority, req.Limit)
	case "note":
		results, err = models.QueryNotes(req.Limit)
	case "document":
		results, err = models.QueryDocuments(req.Limit)
	case "project":
		results, err = models.QueryProjects(req.Limit)
	default:
		return nil, fmt.Errorf("unknown entity type: %s", req.EntityType)
	}

	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"entity_type": req.EntityType,
		"filters": map[string]interface{}{
			"status":   req.Status,
			"category": req.Category,
			"priority": req.Priority,
		},
		"results": results,
	}, nil
}

// handleCredentialsGet retrieves a stored credential by key.
func (s *MCPServer) handleCredentialsGet(params json.RawMessage) (interface{}, error) {
	var req struct {
		Key string `json:"key"`
	}
	if err := json.Unmarshal(params, &req); err != nil {
		return nil, fmt.Errorf("invalid parameters: %v", err)
	}

	value, err := models.GetCredential(req.Key)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"key":   req.Key,
		"value": value,
	}, nil
}

// handleCredentialsSet stores a credential securely, protected from vector indexing.
func (s *MCPServer) handleCredentialsSet(params json.RawMessage) (interface{}, error) {
	var req struct {
		Key      string `json:"key"`
		Value    string `json:"value"`
		Category string `json:"category"`
	}
	if err := json.Unmarshal(params, &req); err != nil {
		return nil, fmt.Errorf("invalid parameters: %v", err)
	}

	if req.Value == "" {
		return nil, fmt.Errorf("value is required")
	}

	id, err := models.SetCredential(req.Key, req.Value, req.Category)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"id":       id,
		"key":      req.Key,
		"status":   "saved",
		"category": req.Category,
	}, nil
}

// handleGetGraph returns the knowledge graph of all memory connections.
func (s *MCPServer) handleGetGraph(params json.RawMessage) (interface{}, error) {
	graph, err := models.GetMindmapGraph()
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"nodes": graph.Nodes,
		"edges": graph.Edges,
		"count": map[string]interface{}{
			"nodes": len(graph.Nodes),
			"edges": len(graph.Edges),
		},
	}, nil
}

// enrichSearchResult converts a vector search result into a structured response with entity metadata.
func enrichSearchResult(result vector.QdrantSearchResult) map[string]interface{} {
	payload := result.Payload
	entityType, ok := payload["entity_type"].(string)
	if !ok {
		return nil
	}

	entityID, _ := payload["entity_id"].(float64)

	var title, content string
	if t, ok := payload["title"].(string); ok {
		title = t
	}
	if c, ok := payload["text"].(string); ok {
		content = c
	}

	if len(content) > 200 {
		content = content[:200]
	}

	return map[string]interface{}{
		"id":         int64(entityID),
		"type":       entityType,
		"title":      title,
		"content":    content,
		"similarity": result.Score,
		"payload":    payload,
	}
}

// isRequest checks if this is a request (has ID) vs a notification (no ID). Per JSON-RPC 2.0 spec.
func isRequest(req *JSONRPCRequest) bool {
	return req.ID != nil
}

// min returns the smaller of two integers.
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
