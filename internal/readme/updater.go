package readme

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"spectra/internal/git"
	"spectra/internal/llm"
)

// These HTML comment markers act as invisible bookmarks inside your README.
// spectra writes content between them and replaces it on every update.
// They are intentionally invisible when the README is rendered on GitHub.
const (
	managedSectionStartMarker = "<!-- spectra:readme:start -->"
	managedSectionEndMarker   = "<!-- spectra:readme:end -->"
)

// UpdateResult describes what spectra did to the README file.
type UpdateResult struct {
	ReadmeFilePath  string
	WasCreated      bool // true if the README didn't exist and was created from scratch
	SectionWasAdded bool // true if the managed markers didn't exist yet and were appended
	UsedLLM         bool // true if the section content came from the LLM
}

// FileUpdater manages updates to a single README file.
type FileUpdater struct {
	ReadmeFilePath string
}

// NewFileUpdater creates a FileUpdater that will manage the given README file.
func NewFileUpdater(readmeFilePath string) *FileUpdater {
	return &FileUpdater{ReadmeFilePath: readmeFilePath}
}

// ApplyCommitUpdate writes or refreshes the spectra-managed section of the README.
//
// How it works:
//  1. Generates new section content via LLM (falls back to deterministic text if unavailable).
//  2. Reads the existing README, or starts with empty content if it doesn't exist.
//  3. Replaces the content between the managed markers, or appends the markers if missing.
//  4. Writes the result back to disk.
func (updater *FileUpdater) ApplyCommitUpdate(
	ctx context.Context,
	commitSummary git.CommitSummary,
	significance SignificanceLevel,
	sectionGenerator llm.ReadmeSectionGenerator,
) (UpdateResult, error) {
	result := UpdateResult{ReadmeFilePath: updater.ReadmeFilePath}

	// Build the section content — try LLM first, fall back to deterministic text
	sectionContent := buildDeterministicReadmeSection(commitSummary, significance)
	if sectionGenerator != nil {
		generated, err := sectionGenerator.GenerateReadmeSectionUpdate(ctx, commitSummary, string(significance))
		if err == nil && strings.TrimSpace(generated) != "" {
			sectionContent = generated
			result.UsedLLM = true
		}
	}

	existingContent, wasCreated, err := readOrInitializeReadme(updater.ReadmeFilePath)
	if err != nil {
		return UpdateResult{}, err
	}
	result.WasCreated = wasCreated

	updatedContent, sectionWasAdded := applyManagedSection(existingContent, sectionContent)
	result.SectionWasAdded = sectionWasAdded

	if err := os.WriteFile(updater.ReadmeFilePath, []byte(updatedContent), 0o644); err != nil {
		return UpdateResult{}, fmt.Errorf("could not write README file: %w", err)
	}

	return result, nil
}

// readOrInitializeReadme reads the current README file.
// If it doesn't exist yet, returns empty content and wasCreated=true.
func readOrInitializeReadme(filePath string) (existingContent string, wasCreated bool, err error) {
	contentBytes, readErr := os.ReadFile(filePath)
	if readErr == nil {
		return string(contentBytes), false, nil
	}
	if os.IsNotExist(readErr) {
		return "", true, nil
	}
	return "", false, readErr
}

// applyManagedSection either replaces the content between the spectra markers
// (if they already exist in the README) or appends the whole managed block at the end.
//
// Example of what the managed block looks like in the README:
//
//	<!-- spectra:readme:start -->
//	## Recent Changes
//	...LLM or fallback content here...
//	<!-- spectra:readme:end -->
func applyManagedSection(existingContent string, newSectionContent string) (updatedContent string, sectionWasAdded bool) {
	managedBlock := managedSectionStartMarker + "\n" +
		strings.TrimSpace(newSectionContent) + "\n" +
		managedSectionEndMarker

	startIndex := strings.Index(existingContent, managedSectionStartMarker)
	endIndex := strings.Index(existingContent, managedSectionEndMarker)

	// Markers not present yet: append the managed block at the end
	if startIndex == -1 || endIndex == -1 {
		trimmed := strings.TrimRight(existingContent, "\n")
		if trimmed == "" {
			return managedBlock + "\n", true
		}
		return trimmed + "\n\n" + managedBlock + "\n", true
	}

	// Markers already present: surgically replace only what's between them
	contentBeforeMarker := existingContent[:startIndex]
	contentAfterMarker := existingContent[endIndex+len(managedSectionEndMarker):]
	return contentBeforeMarker + managedBlock + contentAfterMarker, false
}

// buildDeterministicReadmeSection produces a plain-text "Recent Changes" section
// when no LLM is available. Always produces valid, readable markdown.
func buildDeterministicReadmeSection(commitSummary git.CommitSummary, significance SignificanceLevel) string {
	var sectionBuilder strings.Builder

	sectionBuilder.WriteString("## Recent Changes\n\n")
	sectionBuilder.WriteString(fmt.Sprintf("_Last updated: %s_\n\n", time.Now().Format("2006-01-02")))
	sectionBuilder.WriteString(fmt.Sprintf(
		"- **%s** (`%s`) — %s change, %d files affected (+%d/-%d lines)\n",
		commitSummary.Subject,
		commitSummary.ShortHash,
		string(significance),
		commitSummary.FilesChanged,
		commitSummary.Insertions,
		commitSummary.Deletions,
	))

	return sectionBuilder.String()
}
