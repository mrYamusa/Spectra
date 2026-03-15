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

// CommitSummaryGenerator generates a one-sentence plain-English summary of a commit.
// Used by the `track` command to enrich changelog entries.
type CommitSummaryGenerator interface {
	GenerateCommitSummaryText(ctx context.Context, commitSummary git.CommitSummary) (string, error)
}

// ReadmeSectionGenerator produces a markdown "Recent Changes" section for a README.
// Used by the `readme` command to update the spectra-managed block.
type ReadmeSectionGenerator interface {
	GenerateReadmeSectionUpdate(ctx context.Context, commitSummary git.CommitSummary, significance string) (string, error)
}

// OpenAICompatibleGenerator talks to any OpenAI-compatible /chat/completions endpoint.
// This covers both local models (e.g. Ollama) and cloud APIs (e.g. OpenAI, Anthropic).
type OpenAICompatibleGenerator struct {
	baseURL    string
	model      string
	apiKey     string
	httpClient *http.Client
}

// newOpenAIGenerator is the shared internal constructor used by both public constructors below.
func newOpenAIGenerator(applicationConfig config.Config) (*OpenAICompatibleGenerator, error) {
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

// NewGeneratorFromConfig returns a CommitSummaryGenerator for use by the `track` command.
func NewGeneratorFromConfig(applicationConfig config.Config) (CommitSummaryGenerator, error) {
	return newOpenAIGenerator(applicationConfig)
}

// NewReadmeGeneratorFromConfig returns a ReadmeSectionGenerator for use by the `readme` command.
func NewReadmeGeneratorFromConfig(applicationConfig config.Config) (ReadmeSectionGenerator, error) {
	return newOpenAIGenerator(applicationConfig)
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

// GenerateReadmeSectionUpdate asks the LLM to write a "## Recent Changes" markdown
// section based on the commit and its significance level.
// The returned string is placed between the spectra managed markers in README.md.
func (generator *OpenAICompatibleGenerator) GenerateReadmeSectionUpdate(ctx context.Context, commitSummary git.CommitSummary, significance string) (string, error) {
	requestPayload := chatCompletionsRequest{
		Model: generator.model,
		Messages: []chatMessage{
			{
				Role:    "system",
				Content: "You update README files for software projects. Return only the markdown content for a 'Recent Changes' section. Use a '## Recent Changes' heading, an italicised date line, and 2-3 bullet points describing what changed. Plain markdown only — no code fences, no extra commentary.",
			},
			{
				Role:    "user",
				Content: buildReadmePrompt(commitSummary, significance),
			},
		},
		Temperature: 0.3,
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
		return "", fmt.Errorf("llm generated empty readme section")
	}

	return generatedText, nil
}

func buildReadmePrompt(commitSummary git.CommitSummary, significance string) string {
	joinedFiles := "none"
	if len(commitSummary.ChangedFiles) > 0 {
		joinedFiles = strings.Join(commitSummary.ChangedFiles, ", ")
	}

	return fmt.Sprintf(
		"Commit: %s\nShort hash: %s\nAuthor: %s\nSignificance: %s\nFiles changed: %d\nInsertions: %d, Deletions: %d\nChanged files: %s\nWrite the 'Recent Changes' README section for this commit.",
		commitSummary.Subject,
		commitSummary.ShortHash,
		commitSummary.Author,
		significance,
		commitSummary.FilesChanged,
		commitSummary.Insertions,
		commitSummary.Deletions,
		joinedFiles,
	)
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
