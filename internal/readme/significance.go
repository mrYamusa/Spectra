package readme

import "spectra/internal/git"

// SignificanceLevel describes how impactful a commit is to end users.
// spectra uses this to decide whether a README update is warranted.
type SignificanceLevel string

const (
	// SignificanceLow means a small or internal change (e.g. typo fix, refactor).
	// Policy: changelog only, README left untouched.
	SignificanceLow SignificanceLevel = "low"

	// SignificanceMedium means a user-visible change (e.g. new flag, behavior change).
	// Policy: changelog + README "Recent Changes" section updated.
	SignificanceMedium SignificanceLevel = "medium"

	// SignificanceHigh means a major or breaking change (e.g. new feature, API change).
	// Policy: changelog + README updated prominently.
	SignificanceHigh SignificanceLevel = "high"
)

// ScoreCommitSignificance decides the impact level of a commit based on
// how many files it touched and how many lines it changed.
//
//   - high:   8+ files changed, or 120+ total line changes
//   - medium: 3+ files changed, or 40+ total line changes
//   - low:    everything else
func ScoreCommitSignificance(commitSummary git.CommitSummary) SignificanceLevel {
	totalLinesChanged := commitSummary.Insertions + commitSummary.Deletions

	if commitSummary.FilesChanged >= 8 || totalLinesChanged >= 120 {
		return SignificanceHigh
	}
	if commitSummary.FilesChanged >= 3 || totalLinesChanged >= 40 {
		return SignificanceMedium
	}
	return SignificanceLow
}

// MeetsThreshold returns true when a commit's significance is at or above
// the threshold set in .spectra.yaml (readme_threshold key).
// If the threshold is unrecognized it defaults to medium behavior.
func MeetsThreshold(significance SignificanceLevel, configuredThreshold string) bool {
	switch configuredThreshold {
	case "low":
		// Any change at all qualifies
		return true
	case "high":
		// Only major/breaking changes qualify
		return significance == SignificanceHigh
	default:
		// "medium" and anything else: medium or high qualifies
		return significance == SignificanceMedium || significance == SignificanceHigh
	}
}
