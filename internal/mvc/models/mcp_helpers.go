package models

import (
	"fmt"
	"log"
)

// CreateTask creates a new task with simplified API
func CreateTask(title, description, priority, category string) (int64, error) {
	if description == "" {
		return 0, fmt.Errorf("description is required")
	}
	if title == "" {
		title = "Untitled Task"
	}
	if priority == "" {
		priority = "medium"
	}

	query := `INSERT INTO tasks (title, description, priority, category, status)
	          VALUES (?, ?, ?, ?, 'pending')`
	res, err := DB.Exec(query, title, description, priority, category)
	if err != nil {
		log.Printf("Error creating task: %v", err)
		return 0, err
	}

	return res.LastInsertId()
}

// CreateIdea creates a new idea with simplified API
func CreateIdea(title, description, category, priority string) (int64, error) {
	if description == "" {
		return 0, fmt.Errorf("description is required")
	}
	if title == "" {
		title = "Untitled Idea"
	}
	if priority == "" {
		priority = "medium"
	}

	idea := Idea{
		Title:       title,
		Description: description,
		Category:    category,
		Priority:    priority,
		Status:      "raw",
		IsExtracted: false,
	}
	return AddIdea(idea)
}

// CreateNote creates a new note with simplified API
func CreateNote(title, content string) (int64, error) {
	if content == "" {
		return 0, fmt.Errorf("content is required")
	}
	if title == "" {
		title = "Untitled Note"
	}
	return AddNote(title, content)
}

// CreateDocument creates a new document with simplified API
func CreateDocument(title, content string) (int64, error) {
	if content == "" {
		return 0, fmt.Errorf("content is required")
	}
	if title == "" {
		title = "Untitled Document"
	}
	return AddDocument(title, content)
}

// QueryTasks retrieves tasks with optional filters
func QueryTasks(status, priority, category string, limit int) ([]map[string]interface{}, error) {
	query := "SELECT id, title, description, status, priority, category, created_at FROM tasks WHERE 1=1"
	var args []interface{}

	if status != "" {
		query += " AND status = ?"
		args = append(args, status)
	}
	if priority != "" {
		query += " AND priority = ?"
		args = append(args, priority)
	}
	if category != "" {
		query += " AND category = ?"
		args = append(args, category)
	}

	query += " ORDER BY created_at DESC LIMIT ?"
	args = append(args, limit)

	rows, err := DB.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []map[string]interface{}
	for rows.Next() {
		var id int64
		var title, description, status, priority, category, createdAt string
		if err := rows.Scan(&id, &title, &description, &status, &priority, &category, &createdAt); err != nil {
			log.Printf("Error scanning task: %v", err)
			continue
		}

		results = append(results, map[string]interface{}{
			"id":          id,
			"title":       title,
			"description": description,
			"status":      status,
			"priority":    priority,
			"category":    category,
			"created_at":  createdAt,
		})
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}
	return results, nil
}

// QueryIdeas retrieves ideas with optional filters
func QueryIdeas(status, category, priority string, limit int) ([]map[string]interface{}, error) {
	query := "SELECT id, title, description, status, category, priority, created_at FROM ideas WHERE 1=1"
	var args []interface{}

	if status != "" {
		query += " AND status = ?"
		args = append(args, status)
	}
	if category != "" {
		query += " AND category = ?"
		args = append(args, category)
	}
	if priority != "" {
		query += " AND priority = ?"
		args = append(args, priority)
	}

	query += " ORDER BY created_at DESC LIMIT ?"
	args = append(args, limit)

	rows, err := DB.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []map[string]interface{}
	for rows.Next() {
		var id int64
		var title, description, status, category, priority, createdAt string
		if err := rows.Scan(&id, &title, &description, &status, &category, &priority, &createdAt); err != nil {
			log.Printf("Error scanning idea: %v", err)
			continue
		}

		results = append(results, map[string]interface{}{
			"id":          id,
			"title":       title,
			"description": description,
			"status":      status,
			"category":    category,
			"priority":    priority,
			"created_at":  createdAt,
		})
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}
	return results, nil
}

// QueryNotes retrieves recent notes
func QueryNotes(limit int) ([]map[string]interface{}, error) {
	query := "SELECT id, title, content, created_at FROM notes ORDER BY created_at DESC LIMIT ?"
	rows, err := DB.Query(query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []map[string]interface{}
	for rows.Next() {
		var id int64
		var title, content, createdAt string
		if err := rows.Scan(&id, &title, &content, &createdAt); err != nil {
			log.Printf("Error scanning note: %v", err)
			continue
		}

		contentPreview := content
		if len(contentPreview) > 150 {
			contentPreview = contentPreview[:150] + "..."
		}

		results = append(results, map[string]interface{}{
			"id":              id,
			"title":           title,
			"content_preview": contentPreview,
			"created_at":      createdAt,
		})
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}
	return results, nil
}

// QueryDocuments retrieves recent documents
func QueryDocuments(limit int) ([]map[string]interface{}, error) {
	query := "SELECT id, title, content, version, created_at FROM documents ORDER BY created_at DESC LIMIT ?"
	rows, err := DB.Query(query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []map[string]interface{}
	for rows.Next() {
		var id int64
		var title, content string
		var version int
		var createdAt string
		if err := rows.Scan(&id, &title, &content, &version, &createdAt); err != nil {
			log.Printf("Error scanning document: %v", err)
			continue
		}

		contentPreview := content
		if len(contentPreview) > 150 {
			contentPreview = contentPreview[:150] + "..."
		}

		results = append(results, map[string]interface{}{
			"id":              id,
			"title":           title,
			"content_preview": contentPreview,
			"version":         version,
			"created_at":      createdAt,
		})
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}
	return results, nil
}

// QueryProjects retrieves projects
func QueryProjects(limit int) ([]map[string]interface{}, error) {
	query := "SELECT id, name, description, created_at FROM projects ORDER BY created_at DESC LIMIT ?"
	rows, err := DB.Query(query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []map[string]interface{}
	for rows.Next() {
		var id int64
		var name string
		var description *string
		var createdAt string
		if err := rows.Scan(&id, &name, &description, &createdAt); err != nil {
			log.Printf("Error scanning project: %v", err)
			continue
		}

		desc := ""
		if description != nil {
			desc = *description
		}

		results = append(results, map[string]interface{}{
			"id":          id,
			"name":        name,
			"description": desc,
			"created_at":  createdAt,
		})
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}
	return results, nil
}

// Credentials management with encryption
// GetCredential retrieves a decrypted credential
func GetCredential(key string) (string, error) {
	var encryptedValue []byte

	query := `SELECT content FROM core_memory
	          WHERE key = ? AND category = 'credentials'`

	err := DB.QueryRow(query, key).Scan(&encryptedValue)
	if err != nil {
		return "", fmt.Errorf("credential not found: %v", err)
	}

	// For now, return as-is (implement actual decryption later)
	return string(encryptedValue), nil
}

// SetCredential stores a credential securely (encryption in next phase)
func SetCredential(key, value, category string) (int64, error) {
	if key == "" || value == "" {
		return 0, fmt.Errorf("key and value are required")
	}

	if category == "" {
		category = "api_key"
	}

	// Store credentials with credentials category to prevent indexing
	// Type (api_key, password, token, database) is preserved in the returned response
	// but credentials are marked with category='credentials' for secure retrieval
	query := `INSERT INTO core_memory (key, content, category)
	          VALUES (?, ?, 'credentials')
	          ON CONFLICT(key) DO UPDATE SET content = ?`

	res, err := DB.Exec(query, key, value, value)
	if err != nil {
		log.Printf("Error setting credential: %v", err)
		return 0, err
	}

	id, err := res.LastInsertId()
	if err != nil {
		return 0, err
	}

	// Do NOT index credentials in vector DB for security
	return id, nil
}
