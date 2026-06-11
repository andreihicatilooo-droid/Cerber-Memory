package models

import (
	"log"
)

type Note struct {
	ID        int64
	Title     string
	Content   string
	Tags      string
	CreatedAt string
}

func AddNote(title, content string) (int64, error) {
	query := `INSERT INTO notes (title, content) VALUES (?, ?)`
	res, err := DB.Exec(query, title, content)
	if err != nil {
		log.Printf("Error adding note: %v", err)
		return 0, err
	}
	id, err := res.LastInsertId()
	if err == nil {
		_ = EnqueueVectorIndex("note", id)
	}
	return id, err
}
