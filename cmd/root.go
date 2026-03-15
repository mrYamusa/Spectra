package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var cfgPath string

var rootCmd = &cobra.Command{
	Use:   "spectra",
	Short: "Track git changes and generate changelog/README updates",
	Long:  "spectra is a Git-aware CLI scaffold for changelog generation and README updates with local or API LLM backends.",
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.PersistentFlags().StringVar(&cfgPath, "config", ".spectra.yaml", "Path to config file")
	rootCmd.SilenceUsage = true
	rootCmd.SetErrPrefix("spectra: ")

	rootCmd.AddCommand(newInitCmd())
	rootCmd.AddCommand(newTrackCmd())
	rootCmd.AddCommand(newUntrackCmd())
	rootCmd.AddCommand(newReadmeCmd())
	rootCmd.AddCommand(newDoctorCmd())
}

func printKV(key, value string) {
	fmt.Printf("%-18s %s\n", key+":", value)
}
