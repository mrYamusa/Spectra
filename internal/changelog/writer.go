package changelog

import (
	"fmt"
	"os"
	"sort"
	"strings"
)

const defaultHeading = "# Changelog\n\n"

type WriteResult struct {
	AddedEntries   int
	SkippedEntries int
}

type FileWriter struct {
	FilePath string
}

func NewFileWriter(filePath string) *FileWriter {
	return &FileWriter{FilePath: filePath}
}

func (writer *FileWriter) AppendCommitEntries(commitEntries []CommitEntry) (WriteResult, error) {
	existingContent, err := writer.readOrInitializeFile()
	if err != nil {
		return WriteResult{}, err
	}

	entriesGroupedByDate := make(map[string][]CommitEntry)
	result := WriteResult{}

	for _, commitEntry := range commitEntries {
		if isCommitAlreadyLogged(existingContent, commitEntry.CommitSummary.Hash) {
			result.SkippedEntries++
			continue
		}

		dateLabel := commitEntry.CommitSummary.Date.Format("2006-01-02")
		entriesGroupedByDate[dateLabel] = append(entriesGroupedByDate[dateLabel], commitEntry)
		result.AddedEntries++
	}

	if result.AddedEntries == 0 {
		return result, nil
	}

	newContent := buildUpdatedChangelogContent(existingContent, entriesGroupedByDate)
	if err := os.WriteFile(writer.FilePath, []byte(newContent), 0o644); err != nil {
		return WriteResult{}, err
	}

	return result, nil
}

func buildUpdatedChangelogContent(existingContent string, entriesGroupedByDate map[string][]CommitEntry) string {
	var outputBuilder strings.Builder
	outputBuilder.WriteString(strings.TrimRight(existingContent, "\n"))

	sortedDates := make([]string, 0, len(entriesGroupedByDate))
	for dateLabel := range entriesGroupedByDate {
		sortedDates = append(sortedDates, dateLabel)
	}
	sort.Strings(sortedDates)

	for _, dateLabel := range sortedDates {
		outputBuilder.WriteString("\n\n## ")
		outputBuilder.WriteString(dateLabel)
		outputBuilder.WriteString("\n")

		for _, commitEntry := range entriesGroupedByDate[dateLabel] {
			outputBuilder.WriteString(formatEntryLine(commitEntry))
		}
	}

	outputBuilder.WriteString("\n")
	return outputBuilder.String()
}

func (writer *FileWriter) readOrInitializeFile() (string, error) {
	contentBytes, err := os.ReadFile(writer.FilePath)
	if err == nil {
		content := string(contentBytes)
		if strings.TrimSpace(content) == "" {
			return defaultHeading, nil
		}
		return ensureHeading(content), nil
	}

	if !os.IsNotExist(err) {
		return "", err
	}

	if writeErr := os.WriteFile(writer.FilePath, []byte(defaultHeading), 0o644); writeErr != nil {
		return "", writeErr
	}
	return defaultHeading, nil
}

func ensureHeading(existingContent string) string {
	trimmed := strings.TrimSpace(existingContent)
	if strings.HasPrefix(trimmed, "# Changelog") {
		return existingContent
	}
	return defaultHeading + existingContent
}

func formatEntryLine(commitEntry CommitEntry) string {
	return fmt.Sprintf("- %s (`%s`) by %s — files: %d, +%d/-%d %s\n",
		commitEntry.SummaryText,
		commitEntry.CommitSummary.ShortHash,
		commitEntry.CommitSummary.Author,
		commitEntry.CommitSummary.FilesChanged,
		commitEntry.CommitSummary.Insertions,
		commitEntry.CommitSummary.Deletions,
		formatCommitHashMarker(commitEntry.CommitSummary.Hash),
	)
}

func formatCommitHashMarker(fullHash string) string {
	return fmt.Sprintf("<!-- glonag:commit:%s -->", fullHash)
}

func isCommitAlreadyLogged(changelogContent string, commitHash string) bool {
	return strings.Contains(changelogContent, formatCommitHashMarker(commitHash))
}
