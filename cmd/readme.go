package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"spectra/internal/config"
	"spectra/internal/git"
)

func newReadmeCmd() *cobra.Command {
	var commitRef string
	var auto bool

	cmd := &cobra.Command{
		Use:   "readme",
		Short: "Evaluate README update necessity from commit significance",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load(cfgPath)
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

			entry, err := repo.SummarizeCommit(commitRef)
			if err != nil {
				return err
			}

			significance := "low"
			if entry.FilesChanged >= 8 || entry.Insertions+entry.Deletions >= 120 {
				significance = "high"
			} else if entry.FilesChanged >= 3 || entry.Insertions+entry.Deletions >= 40 {
				significance = "medium"
			}

			printKV("mode", cfg.Mode)
			printKV("commit", entry.ShortHash)
			printKV("significance", significance)
			printKV("auto", fmt.Sprintf("%t", auto))

			if auto && (significance == "medium" || significance == "high") {
				fmt.Println("README update would be triggered in next milestone implementation.")
			} else {
				fmt.Println("README update skipped by current policy.")
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&commitRef, "commit", "", "Commit reference (default HEAD)")
	cmd.Flags().BoolVar(&auto, "auto", false, "Run auto-update policy")
	return cmd
}
