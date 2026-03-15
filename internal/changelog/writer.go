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
	// Start from the existing content with trailing newlines stripped.
	// We re-add exactly one trailing newline at the very end.
	result := strings.TrimRight(existingContent, "\n")

	sortedDates := make([]string, 0, len(entriesGroupedByDate))
	for dateLabel := range entriesGroupedByDate {
		sortedDates = append(sortedDates, dateLabel)
	}
	sort.Strings(sortedDates)

	for _, dateLabel := range sortedDates {
		newLines := ""
		for _, commitEntry := range entriesGroupedByDate[dateLabel] {
			newLines += formatEntryLine(commitEntry)
		}

		headingLine := "## " + dateLabel + "\n"
		sectionStart := strings.Index(result, headingLine)

		if sectionStart != -1 {
			// This date section already exists in the file.
			// Find where its body ends (just before the next section heading or EOF)
			// and inject the new entries there instead of creating a duplicate heading.
			sectionBodyStart := sectionStart + len(headingLine)
			afterBody := result[sectionBodyStart:]
			nextSectionRel := strings.Index(afterBody, "\n## ")

			var insertAt int
			if nextSectionRel == -1 {
				// Section body runs to end of file
				insertAt = len(result)
			} else {
				// +1 to skip past the \n that precedes "## <next-date>"
				insertAt = sectionBodyStart + nextSectionRel + 1
			}

			prefix := result[:insertAt]
			suffix := result[insertAt:]
			// Guard: make sure there is a newline separating the existing last
			// entry from the one we are about to add.
			if !strings.HasSuffix(prefix, "\n") {
				prefix += "\n"
			}
			result = prefix + newLines + suffix
		} else {
			// Date section does not exist yet — append a brand-new section.
			result += "\n\n## " + dateLabel + "\n" + newLines
		}
	}

	return strings.TrimRight(result, "\n") + "\n"
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
	return fmt.Sprintf("<!-- spectra:commit:%s -->", fullHash)
}

func isCommitAlreadyLogged(changelogContent string, commitHash string) bool {
	currentMarker := formatCommitHashMarker(commitHash)
	legacyMarker := fmt.Sprintf("<!-- glonag:commit:%s -->", commitHash)
	return strings.Contains(changelogContent, currentMarker) || strings.Contains(changelogContent, legacyMarker)
}
