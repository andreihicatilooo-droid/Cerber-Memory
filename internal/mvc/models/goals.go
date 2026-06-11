package models

import (
	"log"
)

type Goal struct {
	ID        int64
	Title     string
	Status    string
	CreatedAt string
}

func AddGoal(title string) (int64, error) {
	query := `INSERT INTO goals (title) VALUES (?)`
	res, err := DB.Exec(query, title)
	if err != nil {
		log.Printf("Error adding goal: %v", err)
		return 0, err
	}
	return res.LastInsertId()
}

func UpdateGoalStatus(id int64, status string) error {
	query := `UPDATE goals SET status = ? WHERE id = ?`
	_, err := DB.Exec(query, status, id)
	return err
}
