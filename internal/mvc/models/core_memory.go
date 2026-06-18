package models

import (
	"cerber-memory/internal/crypto"
	"fmt"
	"log"
)

type CoreMemory struct {
	ID        int64
	Key       string
	Content   string
	Category  string
	CreatedAt string
	UpdatedAt string
}

func UpsertCoreMemory(key, content, category string) (int64, error) {
	// Encrypt sensitive content
	processedContent := content
	if category == "credentials" {
		enc, err := crypto.Encrypt(content)
		if err != nil {
			return 0, fmt.Errorf("failed to encrypt credentials: %w", err)
		}
		processedContent = enc
	}

	query := `
	INSERT INTO core_memory (key, content, category, updated_at) 
	VALUES (?, ?, ?, CURRENT_TIMESTAMP)
	ON CONFLICT(key) DO UPDATE SET 
		content=excluded.content, 
		category=excluded.category,
		updated_at=CURRENT_TIMESTAMP;
	`
	_, err := DB.Exec(query, key, processedContent, category)
	if err != nil {
		log.Printf("Error upserting core memory: %v", err)
		return 0, err
	}
	
	var id int64
	err = DB.QueryRow("SELECT id FROM core_memory WHERE key = ?", key).Scan(&id)
	if err == nil && category != "credentials" {
		_ = EnqueueVectorIndex("core_memory", id)
	}
	return id, err
}

func GetCoreMemory(key string) (*CoreMemory, error) {
	row := DB.QueryRow("SELECT id, key, content, category, created_at, updated_at FROM core_memory WHERE key = ?", key)
	var cm CoreMemory
	err := row.Scan(&cm.ID, &cm.Key, &cm.Content, &cm.Category, &cm.CreatedAt, &cm.UpdatedAt)
	if err != nil {
		return nil, err
	}

	// Decrypt if it's credentials
	if cm.Category == "credentials" {
		dec, err := crypto.Decrypt(cm.Content)
		if err == nil {
			cm.Content = dec
		}
	}

	return &cm, nil
}
