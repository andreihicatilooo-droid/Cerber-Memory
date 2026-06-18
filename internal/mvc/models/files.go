package models

import (
	"log"
)

type FileMetadata struct {
	ID           int64
	Filename     string
	OriginalName string
	FilePath     string
	MimeType     string
	Size         int64
	CreatedAt    string
}

func AddFileMetadata(f FileMetadata) (int64, error) {
	query := `INSERT INTO files (filename, original_name, file_path, mime_type, size) VALUES (?, ?, ?, ?, ?)`
	res, err := DB.Exec(query, f.Filename, f.OriginalName, f.FilePath, f.MimeType, f.Size)
	if err != nil {
		log.Printf("Error adding file metadata: %v", err)
		return 0, err
	}
	return res.LastInsertId()
}
