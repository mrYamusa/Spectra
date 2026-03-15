package cmd

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"spectra/internal/config"
	"spectra/internal/git"
	"spectra/internal/llm"
	readmepkg "spectra/internal/readme"
)

func newReadmeCmd() *cobra.Command {
	var commitRef string
	var auto bool
	var readmeFilePath string

	cmd := &cobra.Command{
		Use:   "readme",
		Short: "Update README.md based on commit significance",
		Long: `Evaluates how significant a commit is and updates the spectra-managed
section of README.md when the significance meets your configured threshold.

Without --auto this is a dry run: it shows what would happen without writing anything.
With --auto it writes the update to the README file.

The managed section is wrapped in invisible HTML comment markers so spectra
can safely update it on every qualifying commit without touching the rest of
your README.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			resolvedConfigPath := resolveConfigPathForRead(cfgPath)
			cfg, err := config.Load(resolvedConfigPath)
			if err != nil {
				return err
			}

			repo, err := git.NewClientFromWD()
			if err != nil {
				return err
			}

			if commitRef == "" {
				commitRef = "HEAD"
			}

			commit, err := repo.SummarizeCommit(commitRef)
			if err != nil {
				return err
			}

			significance := readmepkg.ScoreCommitSignificance(commit)
			meetsPolicy := readmepkg.MeetsThreshold(significance, cfg.ReadmeLevel)

			printKV("mode", cfg.Mode)
			printKV("commit", commit.ShortHash)
			printKV("significance", string(significance))
			printKV("threshold", cfg.ReadmeLevel)
			printKV("qualifies", fmt.Sprintf("%t", meetsPolicy))

			// Without --auto, just show the diagnosis and exit (safe dry run)
			if !auto {
				if meetsPolicy {
					fmt.Println("Would update README.md — run with --auto to apply.")
				} else {
					fmt.Printf("README update not needed (significance %q is below threshold %q).\n",
						significance, cfg.ReadmeLevel)
				}
				return nil
			}

			// With --auto but below threshold, skip gracefully
			if !meetsPolicy {
				fmt.Printf("README update skipped (significance %q is below threshold %q).\n",
					significance, cfg.ReadmeLevel)
				return nil
			}

			// Resolve the README file path relative to the repo root
			resolvedReadmePath := resolveRepoRelativePath(repo.RootPath, readmeFilePath)
			fileUpdater := readmepkg.NewFileUpdater(resolvedReadmePath)

			// Try to get an LLM-powered section generator; fall back gracefully if unavailable
			sectionGenerator, llmErr := llm.NewReadmeGeneratorFromConfig(cfg)
			if llmErr != nil {
				sectionGenerator = nil
			}

			updateResult, err := fileUpdater.ApplyCommitUpdate(
				context.Background(),
				commit,
				significance,
				sectionGenerator,
			)
			if err != nil {
				return err
			}

			printKV("readme", updateResult.ReadmeFilePath)

			switch {
			case updateResult.WasCreated:
				printKV("action", "created README.md with managed section")
			case updateResult.SectionWasAdded:
				printKV("action", "appended managed section to existing README.md")
			default:
				printKV("action", "updated managed section in README.md")
			}

			if updateResult.UsedLLM {
				printKV("summary_source", "llm")
			} else {
				printKV("summary_source", "deterministic fallback")
				if llmErr != nil {
					printKV("llm_notice", llmErr.Error())
				}
			}

			fmt.Println("README updated successfully.")
			return nil
		},
	}

	cmd.Flags().StringVar(&commitRef, "commit", "", "Commit reference (default HEAD)")
	cmd.Flags().BoolVar(&auto, "auto", false, "Apply the update (omitting this flag is a safe dry run)")
	cmd.Flags().StringVar(&readmeFilePath, "readme-file", "README.md", "Path to README file")
	return cmd
}

// scoreCommitSignificance is kept here as a lightweight local alias
// so other cmd files can reference the scorer without importing the readme package.
func scoreCommitSignificance(commit git.CommitSummary) string {
	return string(readmepkg.ScoreCommitSignificance(commit))
}
