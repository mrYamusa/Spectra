package changelog

import (
	"context"
	"fmt"

	"spectra/internal/git"
	"spectra/internal/llm"
)

type CommitEntry struct {
	CommitSummary git.CommitSummary
	SummaryText   string
}

type SummaryGenerationStats struct {
	TotalEntries       int
	LLMSuccessCount    int
	FallbackUsedCount  int
	GeneratorAvailable bool
}

func BuildCommitEntries(ctx context.Context, commitSummaries []git.CommitSummary, summaryGenerator llm.CommitSummaryGenerator) ([]CommitEntry, SummaryGenerationStats) {
	entries := make([]CommitEntry, 0, len(commitSummaries))
	stats := SummaryGenerationStats{
		TotalEntries:       len(commitSummaries),
		GeneratorAvailable: summaryGenerator != nil,
	}

	for _, commitSummary := range commitSummaries {
		summaryText := buildDeterministicSummary(commitSummary)
		usedFallback := true
		if summaryGenerator != nil {
			generatedSummaryText, err := summaryGenerator.GenerateCommitSummaryText(ctx, commitSummary)
			if err == nil && generatedSummaryText != "" {
				summaryText = generatedSummaryText
				usedFallback = false
			}
		}

		if usedFallback {
			stats.FallbackUsedCount++
		} else {
			stats.LLMSuccessCount++
		}

		entries = append(entries, CommitEntry{
			CommitSummary: commitSummary,
			SummaryText:   summaryText,
		})
	}

	return entries, stats
}

func buildDeterministicSummary(commitSummary git.CommitSummary) string {
	return fmt.Sprintf("%s (changed %d files).", commitSummary.Subject, commitSummary.FilesChanged)
}
