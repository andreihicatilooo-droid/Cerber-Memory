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

type JSONRPCRequest struct {
	Jsonrpc string            `json:"jsonrpc"`
	ID      interface{}       `json:"id"`
	Method  string            `json:"method"`
	Params  json.RawMessage   `json:"params"`
}

type JSONRPCResponse struct {
	Jsonrpc string      `json:"jsonrpc"`
	ID      interface{} `json:"id"`
	Result  interface{} `json:"result,omitempty"`
	Error   *JSONRPCError `json:"error,omitempty"`
}

type JSONRPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

type MCPServer struct {
	handlers map[string]Handler
	mu       sync.RWMutex
}

type Handler func(params json.RawMessage) (interface{}, error)

func NewMCPServer() *MCPServer {
	server := &MCPServer{
		handlers: make(map[string]Handler),
	}
	server.registerHandlers()
	return server
}

func (s *MCPServer) registerHandlers() {
	s.handlers["initialize"] = s.handleInitialize
	s.handlers["tools/list"] = s.handleToolsList
	s.handlers["memory/semantic_search"] = s.handleSemanticSearch
	s.handlers["memory/save"] = s.handleSaveMemory
	s.handlers["memory/query_structured"] = s.handleQueryStructured
	s.handlers["credentials/get"] = s.handleCredentialsGet
	s.handlers["credentials/set"] = s.handleCredentialsSet
	s.handlers["memory/get_graph"] = s.handleGetGraph
}

func (s *MCPServer) Start() {
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		var req JSONRPCRequest
		if err := json.Unmarshal([]byte(line), &req); err != nil {
			log.Printf("Failed to parse request: %v", err)
			continue
		}

		response := s.handleRequest(&req)
		if response != nil {
			if data, err := json.Marshal(response); err == nil {
				fmt.Println(string(data))
			}
		}
	}
}

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

func (s *MCPServer) handleSemanticSearch(params json.RawMessage) (interface{}, error) {
	var req struct {
		Query string `json:"query"`
		Limit int    `json:"limit"`
	}
	if err := json.Unmarshal(params, &req); err != nil {
		return nil, fmt.Errorf("invalid parameters: %v", err)
	}

	if req.Limit == 0 {
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
		id, err := models.EnsureProject(req.Title, "")
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
	if err := models.EnqueueVectorIndex(entityType, entityID); err != nil {
		log.Printf("Warning: failed to queue vector index: %v", err)
	}

	return map[string]interface{}{
		"id":           entityID,
		"type":         entityType,
		"status":       "created",
		"queued_index": true,
	}, nil
}

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

	if req.Limit == 0 {
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

func enrichSearchResult(result vector.QdrantSearchResult) map[string]interface{} {
	payload := result.Payload
	entityType, ok := payload["entity_type"].(string)
	if !ok {
		return nil
	}

	var title, content string
	switch entityType {
	case "task":
		if t, ok := payload["title"].(string); ok {
			title = t
		}
		if c, ok := payload["description"].(string); ok {
			content = c
		}
	case "idea":
		if t, ok := payload["title"].(string); ok {
			title = t
		}
		if c, ok := payload["description"].(string); ok {
			content = c
		}
	case "note", "document":
		if t, ok := payload["title"].(string); ok {
			title = t
		}
		if c, ok := payload["content"].(string); ok {
			content = c
		}
	}

	return map[string]interface{}{
		"id":           result.ID,
		"type":         entityType,
		"title":        title,
		"content":      content[:min(len(content), 200)],
		"similarity":   result.Score,
		"payload":      payload,
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
