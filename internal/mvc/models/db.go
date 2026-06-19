package models

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	_ "github.com/mattn/go-sqlite3"
)

var DB *sql.DB

func InitDB(dbPath string) {
	// Ensure directory exists
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		log.Fatalf("Failed to create database directory: %v", err)
	}

	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		log.Fatalf("Failed to open database: %v", err)
	}

	if err := db.Ping(); err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	DB = db

	createTables()
}

func createTables() {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS projects (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT UNIQUE NOT NULL,
			parent_id INTEGER,
			description TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY(parent_id) REFERENCES projects(id)
		);`,
		`CREATE TABLE IF NOT EXISTS project_entities (
			project_id INTEGER NOT NULL,
			entity_type TEXT NOT NULL,
			entity_id INTEGER NOT NULL,
			PRIMARY KEY(project_id, entity_type, entity_id),
			FOREIGN KEY(project_id) REFERENCES projects(id)
		);`,
		`CREATE TABLE IF NOT EXISTS notebooks (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT UNIQUE NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);`,
		`CREATE TABLE IF NOT EXISTS notebook_entities (
			notebook_id INTEGER NOT NULL,
			entity_type TEXT NOT NULL,
			entity_id INTEGER NOT NULL,
			PRIMARY KEY(notebook_id, entity_type, entity_id),
			FOREIGN KEY(notebook_id) REFERENCES notebooks(id)
		);`,
		`CREATE TABLE IF NOT EXISTS documents (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			title TEXT NOT NULL,
			content TEXT NOT NULL,
			version INTEGER DEFAULT 1,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);`,
		`CREATE TABLE IF NOT EXISTS files (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			filename TEXT UNIQUE NOT NULL,
			original_name TEXT NOT NULL,
			file_path TEXT NOT NULL,
			mime_type TEXT,
			size INTEGER,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);`,
		`CREATE TABLE IF NOT EXISTS core_memory (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			key TEXT UNIQUE NOT NULL,
			content TEXT NOT NULL,
			category TEXT DEFAULT 'general',
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);`,
		`CREATE TABLE IF NOT EXISTS notes (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			title TEXT,
			content TEXT NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);`,
		`CREATE TABLE IF NOT EXISTS ideas (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			title TEXT,
			description TEXT NOT NULL,
			category TEXT,          -- Architecture, Feature, Optimization, UI/UX
			status TEXT DEFAULT 'raw', -- raw, elaborated, implemented, rejected
			priority TEXT DEFAULT 'medium',
			is_extracted BOOLEAN DEFAULT 0,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);`,
		`CREATE TABLE IF NOT EXISTS goals (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			title TEXT NOT NULL,
			status TEXT DEFAULT 'pending',
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);`,
		`CREATE TABLE IF NOT EXISTS tasks (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			title TEXT NOT NULL,
			description TEXT,
			goal_id INTEGER,
			status TEXT DEFAULT 'pending',
			priority TEXT DEFAULT 'medium',
			cost TEXT,
			risk TEXT,
			expected_outcome TEXT,
			category TEXT,
			is_submodule_task BOOLEAN DEFAULT 0,
			needs_clarification BOOLEAN DEFAULT 0,
			missing_info_details TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY(goal_id) REFERENCES goals(id)
		);`,
		`CREATE TABLE IF NOT EXISTS subtasks (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			task_id INTEGER NOT NULL,
			title TEXT NOT NULL,
			details TEXT,
			tools TEXT,
			status TEXT DEFAULT 'pending',
			needs_resolution BOOLEAN DEFAULT 0,
			FOREIGN KEY(task_id) REFERENCES tasks(id)
		);`,
		`CREATE TABLE IF NOT EXISTS external_resources (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			title TEXT,
			url TEXT UNIQUE NOT NULL,
			description TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);`,
		`CREATE TABLE IF NOT EXISTS universal_links (
			from_type TEXT NOT NULL,
			from_id INTEGER NOT NULL,
			to_type TEXT NOT NULL,
			to_id INTEGER NOT NULL,
			relation_type TEXT DEFAULT 'relates_to',
			PRIMARY KEY(from_type, from_id, to_type, to_id)
		);`,
		`CREATE TABLE IF NOT EXISTS tags (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT UNIQUE NOT NULL
		);`,
		`CREATE TABLE IF NOT EXISTS entity_tags (
			entity_type TEXT NOT NULL,
			entity_id INTEGER NOT NULL,
			tag_id INTEGER NOT NULL,
			PRIMARY KEY(entity_type, entity_id, tag_id),
			FOREIGN KEY(tag_id) REFERENCES tags(id)
		);`,
		`CREATE TABLE IF NOT EXISTS vector_index_queue (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			entity_type TEXT NOT NULL,
			entity_id INTEGER NOT NULL,
			status TEXT DEFAULT 'pending',
			attempts INTEGER DEFAULT 0,
			last_error TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);`,
	}

	for _, query := range queries {
		if _, err := DB.Exec(query); err != nil {
			log.Fatalf("Failed to execute migration: %v\nQuery: %s", err, query)
		}
	}
}

// EnsureProject exists and returns its ID
func EnsureProject(name string, parentName string) (int64, error) {
	return ensureProjectDepth(name, parentName, 0)
}

func ensureProjectDepth(name string, parentName string, depth int) (int64, error) {
	if depth > 10 {
		return 0, fmt.Errorf("project hierarchy too deep or circular at: %s", name)
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return 0, nil
	}
	if parentName != "" && strings.TrimSpace(parentName) == name {
		return 0, fmt.Errorf("project cannot be its own parent: %s", name)
	}

	var parentID sql.NullInt64
	if parentName != "" {
		pid, err := ensureProjectDepth(parentName, "", depth+1)
		if err == nil {
			parentID = sql.NullInt64{Int64: pid, Valid: true}
		}
	}

	var id int64
	err := DB.QueryRow("SELECT id FROM projects WHERE name = ?", name).Scan(&id)
	if err == nil {
		return id, nil
	}

	res, err := DB.Exec("INSERT INTO projects (name, parent_id) VALUES (?, ?)", name, parentID)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func AddToProject(projectID int64, entityType string, entityID int64) error {
	_, err := DB.Exec("INSERT OR IGNORE INTO project_entities (project_id, entity_type, entity_id) VALUES (?, ?, ?)", projectID, entityType, entityID)
	return err
}

func EnsureNotebook(name string) (int64, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return 0, nil
	}

	var id int64
	err := DB.QueryRow("SELECT id FROM notebooks WHERE name = ?", name).Scan(&id)
	if err == nil {
		return id, nil
	}

	res, err := DB.Exec("INSERT INTO notebooks (name) VALUES (?)", name)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func AddToNotebook(notebookID int64, entityType string, entityID int64) error {
	_, err := DB.Exec("INSERT OR IGNORE INTO notebook_entities (notebook_id, entity_type, entity_id) VALUES (?, ?, ?)", notebookID, entityType, entityID)
	return err
}

func AddTagsToEntity(entityType string, entityID int64, tagNames []string) error {
	for _, name := range tagNames {
		name = strings.TrimSpace(strings.ToLower(name))
		if name == "" {
			continue
		}

		var tagID int64
		err := DB.QueryRow("INSERT OR IGNORE INTO tags (name) VALUES (?) RETURNING id", name).Scan(&tagID)
		if err != nil {
			err = DB.QueryRow("SELECT id FROM tags WHERE name = ?", name).Scan(&tagID)
			if err != nil {
				return err
			}
		}

		_, err = DB.Exec("INSERT OR IGNORE INTO entity_tags (entity_type, entity_id, tag_id) VALUES (?, ?, ?)", entityType, entityID, tagID)
		if err != nil {
			return err
		}
	}
	return nil
}

func EnqueueVectorIndex(entityType string, entityID int64) error {
	_, err := DB.Exec("INSERT INTO vector_index_queue (entity_type, entity_id, status) VALUES (?, ?, 'pending')", entityType, entityID)
	return err
}

func CloseDB() {
	if DB != nil {
		DB.Close()
	}
}

func GetEntityText(entityType string, entityID int64) (string, string, error) {
	switch entityType {
	case "core_memory":
		var key, content, category string
		err := DB.QueryRow("SELECT key, content, category FROM core_memory WHERE id = ?", entityID).Scan(&key, &content, &category)
		if err != nil {
			return "", "", err
		}
		if category == "credentials" {
			return "", "", fmt.Errorf("cannot fetch text for credentials category")
		}
		return key, content, nil
	case "task":
		var title, description string
		err := DB.QueryRow("SELECT title, description FROM tasks WHERE id = ?", entityID).Scan(&title, &description)
		return title, description, err
	case "note":
		var title, content string
		err := DB.QueryRow("SELECT title, content FROM notes WHERE id = ?", entityID).Scan(&title, &content)
		return title, content, err
	case "idea":
		var title, description string
		err := DB.QueryRow("SELECT title, description FROM ideas WHERE id = ?", entityID).Scan(&title, &description)
		return title, description, err
	case "document":
		var title, content string
		err := DB.QueryRow("SELECT title, content FROM documents WHERE id = ?", entityID).Scan(&title, &content)
		return title, content, err
	case "project":
		var name, description string
		err := DB.QueryRow("SELECT name, COALESCE(description, '') FROM projects WHERE id = ?", entityID).Scan(&name, &description)
		return name, description, err
	}
	return "", "", fmt.Errorf("unknown entity type: %s", entityType)
}

type GraphNode struct {
	ID    string `json:"id"`    // формат: "entityType_id"
	Type  string `json:"type"`  // e.g. "idea", "task"
	Title string `json:"title"` // человекочитаемое имя
}

type GraphEdge struct {
	Source string `json:"source"`
	Target string `json:"target"`
	Label  string `json:"label"`
}

type MindmapGraph struct {
	Nodes []GraphNode `json:"nodes"`
	Edges []GraphEdge `json:"edges"`
}

func GetMindmapGraph() (MindmapGraph, error) {
	var graph MindmapGraph
	graph.Nodes = make([]GraphNode, 0)
	graph.Edges = make([]GraphEdge, 0)

	// Вспомогательный маппинг для предотвращения дублирования нод
	addedNodes := make(map[string]bool)
	addNode := func(t string, id int64, title string) {
		nodeID := fmt.Sprintf("%s_%d", t, id)
		if !addedNodes[nodeID] {
			if title == "" {
				title = fmt.Sprintf("%s #%d", t, id)
			}
			if len(title) > 40 {
				title = title[:37] + "..."
			}
			graph.Nodes = append(graph.Nodes, GraphNode{ID: nodeID, Type: t, Title: title})
			addedNodes[nodeID] = true
		}
	}

	// 1. Извлекаем ноды из таблиц
	// Идеи
	rows, err := DB.Query("SELECT id, title FROM ideas")
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var id int64
			var title sql.NullString
			if err := rows.Scan(&id, &title); err == nil {
				addNode("idea", id, title.String)
			}
		}
	}

	// Задачи
	rows, err = DB.Query("SELECT id, title FROM tasks")
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var id int64
			var title string
			if err := rows.Scan(&id, &title); err == nil {
				addNode("task", id, title)
			}
		}
	}

	// Блокноты
	rows, err = DB.Query("SELECT id, name FROM notebooks")
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var id int64
			var name string
			if err := rows.Scan(&id, &name); err == nil {
				addNode("notebook", id, name)
			}
		}
	}

	// Проекты
	rows, err = DB.Query("SELECT id, name FROM projects")
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var id int64
			var name string
			if err := rows.Scan(&id, &name); err == nil {
				addNode("project", id, name)
			}
		}
	}

	// Документы
	rows, err = DB.Query("SELECT id, title FROM documents")
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var id int64
			var title string
			if err := rows.Scan(&id, &title); err == nil {
				addNode("document", id, title)
			}
		}
	}

	// Внешние ресурсы
	rows, err = DB.Query("SELECT id, title, url FROM external_resources")
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var id int64
			var title sql.NullString
			var url string
			if err := rows.Scan(&id, &title, &url); err == nil {
				name := title.String
				if name == "" {
					name = url
				}
				addNode("external_resource", id, name)
			}
		}
	}

	// Загруженные файлы
	rows, err = DB.Query("SELECT id, original_name FROM files")
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var id int64
			var originalName string
			if err := rows.Scan(&id, &originalName); err == nil {
				addNode("file", id, originalName)
			}
		}
	}

	// Core Memory
	rows, err = DB.Query("SELECT id, key FROM core_memory")
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var id int64
			var key string
			if err := rows.Scan(&id, &key); err == nil {
				addNode("core_memory", id, key)
			}
		}
	}

	// 2. Извлекаем связи из universal_links
	rows, err = DB.Query("SELECT from_type, from_id, to_type, to_id, relation_type FROM universal_links")
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var fType, tType, rel string
			var fID, tID int64
			if err := rows.Scan(&fType, &fID, &tType, &tID, &rel); err == nil {
				src := fmt.Sprintf("%s_%d", fType, fID)
				dst := fmt.Sprintf("%s_%d", tType, tID)
				// Добавим связь, если оба узла существуют
				if addedNodes[src] && addedNodes[dst] {
					graph.Edges = append(graph.Edges, GraphEdge{Source: src, Target: dst, Label: rel})
				}
			}
		}
	}

	// 3. Дополнительные реляционные связи из связующих таблиц
	// Проекты -> Сущности
	rows, err = DB.Query("SELECT project_id, entity_type, entity_id FROM project_entities")
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var pID, eID int64
			var eType string
			if err := rows.Scan(&pID, &eType, &eID); err == nil {
				src := fmt.Sprintf("project_%d", pID)
				dst := fmt.Sprintf("%s_%d", eType, eID)
				if addedNodes[src] && addedNodes[dst] {
					graph.Edges = append(graph.Edges, GraphEdge{Source: src, Target: dst, Label: "contains"})
				}
			}
		}
	}

	// Блокноты -> Сущности
	rows, err = DB.Query("SELECT notebook_id, entity_type, entity_id FROM notebook_entities")
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var nID, eID int64
			var eType string
			if err := rows.Scan(&nID, &eType, &eID); err == nil {
				src := fmt.Sprintf("notebook_%d", nID)
				dst := fmt.Sprintf("%s_%d", eType, eID)
				if addedNodes[src] && addedNodes[dst] {
					graph.Edges = append(graph.Edges, GraphEdge{Source: src, Target: dst, Label: "notes"})
				}
			}
		}
	}

	return graph, nil
}

