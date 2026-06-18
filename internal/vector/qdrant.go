package vector

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
)

type QdrantPoint struct {
	ID      int64                  `json:"id"`
	Vector  []float32              `json:"vector"`
	Payload map[string]interface{} `json:"payload"`
}

type QdrantUpsertRequest struct {
	Points []QdrantPoint `json:"points"`
}

type QdrantSearchRequest struct {
	Vector      []float32 `json:"vector"`
	Limit       int       `json:"limit"`
	WithPayload bool      `json:"with_payload"`
}

type QdrantSearchResult struct {
	ID      int64                  `json:"id"`
	Score   float32                `json:"score"`
	Payload map[string]interface{} `json:"payload"`
}

type QdrantSearchResponse struct {
	Result []QdrantSearchResult `json:"result"`
}

func UpsertVector(id int64, vector []float32, payload map[string]interface{}) error {
	host := os.Getenv("QDRANT_HOST")
	port := os.Getenv("QDRANT_PORT")
	if host == "" { host = "localhost" }
	if port == "" { port = "6333" }

	collection := "cerber_memory"
	url := fmt.Sprintf("http://%s:%s/collections/%s/points?wait=true", host, port, collection)

	reqBody := QdrantUpsertRequest{
		Points: []QdrantPoint{
			{ID: id, Vector: vector, Payload: payload},
		},
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("PUT", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("qdrant upsert failed with status: %d", resp.StatusCode)
	}

	return nil
}

func SearchSimilarVectors(vector []float32, limit int) ([]QdrantSearchResult, error) {
	host := os.Getenv("QDRANT_HOST")
	port := os.Getenv("QDRANT_PORT")
	if host == "" { host = "localhost" }
	if port == "" { port = "6333" }

	collection := "cerber_memory"
	url := fmt.Sprintf("http://%s:%s/collections/%s/points/search", host, port, collection)

	reqBody := QdrantSearchRequest{
		Vector:      vector,
		Limit:       limit,
		WithPayload: true,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("qdrant search failed with status: %d", resp.StatusCode)
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var searchResp QdrantSearchResponse
	if err := json.Unmarshal(bodyBytes, &searchResp); err != nil {
		return nil, err
	}

	return searchResp.Result, nil
}

func InitCollection() error {
	host := os.Getenv("QDRANT_HOST")
	port := os.Getenv("QDRANT_PORT")
	if host == "" { host = "localhost" }
	if port == "" { port = "6333" }

	collection := "cerber_memory"
	url := fmt.Sprintf("http://%s:%s/collections/%s", host, port, collection)

	// Check if exists
	resp, err := http.Get(url)
	if resp != nil {
		defer resp.Body.Close()
	}
	if err == nil && resp != nil && resp.StatusCode == http.StatusOK {
		return nil
	}

	// Create collection
	createURL := fmt.Sprintf("http://%s:%s/collections/%s", host, port, collection)
	config := map[string]interface{}{
		"vectors": map[string]interface{}{
			"size":     768, // Gemini text-embedding-004 size
			"distance": "Cosine",
		},
	}

	jsonData, err2 := json.Marshal(config)
	if err2 != nil {
		return fmt.Errorf("failed to marshal collection config: %w", err2)
	}
	req, err2 := http.NewRequest("PUT", createURL, bytes.NewBuffer(jsonData))
	if err2 != nil {
		return fmt.Errorf("failed to create collection request: %w", err2)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp2, err2 := client.Do(req)
	if err2 != nil {
		return err2
	}
	defer resp2.Body.Close()

	if resp2.StatusCode < 200 || resp2.StatusCode >= 300 {
		return fmt.Errorf("qdrant collection creation failed with status: %d", resp2.StatusCode)
	}

	return nil
}
