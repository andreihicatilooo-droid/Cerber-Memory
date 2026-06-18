package pipeline

import (
	"context"
	"fmt"
	"os"

	"github.com/google/generative-ai-go/genai"
	"github.com/spf13/viper"
)

// GenerateEmbedding generates embeddings using official Google GenAI Go SDK.
func GenerateEmbedding(text string) ([]float32, error) {
	ctx := context.Background()
	client, err := getGenAIClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create GenAI client: %v", err)
	}
	defer client.Close()

	modelName := viper.GetString("GEMINI_EMBEDDING_MODEL")
	if modelName == "" {
		modelName = os.Getenv("GEMINI_EMBEDDING_MODEL")
	}
	if modelName == "" {
		modelName = "text-embedding-004"
	}

	model := client.EmbeddingModel(modelName)
	resp, err := model.EmbedContent(ctx, genai.Text(text))
	if err != nil {
		return nil, fmt.Errorf("failed to generate embedding: %v", err)
	}

	if resp.Embedding == nil {
		return nil, fmt.Errorf("no embedding returned")
	}

	return resp.Embedding.Values, nil
}
