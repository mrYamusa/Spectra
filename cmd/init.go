package cmd

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"spectra/internal/config"
	"spectra/internal/git"
)

// spectraBanner is a compact pixel-art logo shown on the very first run.
// Each row of block characters spells out S-P-E-C-T-R-A at ~60 columns wide.
const spectraBanner = `
  ███████  ██████  ███████  ██████  ████████  ██████   █████
  ██       ██  ██  ██       ██         ██     ██   ██ ██   ██
  ███████  ██████  █████    ██         ██     ██████  ███████
       ██  ██      ██       ██         ██     ██  ██  ██   ██
  ███████  ██      ███████  ██████     ██     ██   ██ ██   ██

              · document those changes ·
`

func newInitCmd() *cobra.Command {
	var force bool
	var skipWizard bool

	cmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize spectra config and git hook",
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

			// Show the wizard only when this is the user's first time (no config exists yet)
			isFirstTime := !fileExists(cfgPath) && !fileExists(defaultLegacyConfigPath)

			var cfg config.Config
			if isFirstTime && !skipWizard {
				fmt.Print(spectraBanner)
				fmt.Println()
				fmt.Println(wizardDivider())
				fmt.Println("  Welcome to Spectra! Let's set you up in under a minute.")
				fmt.Println("  Press Enter on any question to accept the [default].")
				fmt.Println(wizardDivider())
				fmt.Println()

				cfg, err = runSetupWizard()
				if err != nil {
					return err
				}
			} else {
				cfg = config.Default()
			}

			if err := cfg.WriteIfMissing(cfgPath, force); err != nil {
				return err
			}

			hookPath := filepath.Join(repo.RootPath, ".git", "hooks", "post-commit")
			hookBody := "#!/usr/bin/env sh\n" +
				"command -v spectra >/dev/null 2>&1 || exit 0\n" +
				"spectra track --commit HEAD\n"

			if _, statErr := os.Stat(hookPath); statErr == nil && !force {
				fmt.Printf("hook already exists at %s (use --force to overwrite)\n", hookPath)
			} else {
				if err := os.WriteFile(hookPath, []byte(hookBody), 0o755); err != nil {
					return err
				}
				fmt.Printf("installed git hook: %s\n", hookPath)
			}

			fmt.Printf("config ready: %s\n", cfgPath)
			fmt.Println()
			fmt.Println("  All done!  Run `spectra doctor` to verify your setup.")
			fmt.Println()
			return nil
		},
	}

	cmd.Flags().BoolVar(&force, "force", false, "Overwrite existing config and hook")
	cmd.Flags().BoolVar(&skipWizard, "no-wizard", false, "Skip interactive setup and write default config")
	return cmd
}

// wizardDivider is a horizontal rule used to section the wizard output.
func wizardDivider() string {
	return "  " + strings.Repeat("─", 54)
}

// runSetupWizard asks the user each config question in turn, showing helpful examples.
// Pressing Enter on any question accepts the shown [default] value.
func runSetupWizard() (config.Config, error) {
	cfg := config.Default()
	reader := bufio.NewReader(os.Stdin)

	// ── Mode ──────────────────────────────────────────────────────────────
	askQuestion(
		"Mode",
		"How should Spectra connect to an AI model?",
		[]string{
			"local  →  a model running on your own machine (e.g. Ollama, LM Studio)",
			"api    →  a cloud model via API (e.g. OpenAI, Gemini, Groq)",
		},
		"local",
	)
	mode := collectLine(reader, "local")
	if mode != "local" && mode != "api" {
		fmt.Println("  ⚠  Unrecognized value — defaulting to \"local\".")
		mode = "local"
	}
	cfg.Mode = mode
	fmt.Println()

	// ── URL ───────────────────────────────────────────────────────────────
	if mode == "local" {
		askQuestion(
			"Local base URL",
			"The URL of your local model server (must be OpenAI-API-compatible).",
			[]string{
				"Ollama (default):  http://localhost:11434/v1",
				"LM Studio:         http://localhost:1234/v1",
			},
			cfg.LocalBaseURL,
		)
		cfg.LocalBaseURL = collectLine(reader, cfg.LocalBaseURL)
		fmt.Println()
	} else {
		askQuestion(
			"API base URL",
			"The base URL your cloud model provider uses.",
			[]string{
				"OpenAI:   https://api.openai.com/v1",
				"Gemini:   https://generativelanguage.googleapis.com/v1beta/openai",
				"Groq:     https://api.groq.com/openai/v1",
			},
			cfg.APIBaseURL,
		)
		cfg.APIBaseURL = collectLine(reader, cfg.APIBaseURL)
		fmt.Println()

		askQuestion(
			"API key environment variable",
			"The name of the env var that holds your API key (not the key itself!).",
			[]string{
				"e.g.  OPENAI_API_KEY, GEMINI_API_KEY, GROQ_API_KEY",
				"Set it in your terminal:  export OPENAI_API_KEY=\"sk-...\"",
				"Spectra reads the variable at runtime — your key is never stored in the file.",
			},
			cfg.APIKeyEnv,
		)
		cfg.APIKeyEnv = collectLine(reader, cfg.APIKeyEnv)
		fmt.Println()
	}

	// ── Model name ────────────────────────────────────────────────────────
	defaultModel := "llama3.1"
	if mode == "api" {
		defaultModel = "gpt-4o-mini"
	}
	askQuestion(
		"Model name",
		"The exact model identifier your provider uses.",
		[]string{
			"Local examples:  llama3.1, mistral, codellama, phi3, qwen2.5",
			"API examples:    gpt-4o-mini, gemini-2.0-flash, claude-3-haiku, llama3-70b-8192",
		},
		defaultModel,
	)
	cfg.Model = collectLine(reader, defaultModel)
	fmt.Println()

	// ── README threshold ──────────────────────────────────────────────────
	askQuestion(
		"README update threshold",
		"How significant must a commit be to trigger a README.md update?",
		[]string{
			"low    →  update for every commit",
			"medium →  only when 3+ files or 40+ lines changed  (default, recommended)",
			"high   →  only when 8+ files or 120+ lines changed",
		},
		cfg.ReadmeLevel,
	)
	cfg.ReadmeLevel = collectLine(reader, cfg.ReadmeLevel)
	fmt.Println()

	fmt.Println(wizardDivider())
	return cfg, nil
}

// askQuestion prints a neatly-boxed prompt with a description and example values.
func askQuestion(field, description string, examples []string, defaultValue string) {
	fmt.Printf("  ┌─ %s\n", field)
	fmt.Printf("  │  %s\n", description)
	for _, ex := range examples {
		fmt.Printf("  │    ▸ %s\n", ex)
	}
	fmt.Printf("  └▶ [%s]: ", defaultValue)
}

// collectLine reads one line from stdin.
// If the user just presses Enter without typing anything, defaultValue is returned.
func collectLine(reader *bufio.Reader, defaultValue string) string {
	line, _ := reader.ReadString('\n')
	line = strings.TrimRight(line, "\r\n")
	line = strings.TrimSpace(line)
	if line == "" {
		return defaultValue
	}
	return line
}
