package cmd

import "os"

const (
	defaultCurrentConfigPath = ".spectra.yaml"
	defaultLegacyConfigPath  = ".glonag.yaml"
)

func resolveConfigPathForRead(requestedPath string) string {
	if requestedPath != defaultCurrentConfigPath {
		return requestedPath
	}

	if fileExists(defaultCurrentConfigPath) {
		return defaultCurrentConfigPath
	}

	if fileExists(defaultLegacyConfigPath) {
		return defaultLegacyConfigPath
	}

	return requestedPath
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
