package changelog

import (
	"context"
	"fmt"
	"sort"
	"strings"

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
	FirstLLMError      string
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
			} else if err != nil && stats.FirstLLMError == "" {
				stats.FirstLLMError = err.Error()
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
	lead := buildSummaryLead(commitSummary.Subject)
	topChanges := buildTopChangeSnippet(commitSummary)
	impact := inferImpactArea(commitSummary)

	return fmt.Sprintf("%s across %d files (+%d/-%d), with the largest edits in %s, improving %s.",
		lead,
		commitSummary.FilesChanged,
		commitSummary.Insertions,
		commitSummary.Deletions,
		topChanges,
		impact,
	)
}

func buildSummaryLead(subject string) string {
	trimmed := strings.TrimSpace(subject)
	trimmed = strings.Trim(trimmed, ".!?")
	if trimmed == "" {
		return "Updated project files"
	}

	runes := []rune(trimmed)
	runes[0] = []rune(strings.ToUpper(string(runes[0])))[0]
	return string(runes)
}

func buildTopChangeSnippet(commitSummary git.CommitSummary) string {
	if len(commitSummary.FileChanges) > 0 {
		top := make([]git.FileChange, len(commitSummary.FileChanges))
		copy(top, commitSummary.FileChanges)
		sort.SliceStable(top, func(left, right int) bool {
			leftChurn := top[left].Insertions + top[left].Deletions
			rightChurn := top[right].Insertions + top[right].Deletions
			if leftChurn == rightChurn {
				return top[left].Path < top[right].Path
			}
			return leftChurn > rightChurn
		})

		limit := 2
		if len(top) < limit {
			limit = len(top)
		}

		parts := make([]string, 0, limit)
		for _, changedFile := range top[:limit] {
			parts = append(parts, fmt.Sprintf("%s (+%d/-%d)", changedFile.Path, changedFile.Insertions, changedFile.Deletions))
		}
		return strings.Join(parts, " and ")
	}

	if len(commitSummary.ChangedFiles) == 0 {
		return "core project files"
	}

	limit := 2
	if len(commitSummary.ChangedFiles) < limit {
		limit = len(commitSummary.ChangedFiles)
	}

	return strings.Join(commitSummary.ChangedFiles[:limit], " and ")
}

func inferImpactArea(commitSummary git.CommitSummary) string {
	allPaths := make([]string, 0, len(commitSummary.ChangedFiles)+len(commitSummary.FileChanges))
	allPaths = append(allPaths, commitSummary.ChangedFiles...)
	for _, changedFile := range commitSummary.FileChanges {
		allPaths = append(allPaths, changedFile.Path)
	}

	lowerSubject := strings.ToLower(commitSummary.Subject)
	if strings.Contains(lowerSubject, "readme") || hasPathPrefix(allPaths, "readme") || hasPathPrefix(allPaths, "docs/") {
		return "documentation clarity and onboarding"
	}
	if hasPathPrefix(allPaths, "cmd/") {
		return "cli behavior and developer workflow"
	}
	if hasPathPrefix(allPaths, "internal/config/") || strings.Contains(lowerSubject, "config") {
		return "configuration reliability"
	}
	if hasPathPrefix(allPaths, "internal/changelog/") || strings.Contains(lowerSubject, "changelog") {
		return "changelog quality and release visibility"
	}
	if strings.Contains(lowerSubject, "fix") || strings.Contains(lowerSubject, "bug") {
		return "runtime correctness"
	}

	return "project maintainability"
}

func hasPathPrefix(paths []string, prefix string) bool {
	for _, path := range paths {
		if strings.HasPrefix(strings.ToLower(path), prefix) {
			return true
		}
	}
	return false
}
