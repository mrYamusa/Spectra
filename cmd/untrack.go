package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"spectra/internal/changelog"
	"spectra/internal/git"
)

func newUntrackCmd() *cobra.Command {
	var commitRef string
	var changelogFilePath string

	cmd := &cobra.Command{
		Use:   "untrack",
		Short: "Remove a commit entry from CHANGELOG.md",
		Long: `Removes the changelog entry for a specific commit.
If the date section becomes empty after removal, the section heading
is cleaned up too. This is the reverse of 'spectra track'.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if commitRef == "" {
				commitRef = "HEAD"
			}

			repo, err := git.NewClientFromWD()
			if err != nil {
				return err
			}

			commit, err := repo.SummarizeCommit(commitRef)
			if err != nil {
				return err
			}

			resolvedChangelogPath := resolveRepoRelativePath(repo.RootPath, changelogFilePath)
			fileWriter := changelog.NewFileWriter(resolvedChangelogPath)

			removed, err := fileWriter.RemoveCommitEntry(commit.Hash)
			if err != nil {
				return err
			}

			printKV("commit", commit.ShortHash)
			printKV("subject", commit.Subject)
			printKV("changelog", resolvedChangelogPath)
			if removed {
				printKV("result", "entry removed")
				fmt.Println("Changelog entry removed successfully.")
			} else {
				printKV("result", "not found")
				fmt.Println("No entry found for this commit (nothing changed).")
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&commitRef, "commit", "", "Commit reference to remove (default HEAD)")
	cmd.Flags().StringVar(&changelogFilePath, "changelog-file", "CHANGELOG.md", "Path to changelog file")
	return cmd
}
