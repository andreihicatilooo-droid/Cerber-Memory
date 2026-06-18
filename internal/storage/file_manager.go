package storage

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"cloud.google.com/go/storage"
)

// SaveFile copies a file from src to GCS or local storage.
func SaveFile(srcPath string) (string, string, int64, error) {
	file, err := os.Open(srcPath)
	if err != nil {
		return "", "", 0, fmt.Errorf("failed to open source file: %v", err)
	}
	defer file.Close()

	return SaveReader(file, filepath.Base(srcPath))
}

// SaveReader uploads a file from a reader (e.g. HTTP form-data) to GCS if configured,
// otherwise to local storage.
func SaveReader(reader io.Reader, originalName string) (string, string, int64, error) {
	bucketName := os.Getenv("GCS_BUCKET_NAME")
	safeName := filepath.Base(originalName)
	safeName = strings.ReplaceAll(safeName, "..", "")
	if safeName == "" || safeName == "." {
		safeName = "upload"
	}
	newFilename := fmt.Sprintf("%d_%s", time.Now().UnixNano(), safeName)

	if bucketName != "" {
		ctx := context.Background()
		client, err := storage.NewClient(ctx)
		if err != nil {
			return "", "", 0, fmt.Errorf("failed to create GCS client: %v", err)
		}
		defer client.Close()

		bucket := client.Bucket(bucketName)
		obj := bucket.Object(newFilename)
		wc := obj.NewWriter(ctx)

		size, err := io.Copy(wc, reader)
		if err != nil {
			return "", "", 0, fmt.Errorf("failed to write object to GCS: %v", err)
		}
		if err := wc.Close(); err != nil {
			return "", "", 0, fmt.Errorf("failed to close GCS writer: %v", err)
		}

		gcsURI := fmt.Sprintf("gs://%s/%s", bucketName, newFilename)
		return newFilename, gcsURI, size, nil
	}

	// Local fallback
	uploadDir := "data/uploads"
	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		return "", "", 0, fmt.Errorf("failed to create upload directory: %v", err)
	}

	dstPath := filepath.Join(uploadDir, newFilename)
	dstFile, err := os.Create(dstPath)
	if err != nil {
		return "", "", 0, fmt.Errorf("failed to create destination file: %v", err)
	}
	defer dstFile.Close()

	size, err := io.Copy(dstFile, reader)
	if err != nil {
		return "", "", 0, fmt.Errorf("failed to copy file: %v", err)
	}

	return newFilename, dstPath, size, nil
}
