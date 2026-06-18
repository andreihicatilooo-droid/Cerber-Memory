package models

import "log"

type Document struct {
	ID        int64
	Title     string
	Content   string
	Version   int
	CreatedAt string
	UpdatedAt string
}

func AddDocument(title, content string) (int64, error) {
	query := `INSERT INTO documents (title, content) VALUES (?, ?)`
	res, err := DB.Exec(query, title, content)
	if err != nil {
		log.Printf("Error adding document: %v", err)
		return 0, err
	}
	id, err := res.LastInsertId()
	if err == nil {
		_ = EnqueueVectorIndex("document", id)
	}
	return id, err
}
