package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"spectra/internal/config"
	"spectra/internal/git"
)

type CommitSummaryGenerator interface {
	GenerateCommitSummaryText(ctx context.Context, commitSummary git.CommitSummary) (string, error)
}

type OpenAICompatibleGenerator struct {
	baseURL    string
	model      string
	apiKey     string
	httpClient *http.Client
}

func NewGeneratorFromConfig(applicationConfig config.Config) (CommitSummaryGenerator, error) {
	baseURL := applicationConfig.LocalBaseURL
	apiKey := ""

	if applicationConfig.Mode == "api" {
		baseURL = applicationConfig.APIBaseURL
		apiKey = strings.TrimSpace(os.Getenv(applicationConfig.APIKeyEnv))
		if apiKey == "" {
			return nil, fmt.Errorf("environment variable %s is not set", applicationConfig.APIKeyEnv)
		}
	}

	return &OpenAICompatibleGenerator{
		baseURL: strings.TrimRight(baseURL, "/"),
		model:   applicationConfig.Model,
		apiKey:  apiKey,
		httpClient: &http.Client{
			Timeout: time.Duration(applicationConfig.RequestTimeoutSeconds) * time.Second,
		},
	}, nil
}

func (generator *OpenAICompatibleGenerator) GenerateCommitSummaryText(ctx context.Context, commitSummary git.CommitSummary) (string, error) {
	requestPayload := chatCompletionsRequest{
		Model: generator.model,
		Messages: []chatMessage{
			{
				Role:    "system",
				Content: "You write concise changelog summaries for software commits. Return exactly one sentence with plain English and no markdown.",
			},
			{
				Role:    "user",
				Content: buildCommitPrompt(commitSummary),
			},
		},
		Temperature: 0.2,
	}

	serializedPayload, err := json.Marshal(requestPayload)
	if err != nil {
		return "", err
	}

	httpRequest, err := http.NewRequestWithContext(ctx, http.MethodPost, generator.baseURL+"/chat/completions", bytes.NewReader(serializedPayload))
	if err != nil {
		return "", err
	}

	httpRequest.Header.Set("Content-Type", "application/json")
	if generator.apiKey != "" {
		httpRequest.Header.Set("Authorization", "Bearer "+generator.apiKey)
	}

	httpResponse, err := generator.httpClient.Do(httpRequest)
	if err != nil {
		return "", err
	}
	defer httpResponse.Body.Close()

	if httpResponse.StatusCode < 200 || httpResponse.StatusCode >= 300 {
		return "", fmt.Errorf("llm request failed with status %s", httpResponse.Status)
	}

	var responsePayload chatCompletionsResponse
	if err := json.NewDecoder(httpResponse.Body).Decode(&responsePayload); err != nil {
		return "", err
	}

	if len(responsePayload.Choices) == 0 {
		return "", fmt.Errorf("llm response did not contain choices")
	}

	generatedText := strings.TrimSpace(responsePayload.Choices[0].Message.Content)
	if generatedText == "" {
		return "", fmt.Errorf("llm generated empty summary text")
	}

	return sanitizeGeneratedSummary(generatedText), nil
}

func buildCommitPrompt(commitSummary git.CommitSummary) string {
	joinedFiles := "none"
	if len(commitSummary.ChangedFiles) > 0 {
		joinedFiles = strings.Join(commitSummary.ChangedFiles, ", ")
	}

	return fmt.Sprintf("Commit subject: %s\nAuthor: %s\nFiles changed: %d\nInsertions: %d\nDeletions: %d\nChanged files: %s\nWrite one concise changelog sentence describing what changed and why it matters to users.",
		commitSummary.Subject,
		commitSummary.Author,
		commitSummary.FilesChanged,
		commitSummary.Insertions,
		commitSummary.Deletions,
		joinedFiles,
	)
}

func sanitizeGeneratedSummary(generatedText string) string {
	cleaned := strings.ReplaceAll(generatedText, "\n", " ")
	cleaned = strings.TrimSpace(cleaned)
	cleaned = strings.Trim(cleaned, "`\" ")
	return cleaned
}

type chatCompletionsRequest struct {
	Model       string        `json:"model"`
	Messages    []chatMessage `json:"messages"`
	Temperature float64       `json:"temperature,omitempty"`
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatCompletionsResponse struct {
	Choices []chatChoice `json:"choices"`
}

type chatChoice struct {
	Message chatMessage `json:"message"`
}
