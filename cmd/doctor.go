package cmd

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/spf13/cobra"

	"spectra/internal/config"
	"spectra/internal/git"
)

func newDoctorCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "doctor",
		Short: "Check local prerequisites",
		RunE: func(cmd *cobra.Command, args []string) error {
			if _, err := exec.LookPath("git"); err != nil {
				return fmt.Errorf("git not found in PATH: %w", err)
			}
			printKV("git", "ok")

			repo, err := git.NewClientFromWD()
			if err != nil {
				return err
			}
			ok, err := repo.IsGitRepo()
			if err != nil {
				return err
			}
			if !ok {
				return fmt.Errorf("not a git repository")
			}
			printKV("repo", repo.RootPath)

			if _, err := os.Stat(cfgPath); err != nil {
				if os.IsNotExist(err) {
					printKV("config", "missing (run: glonag init)")
					return nil
				}
				return err
			}

			cfg, err := config.Load(cfgPath)
			if err != nil {
				return err
			}
			printKV("config", "ok")
			printKV("mode", cfg.Mode)

			if cfg.Mode == "api" {
				if os.Getenv(cfg.APIKeyEnv) == "" {
					printKV("api_key", fmt.Sprintf("missing (%s)", cfg.APIKeyEnv))
				} else {
					printKV("api_key", fmt.Sprintf("present (%s)", cfg.APIKeyEnv))
				}
			}
			return nil
		},
	}
}
