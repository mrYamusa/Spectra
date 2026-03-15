package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	Mode                  string
	LocalBaseURL          string
	APIBaseURL            string
	Model                 string
	APIKeyEnv             string
	ReadmeLevel           string
	RequestTimeoutSeconds int
}

func Default() Config {
	return Config{
		Mode:                  "local",
		LocalBaseURL:          "http://localhost:11434/v1",
		APIBaseURL:            "https://api.openai.com/v1",
		Model:                 "llama3.1",
		APIKeyEnv:             "SPECTRA_API_KEY",
		ReadmeLevel:           "medium",
		RequestTimeoutSeconds: 30,
	}
}

func Load(path string) (Config, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return Config{}, err
	}

	config := Default()
	for _, rawLine := range strings.Split(string(content), "\n") {
		line := strings.TrimSpace(rawLine)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		value := strings.Trim(strings.TrimSpace(parts[1]), "\"")
		switch key {
		case "mode":
			config.Mode = value
		case "local_base_url":
			config.LocalBaseURL = value
		case "model":
			config.Model = value
		case "api_base_url":
			config.APIBaseURL = value
		case "api_key_env":
			config.APIKeyEnv = value
		case "readme_threshold":
			config.ReadmeLevel = value
		case "request_timeout_seconds":
			parsedTimeout, parseErr := strconv.Atoi(value)
			if parseErr == nil {
				config.RequestTimeoutSeconds = parsedTimeout
			}
		}
	}

	if err := validate(config); err != nil {
		return Config{}, err
	}
	return config, nil
}

func (c Config) WriteIfMissing(path string, force bool) error {
	if _, err := os.Stat(path); err == nil && !force {
		return nil
	}

	content := fmt.Sprintf("# spectra configuration\nmode: %s\nlocal_base_url: %s\napi_base_url: %s\nmodel: %s\napi_key_env: %s\nreadme_threshold: %s\nrequest_timeout_seconds: %d\n",
		c.Mode,
		c.LocalBaseURL,
		c.APIBaseURL,
		c.Model,
		c.APIKeyEnv,
		c.ReadmeLevel,
		c.RequestTimeoutSeconds,
	)

	return os.WriteFile(path, []byte(content), 0o644)
}

func validate(c Config) error {
	switch c.Mode {
	case "local", "api":
	default:
		return errors.New("config mode must be one of: local, api")
	}

	switch c.ReadmeLevel {
	case "low", "medium", "high":
	default:
		return errors.New("readme_threshold must be one of: low, medium, high")
	}

	if c.Model == "" {
		return errors.New("model cannot be empty")
	}
	if c.APIKeyEnv == "" {
		return errors.New("api_key_env cannot be empty")
	}
	if c.Mode == "local" && c.LocalBaseURL == "" {
		return errors.New("local_base_url cannot be empty in local mode")
	}
	if c.Mode == "api" && c.APIBaseURL == "" {
		return errors.New("api_base_url cannot be empty in api mode")
	}
	if c.RequestTimeoutSeconds <= 0 {
		return errors.New("request_timeout_seconds must be greater than 0")
	}

	return nil
}
