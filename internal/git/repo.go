package git

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

type CommitSummary struct {
	Hash         string
	ShortHash    string
	Subject      string
	Author       string
	Date         time.Time
	FilesChanged int
	Insertions   int
	Deletions    int
	ChangedFiles []string
	FileChanges  []FileChange
}

type FileChange struct {
	Path       string
	Insertions int
	Deletions  int
}

type Client struct {
	RootPath string
}

func NewClientFromWD() (*Client, error) {
	wd, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	root, err := runGit(wd, "rev-parse", "--show-toplevel")
	if err != nil {
		return &Client{RootPath: wd}, nil
	}
	return &Client{RootPath: strings.TrimSpace(root)}, nil
}

func (c *Client) IsGitRepo() (bool, error) {
	out, err := runGit(c.RootPath, "rev-parse", "--is-inside-work-tree")
	if err != nil {
		if strings.Contains(err.Error(), "not a git repository") {
			return false, nil
		}
		return false, err
	}
	return strings.TrimSpace(out) == "true", nil
}

func (c *Client) SummarizeCommit(ref string) (CommitSummary, error) {
	meta, err := runGit(c.RootPath, "show", "-s", "--format=%H%n%h%n%s%n%an%n%aI", ref)
	if err != nil {
		return CommitSummary{}, err
	}
	stats, err := runGit(c.RootPath, "show", "--numstat", "--format=", ref)
	if err != nil {
		return CommitSummary{}, err
	}
	return parseSummary(meta, stats)
}

func (c *Client) SummarizeRange(refRange string) ([]CommitSummary, error) {
	hashesRaw, err := runGit(c.RootPath, "rev-list", "--reverse", refRange)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(hashesRaw) == "" {
		return []CommitSummary{}, nil
	}

	hashes := strings.Split(strings.TrimSpace(hashesRaw), "\n")
	result := make([]CommitSummary, 0, len(hashes))
	for _, hash := range hashes {
		hash = strings.TrimSpace(hash)
		if hash == "" {
			continue
		}
		summary, err := c.SummarizeCommit(hash)
		if err != nil {
			return nil, err
		}
		result = append(result, summary)
	}
	return result, nil
}

func parseSummary(meta, stats string) (CommitSummary, error) {
	parts := strings.Split(strings.TrimSpace(meta), "\n")
	if len(parts) < 5 {
		return CommitSummary{}, fmt.Errorf("unexpected git metadata format")
	}

	date, err := time.Parse(time.RFC3339, parts[4])
	if err != nil {
		return CommitSummary{}, err
	}

	summary := CommitSummary{
		Hash:      parts[0],
		ShortHash: parts[1],
		Subject:   parts[2],
		Author:    parts[3],
		Date:      date,
	}

	for _, line := range strings.Split(stats, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		fields := strings.Split(line, "\t")
		if len(fields) < 3 {
			continue
		}

		filePath := fields[2]
		if filePath != "" {
			summary.ChangedFiles = append(summary.ChangedFiles, filePath)
			summary.FilesChanged++
		}

		ins, errIns := strconv.Atoi(fields[0])
		del, errDel := strconv.Atoi(fields[1])
		if errIns != nil || errDel != nil {
			continue
		}
		summary.Insertions += ins
		summary.Deletions += del
		if filePath != "" {
			summary.FileChanges = append(summary.FileChanges, FileChange{
				Path:       filePath,
				Insertions: ins,
				Deletions:  del,
			})
		}
	}

	return summary, nil
}

func runGit(repoPath string, args ...string) (string, error) {
	command := exec.Command("git", args...)
	command.Dir = repoPath

	var stdout, stderr bytes.Buffer
	command.Stdout = &stdout
	command.Stderr = &stderr

	if err := command.Run(); err != nil {
		if stderr.Len() > 0 {
			return "", fmt.Errorf("git %s failed: %s", strings.Join(args, " "), strings.TrimSpace(stderr.String()))
		}
		return "", fmt.Errorf("git %s failed: %w", strings.Join(args, " "), err)
	}

	return stdout.String(), nil
}
