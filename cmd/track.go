package cmd

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"

	"github.com/spf13/cobra"

	"spectra/internal/changelog"
	"spectra/internal/config"
	"spectra/internal/git"
	"spectra/internal/llm"
)

func newTrackCmd() *cobra.Command {
	var commitRef string
	var commitRange string
	var changelogFilePath string

	cmd := &cobra.Command{
		Use:   "track",
		Short: "Generate and append changelog entries from git commits",
		RunE: func(cmd *cobra.Command, args []string) error {
			if commitRef != "" && commitRange != "" {
				return errors.New("use either --commit or --range, not both")
			}

			resolvedConfigPath := resolveConfigPathForRead(cfgPath)
			cfg, err := config.Load(resolvedConfigPath)
			if err != nil {
				return err
			}

			repo, err := git.NewClientFromWD()
			if err != nil {
				return err
			}

			resolvedChangelogPath := resolveRepoRelativePath(repo.RootPath, changelogFilePath)
			fileWriter := changelog.NewFileWriter(resolvedChangelogPath)

			var commitSummaries []git.CommitSummary
			selectionLabel := ""

			if commitRange != "" {
				entries, err := repo.SummarizeRange(commitRange)
				if err != nil {
					return err
				}
				commitSummaries = entries
				selectionLabel = fmt.Sprintf("range %s", commitRange)
			} else {
				if commitRef == "" {
					commitRef = "HEAD"
				}

				entry, err := repo.SummarizeCommit(commitRef)
				if err != nil {
					return err
				}

				commitSummaries = []git.CommitSummary{entry}
				selectionLabel = fmt.Sprintf("commit %s", commitRef)
			}

			summaryGenerator, llmSetupError := llm.NewGeneratorFromConfig(cfg)
			if llmSetupError != nil {
				summaryGenerator = nil
			}

			commitEntries, generationStats := changelog.BuildCommitEntries(context.Background(), commitSummaries, summaryGenerator)

			writeResult, err := fileWriter.AppendCommitEntries(commitEntries)
			if err != nil {
				return err
			}

			printKV("mode", cfg.Mode)
			printKV("source", selectionLabel)
			printKV("changelog", resolvedChangelogPath)
			if generationStats.FallbackUsedCount == 0 {
				printKV("summary_source", "llm")
			} else if generationStats.LLMSuccessCount == 0 {
				printKV("summary_source", "deterministic fallback")
				if llmSetupError != nil {
					printKV("llm_notice", llmSetupError.Error())
				}
			} else {
				printKV("summary_source", "mixed (llm + fallback)")
			}
			printKV("llm_summaries", fmt.Sprintf("%d", generationStats.LLMSuccessCount))
			printKV("fallback_summaries", fmt.Sprintf("%d", generationStats.FallbackUsedCount))
			printKV("added_entries", fmt.Sprintf("%d", writeResult.AddedEntries))
			printKV("skipped_entries", fmt.Sprintf("%d", writeResult.SkippedEntries))
			if writeResult.AddedEntries > 0 {
				fmt.Println("Changelog updated successfully.")
			} else {
				fmt.Println("No new changelog entries were added (all selected commits already logged).")
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&commitRef, "commit", "", "Commit reference (default HEAD)")
	cmd.Flags().StringVar(&commitRange, "range", "", "Commit range, e.g. main..HEAD")
	cmd.Flags().StringVar(&changelogFilePath, "changelog-file", "CHANGELOG.md", "Path to changelog file")
	return cmd
}

func resolveRepoRelativePath(repoRootPath string, requestedPath string) string {
	if filepath.IsAbs(requestedPath) {
		return requestedPath
	}
	return filepath.Join(repoRootPath, requestedPath)
}
