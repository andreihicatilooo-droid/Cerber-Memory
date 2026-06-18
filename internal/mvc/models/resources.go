package models

import "log"

type ExternalResource struct {
	ID          int64
	Title       string
	URL         string
	Description string
	CreatedAt   string
}

func AddExternalResource(title, url, description string) (int64, error) {
	var id int64
	err := DB.QueryRow("SELECT id FROM external_resources WHERE url = ?", url).Scan(&id)
	if err == nil {
		return id, nil
	}

	res, err := DB.Exec("INSERT INTO external_resources (title, url, description) VALUES (?, ?, ?)", title, url, description)
	if err != nil {
		log.Printf("Error adding external resource: %v", err)
		return 0, err
	}
	return res.LastInsertId()
}

func LinkEntities(fromType string, fromID int64, toType string, toID int64, relation string) error {
	_, err := DB.Exec(`INSERT OR IGNORE INTO universal_links (from_type, from_id, to_type, to_id, relation_type) VALUES (?, ?, ?, ?, ?)`,
		fromType, fromID, toType, toID, relation)
	return err
}
