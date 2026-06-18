package vector

import (
	"context"
	"log"
	"time"

	"cerber-memory/internal/mvc/models"
	"cerber-memory/internal/pipeline"
)

// StartBackgroundIndexer runs in a background goroutine and processes new memories
func StartBackgroundIndexer(ctx context.Context) {
	log.Println("[Daemon] Starting background vector indexer...")

	// Recover items stuck in 'processing' state from a previous crash
	if _, recErr := models.DB.Exec("UPDATE vector_index_queue SET status = 'pending' WHERE status = 'processing'"); recErr != nil {
		log.Printf("[Daemon] Warning: failed to recover stale queue items: %v", recErr)
	}

	// Initialize collection first if needed
	err := InitCollection()
	if err != nil {
		log.Printf("[Daemon] Warning: failed to initialize Qdrant collection: %v", err)
	}

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Println("[Daemon] Stopping background vector indexer...")
			return
		case <-ticker.C:
			processQueueBatch()
		}
	}
}

func processQueueBatch() {
	// Query pending items from queue
	rows, err := models.DB.Query(`
		SELECT id, entity_type, entity_id, attempts 
		FROM vector_index_queue 
		WHERE status IN ('pending', 'failed') AND attempts < 3 
		LIMIT 10
	`)
	if err != nil {
		log.Printf("[Daemon] Error querying queue: %v", err)
		return
	}
	defer rows.Close()

	type queueItem struct {
		id         int64
		entityType string
		entityID   int64
		attempts   int
	}

	var items []queueItem
	for rows.Next() {
		var item queueItem
		if err := rows.Scan(&item.id, &item.entityType, &item.entityID, &item.attempts); err == nil {
			items = append(items, item)
		}
	}

	if len(items) == 0 {
		return
	}

	for _, item := range items {
		log.Printf("[Daemon] Indexing entity %s (ID: %d) - attempt %d", item.entityType, item.entityID, item.attempts+1)
		
		// Update status to processing
		_, _ = models.DB.Exec("UPDATE vector_index_queue SET status = 'processing', attempts = attempts + 1, updated_at = CURRENT_TIMESTAMP WHERE id = ?", item.id)

		// 1. Get entity text
		title, text, err := models.GetEntityText(item.entityType, item.entityID)
		if err != nil {
			log.Printf("[Daemon] Failed to get text for %s (ID: %d): %v", item.entityType, item.entityID, err)
			_, _ = models.DB.Exec("UPDATE vector_index_queue SET status = 'failed', last_error = ? WHERE id = ?", err.Error(), item.id)
			continue
		}

		// 2. Generate embedding
		embInput := title + "\n" + text
		embedding, err := pipeline.GenerateEmbedding(embInput)
		if err != nil {
			log.Printf("[Daemon] Failed to generate embedding: %v", err)
			_, _ = models.DB.Exec("UPDATE vector_index_queue SET status = 'failed', last_error = ? WHERE id = ?", err.Error(), item.id)
			continue
		}

		// 3. Upsert to Qdrant
		payload := map[string]interface{}{
			"entity_type": item.entityType,
			"entity_id":   item.entityID,
			"title":       title,
			"text":        text,
		}
		
		pointID := generatePointID(item.entityType, item.entityID)

		err = UpsertVector(pointID, embedding, payload)
		if err != nil {
			log.Printf("[Daemon] Failed to upsert to Qdrant: %v", err)
			_, _ = models.DB.Exec("UPDATE vector_index_queue SET status = 'failed', last_error = ? WHERE id = ?", err.Error(), item.id)
			continue
		}

		// 4. Update status to done
		_, _ = models.DB.Exec("UPDATE vector_index_queue SET status = 'done', updated_at = CURRENT_TIMESTAMP WHERE id = ?", item.id)
		log.Printf("[Daemon] Successfully indexed %s (ID: %d) into Qdrant", item.entityType, item.entityID)
	}
}

// Generate a unique int64 point ID from entityType and entityID
func generatePointID(entityType string, entityID int64) int64 {
	var hash int64 = 17
	for _, char := range entityType {
		hash = hash*37 + int64(char)
	}
	return (hash & 0xFFFFFFFF) << 32 | (entityID & 0xFFFFFFFF)
}
