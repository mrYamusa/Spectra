package cmd

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"spectra/internal/config"
	"spectra/internal/git"
)

func newInitCmd() *cobra.Command {
	var force bool

	cmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize glonag config and git hook",
		RunE: func(cmd *cobra.Command, args []string) error {
			repo, err := git.NewClientFromWD()
			if err != nil {
				return err
			}

			ok, err := repo.IsGitRepo()
			if err != nil {
				return err
			}
			if !ok {
				return errors.New("current directory is not a git repository")
			}

			cfg := config.Default()
			if err := cfg.WriteIfMissing(cfgPath, force); err != nil {
				return err
			}

			hookPath := filepath.Join(repo.RootPath, ".git", "hooks", "post-commit")
			hookBody := "#!/usr/bin/env sh\n" +
				"command -v glonag >/dev/null 2>&1 || exit 0\n" +
				"glonag track --commit HEAD\n"

			if _, statErr := os.Stat(hookPath); statErr == nil && !force {
				fmt.Printf("hook already exists at %s (use --force to overwrite)\n", hookPath)
			} else {
				if err := os.WriteFile(hookPath, []byte(hookBody), 0o755); err != nil {
					return err
				}
				fmt.Printf("installed git hook: %s\n", hookPath)
			}

			fmt.Printf("config ready: %s\n", cfgPath)
			return nil
		},
	}

	cmd.Flags().BoolVar(&force, "force", false, "Overwrite existing config and hook")
	return cmd
}
