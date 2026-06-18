package models

import (
	"fmt"
	"log"
)

type Subtask struct {
	ID              int64
	TaskID          int64
	Title           string
	Details         string
	Tools           string
	Status          string
	NeedsResolution bool
}

type Task struct {
	ID              int64
	Title           string
	Description     string
	GoalID          *int64
	Status          string
	Priority        string
	Cost            string
	Risk            string
	ExpectedOutcome string
	Category        string
	IsSubmodule     bool
	NeedsClarify    bool
	MissingInfo     string
	Subtasks        []Subtask
	CreatedAt       string
}

func AddTask(t Task) (int64, error) {
	tx, err := DB.Begin()
	if err != nil {
		return 0, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	query := `INSERT INTO tasks (
		title, description, goal_id, status, priority, cost, risk, expected_outcome, category, is_submodule_task, needs_clarification, missing_info_details
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	res, err := tx.Exec(query,
		t.Title, t.Description, t.GoalID, "pending", t.Priority,
		t.Cost, t.Risk, t.ExpectedOutcome, t.Category, t.IsSubmodule, t.NeedsClarify, t.MissingInfo,
	)
	if err != nil {
		log.Printf("Error adding smart task: %v", err)
		return 0, err
	}

	taskID, _ := res.LastInsertId()

	for _, st := range t.Subtasks {
		_, err := tx.Exec(`INSERT INTO subtasks (task_id, title, details, tools, needs_resolution) VALUES (?, ?, ?, ?, ?)`,
			taskID, st.Title, st.Details, st.Tools, st.NeedsResolution)
		if err != nil {
			return 0, fmt.Errorf("error adding subtask %q: %w", st.Title, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("failed to commit task transaction: %w", err)
	}

	return taskID, nil
}

func GetAllTasks() ([]Task, error) {
	rows, err := DB.Query(`SELECT id, title, description, goal_id, status, priority, cost, risk, expected_outcome, category, is_submodule_task, needs_clarification, missing_info_details, created_at FROM tasks`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tasks []Task
	for rows.Next() {
		var t Task
		err := rows.Scan(
			&t.ID, &t.Title, &t.Description, &t.GoalID, &t.Status, &t.Priority,
			&t.Cost, &t.Risk, &t.ExpectedOutcome, &t.Category, &t.IsSubmodule,
			&t.NeedsClarify, &t.MissingInfo, &t.CreatedAt,
		)
		if err != nil {
			return nil, err
		}

		// Fetch subtasks
		subRows, err := DB.Query(`SELECT id, task_id, title, details, tools, status, needs_resolution FROM subtasks WHERE task_id = ?`, t.ID)
		if err == nil {
			for subRows.Next() {
				var st Subtask
				if err := subRows.Scan(&st.ID, &st.TaskID, &st.Title, &st.Details, &st.Tools, &st.Status, &st.NeedsResolution); err == nil {
					t.Subtasks = append(t.Subtasks, st)
				}
			}
			subRows.Close()
		}

		tasks = append(tasks, t)
	}
	return tasks, nil
}

func GetTaskByID(id int64) (*Task, error) {
	row := DB.QueryRow(`SELECT id, title, description, goal_id, status, priority, cost, risk, expected_outcome, category, is_submodule_task, needs_clarification, missing_info_details, created_at FROM tasks WHERE id = ?`, id)
	var t Task
	err := row.Scan(
		&t.ID, &t.Title, &t.Description, &t.GoalID, &t.Status, &t.Priority,
		&t.Cost, &t.Risk, &t.ExpectedOutcome, &t.Category, &t.IsSubmodule,
		&t.NeedsClarify, &t.MissingInfo, &t.CreatedAt,
	)
	if err != nil {
		return nil, err
	}

	// Fetch subtasks
	subRows, err := DB.Query(`SELECT id, task_id, title, details, tools, status, needs_resolution FROM subtasks WHERE task_id = ?`, t.ID)
	if err == nil {
		for subRows.Next() {
			var st Subtask
			if err := subRows.Scan(&st.ID, &st.TaskID, &st.Title, &st.Details, &st.Tools, &st.Status, &st.NeedsResolution); err == nil {
				t.Subtasks = append(t.Subtasks, st)
			}
		}
		subRows.Close()
	}

	return &t, nil
}
