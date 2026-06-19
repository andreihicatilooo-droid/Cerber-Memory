package controllers

import (
	"path/filepath"
	"testing"

	"cerber-memory/internal/mvc/models"
	"cerber-memory/internal/pipeline"
)

// TestRouteItemsHelloWorld exercises the core CERBER pipeline -> MVC routing:
// it takes structured items (as the LLM would produce) and verifies they are
// distributed into the correct SQLite memory categories, then reads them back.
func TestRouteItemsHelloWorld(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "cerber_smoke.db")
	models.InitDB(dbPath)
	defer models.CloseDB()

	items := []pipeline.ParsedItem{
		{Type: "idea", Title: "Switch to PostgreSQL", Description: "Move to PostgreSQL for better scalability", Category: "Architecture", Status: "raw", Priority: "high"},
		{Type: "task", Title: "Update documentation", Description: "Refresh the README before Friday", Priority: "medium", Category: "Docs",
			Subtasks: []pipeline.ParsedSubtask{{Title: "Draft outline", Details: "List sections", Tools: "markdown"}}},
		{Type: "goal", Title: "Finish MVP by Friday"},
		{Type: "note", Title: "Standup", Content: "Discussed migration plan"},
		{Type: "core_memory", Key: "db_engine", Content: "sqlite (dev)", Category: "general"},
	}

	RouteItems(items)

	// Verify task + subtask landed in DB
	tasks, err := models.GetAllTasks()
	if err != nil {
		t.Fatalf("GetAllTasks failed: %v", err)
	}
	if len(tasks) != 1 {
		t.Fatalf("expected 1 task, got %d", len(tasks))
	}
	if tasks[0].Title != "Update documentation" {
		t.Fatalf("unexpected task title: %q", tasks[0].Title)
	}
	if len(tasks[0].Subtasks) != 1 {
		t.Fatalf("expected 1 subtask, got %d", len(tasks[0].Subtasks))
	}

	// Verify core memory upsert + retrieval
	cm, err := models.GetCoreMemory("db_engine")
	if err != nil {
		t.Fatalf("GetCoreMemory failed: %v", err)
	}
	if cm.Content != "sqlite (dev)" {
		t.Fatalf("unexpected core memory content: %q", cm.Content)
	}

	// Verify the mindmap graph builds across stored entities
	graph, err := models.GetMindmapGraph()
	if err != nil {
		t.Fatalf("GetMindmapGraph failed: %v", err)
	}
	if len(graph.Nodes) < 3 {
		t.Fatalf("expected >=3 graph nodes (idea, task, core_memory), got %d", len(graph.Nodes))
	}

	t.Logf("routed items OK: tasks=%d subtasks=%d coreMemory=%q graphNodes=%d",
		len(tasks), len(tasks[0].Subtasks), cm.Content, len(graph.Nodes))
}
