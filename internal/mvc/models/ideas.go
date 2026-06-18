package models

import "log"

type Idea struct {
	ID          int64
	Title       string
	Description string
	Category    string
	Status      string
	Priority    string
	IsExtracted bool
	CreatedAt   string
}

func AddIdea(i Idea) (int64, error) {
	query := `INSERT INTO ideas (title, description, category, status, priority, is_extracted) VALUES (?, ?, ?, ?, ?, ?)`
	res, err := DB.Exec(query, i.Title, i.Description, i.Category, i.Status, i.Priority, i.IsExtracted)
	if err != nil {
		log.Printf("Error adding advanced idea: %v", err)
		return 0, err
	}
	id, err := res.LastInsertId()
	if err == nil {
		_ = EnqueueVectorIndex("idea", id)
	}
	return id, err
}

func UpdateIdeaStatus(id int64, status string) error {
	_, err := DB.Exec("UPDATE ideas SET status = ? WHERE id = ?", status, id)
	return err
}

func GetIdeaByID(id int64) (*Idea, error) {
	row := DB.QueryRow("SELECT id, title, description, category, status, priority, is_extracted, created_at FROM ideas WHERE id = ?", id)
	var i Idea
	err := row.Scan(&i.ID, &i.Title, &i.Description, &i.Category, &i.Status, &i.Priority, &i.IsExtracted, &i.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &i, nil
}

