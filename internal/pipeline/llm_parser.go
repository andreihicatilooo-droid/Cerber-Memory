package pipeline

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/google/generative-ai-go/genai"
	"github.com/spf13/viper"
	"google.golang.org/api/option"
)

// ParsedSubtask represents a subtask extracted from the main task
type ParsedSubtask struct {
	Title           string `json:"title"`
	Details         string `json:"details,omitempty"`
	Tools           string `json:"tools,omitempty"`
	NeedsResolution bool   `json:"needs_resolution"`
}

// ParsedLink represents a bidirectional link between the current item and another entity or URL
type ParsedLink struct {
	TargetType string `json:"target_type"` // 'url', 'idea', 'task', 'project', etc.
	TargetID   string `json:"target_id,omitempty"` // key or title
	URL        string `json:"url,omitempty"`
	Relation   string `json:"relation"` // e.g., 'source', 'reference', 'inspiration', 'elaborates'
}

// ParsedItem represents a structured piece of memory
type ParsedItem struct {
	Type        string   `json:"type"` // core_memory, note, idea, goal, task, document, external_resource
	Key         string   `json:"key,omitempty"`
	Content     string   `json:"content,omitempty"`
	Category    string   `json:"category,omitempty"` // For ideas: Architecture, UI/UX, feature, etc.
	Status      string   `json:"status,omitempty"`   // For ideas: raw, elaborated
	Title       string   `json:"title,omitempty"`
	Description string   `json:"description,omitempty"`
	URL         string   `json:"url,omitempty"`
	Tags        []string `json:"tags,omitempty"`
	Project     string   `json:"project,omitempty"`
	Workspace   string   `json:"workspace,omitempty"`
	Notebook    string   `json:"notebook,omitempty"`

	Priority           string          `json:"priority,omitempty"`
	Cost               string          `json:"cost,omitempty"`
	Risk               string          `json:"risk,omitempty"`
	ExpectedOutcome    string          `json:"expected_outcome,omitempty"`
	IsSubmodule        bool            `json:"is_submodule,omitempty"`
	NeedsClarification bool            `json:"needs_clarification"`
	MissingInfoDetails string          `json:"missing_info_details,omitempty"`
	Subtasks           []ParsedSubtask `json:"subtasks,omitempty"`
	Links              []ParsedLink    `json:"links,omitempty"`
}

func getGenAIClient(ctx context.Context) (*genai.Client, error) {
	apiKey := viper.GetString("GEMINI_API_KEY")
	if apiKey == "" {
		apiKey = os.Getenv("GEMINI_API_KEY")
	}

	if apiKey != "" {
		return genai.NewClient(ctx, option.WithAPIKey(apiKey))
	}

	// Fallback to Application Default Credentials (ADC) for Google Cloud
	return genai.NewClient(ctx)
}

func ParseTextWithLLM(text string) ([]ParsedItem, error) {
	ctx := context.Background()
	client, err := getGenAIClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create GenAI client: %v", err)
	}
	defer client.Close()

	modelName := viper.GetString("GEMINI_PARSER_MODEL")
	if modelName == "" {
		modelName = os.Getenv("GEMINI_PARSER_MODEL")
	}
	if modelName == "" {
		modelName = "gemini-2.5-flash"
	}

	model := client.GenerativeModel(modelName)
	model.ResponseMIMEType = "application/json"

	prompt := `You are an expert Strategic Architect and Knowledge Agent. Analyze the text and extract structured info.

SPECIAL MANDATE: IDEAS
- Extract "idea" as a distinct, first-class entity.
- Categorize ideas into: "Architecture", "Feature", "Optimization", "UI/UX", "Security".
- Set "status": "raw" (new thoughts) or "elaborated" (if you are asked to develop an idea).
- If the user asks to "develop" or "elaborate" an idea, create the IDEA object AND multiple TASK objects that represent the development plan. Link tasks to the idea using the 'links' field with relation 'elaborates'.

MANDATE:
- Extract 'external_resource' (URLs).
- CREATE LINKS: Bidirectional connections using the 'links' field.
- Smart Task Analysis (Decomposition, Tools, Priority, Gap Analysis).

TYPES: 
   - "idea", "external_resource", "task", "document", "goal", "core_memory", "note".

LINKING:
- To link to a previous idea or project by name, use: {"target_type": "idea", "target_id": "Idea Name", "relation": "..."}.

Return STRICT JSON array.

Text to analyze:
` + text

	resp, err := model.GenerateContent(ctx, genai.Text(prompt))
	if err != nil {
		return nil, fmt.Errorf("generate content error: %v", err)
	}

	if len(resp.Candidates) == 0 || len(resp.Candidates[0].Content.Parts) == 0 {
		return nil, fmt.Errorf("no content returned from LLM")
	}

	var rawJSON string
	for _, part := range resp.Candidates[0].Content.Parts {
		if textPart, ok := part.(genai.Text); ok {
			rawJSON += string(textPart)
		}
	}

	rawJSON = strings.TrimSpace(rawJSON)
	rawJSON = strings.TrimPrefix(rawJSON, "```json")
	rawJSON = strings.TrimPrefix(rawJSON, "```")
	rawJSON = strings.TrimSuffix(rawJSON, "```")
	rawJSON = strings.TrimSpace(rawJSON)

	var items []ParsedItem
	if err := json.Unmarshal([]byte(rawJSON), &items); err != nil {
		return nil, fmt.Errorf("failed to parse JSON from LLM: %v\nRaw: %s", err, rawJSON)
	}

	return items, nil
}

func ElaborateIdeaWithLLM(title, description string) ([]ParsedItem, error) {
	ctx := context.Background()
	client, err := getGenAIClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create GenAI client: %v", err)
	}
	defer client.Close()

	modelName := viper.GetString("GEMINI_ELABORATOR_MODEL")
	if modelName == "" {
		modelName = os.Getenv("GEMINI_ELABORATOR_MODEL")
	}
	if modelName == "" {
		modelName = "gemini-2.5-pro"
	}

	model := client.GenerativeModel(modelName)
	model.ResponseMIMEType = "application/json"

	prompt := fmt.Sprintf(`You are a Senior Strategic Architect and Project Manager. 
Your task is to take a "raw idea" and develop it into a comprehensive, thought-out project plan.

INPUT IDEA:
Title: %s
Description: %s

MANDATE:
1. Create a "project" entity with the name of the idea.
2. Generate a Roadmap consisting of 3-5 "goal" entities (e.g., Phases: Planning, MVP, Scaling).
3. For EACH Goal, generate a detailed list of 3-5 "task" entities.
4. For EACH Task, provide:
   - Deep decomposition into 3+ "subtasks" with specific tools and implementation details.
   - Analysis of "priority", "cost", "risk", and "expected_outcome".
5. Link everything together using the "links" field. All tasks should "elaborate" the main idea.

OUTPUT:
Return STRICTLY a JSON array of ParsedItem objects. No markdown.

Type of items to return: "project", "goal", "task".
`, title, description)

	resp, err := model.GenerateContent(ctx, genai.Text(prompt))
	if err != nil {
		return nil, fmt.Errorf("generate content error: %v", err)
	}

	if len(resp.Candidates) == 0 || len(resp.Candidates[0].Content.Parts) == 0 {
		return nil, fmt.Errorf("no content returned from LLM")
	}

	var rawJSON string
	for _, part := range resp.Candidates[0].Content.Parts {
		if textPart, ok := part.(genai.Text); ok {
			rawJSON += string(textPart)
		}
	}

	rawJSON = strings.TrimSpace(rawJSON)
	rawJSON = strings.TrimPrefix(rawJSON, "```json")
	rawJSON = strings.TrimPrefix(rawJSON, "```")
	rawJSON = strings.TrimSuffix(rawJSON, "```")
	rawJSON = strings.TrimSpace(rawJSON)

	var items []ParsedItem
	if err := json.Unmarshal([]byte(rawJSON), &items); err != nil {
		return nil, fmt.Errorf("failed to parse JSON from LLM: %v\nRaw: %s", err, rawJSON)
	}

	return items, nil
}
