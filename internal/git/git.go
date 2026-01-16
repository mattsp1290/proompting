// Package git provides shared git operations for vibes commands.
package git

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/vibes-project/vibes/internal/runner"
)

// StatusCounts holds counts of different file states in the working tree.
type StatusCounts struct {
	Staged    int
	Modified  int
	Untracked int
}

// GetCurrentBranch returns the current git branch name.
func GetCurrentBranch(dir string, r runner.CommandRunner) string {
	branch, err := r.Run(dir, "git", "rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		return ""
	}
	return branch
}

// GetWorkingTreeStatus returns a summary string of the working tree status.
// Returns empty string if working tree is clean.
func GetWorkingTreeStatus(dir string, r runner.CommandRunner) string {
	counts := GetStatusCounts(dir, r)
	return FormatStatusCounts(counts)
}

// GetStatusCounts returns counts of staged, modified, and untracked files.
func GetStatusCounts(dir string, r runner.CommandRunner) StatusCounts {
	status, err := r.Run(dir, "git", "status", "--porcelain")
	if err != nil || status == "" {
		return StatusCounts{}
	}

	lines := strings.Split(strings.TrimSpace(status), "\n")
	var counts StatusCounts

	for _, line := range lines {
		if len(line) < 2 {
			continue
		}
		index := line[0]
		worktree := line[1]
		if index == '?' {
			counts.Untracked++
		} else if index != ' ' {
			counts.Staged++
		}
		if worktree != ' ' && worktree != '?' {
			counts.Modified++
		}
	}

	return counts
}

// FormatStatusCounts formats status counts as a human-readable string.
// Returns empty string if all counts are zero.
func FormatStatusCounts(counts StatusCounts) string {
	parts := []string{}
	if counts.Staged > 0 {
		parts = append(parts, fmt.Sprintf("%d staged", counts.Staged))
	}
	if counts.Modified > 0 {
		parts = append(parts, fmt.Sprintf("%d modified", counts.Modified))
	}
	if counts.Untracked > 0 {
		parts = append(parts, fmt.Sprintf("%d untracked", counts.Untracked))
	}

	if len(parts) > 0 {
		return strings.Join(parts, ", ")
	}
	return ""
}

// GetBranchCommits returns commits on the current branch that aren't on main/master.
// For main/master branches, returns the 5 most recent commits.
func GetBranchCommits(dir string, branch string, r runner.CommandRunner) string {
	if branch == "" || branch == "main" || branch == "master" {
		// On main branch, show recent commits instead
		output, err := r.Run(dir, "git", "log", "-5", "--oneline")
		if err != nil {
			return ""
		}
		return output
	}

	// Get commits on this branch that aren't on main/master
	output, err := r.Run(dir, "git", "log", "--oneline", "main..HEAD")
	if err != nil || output == "" {
		output, err = r.Run(dir, "git", "log", "--oneline", "master..HEAD")
		if err != nil {
			return ""
		}
	}

	if output == "" {
		// No commits ahead of main, show recent commits
		output, _ = r.Run(dir, "git", "log", "-5", "--oneline")
	}

	return output
}

// GetRecentCommit returns the most recent commit message with relative time.
func GetRecentCommit(dir string, r runner.CommandRunner) string {
	output, err := r.Run(dir, "git", "log", "-1", "--format=%s (%ar)")
	if err != nil {
		return ""
	}
	return output
}

// GetStashCount returns the number of stashed changes.
func GetStashCount(dir string, r runner.CommandRunner) int {
	stash, err := r.Run(dir, "git", "stash", "list")
	if err != nil || stash == "" {
		return 0
	}
	return CountLines(stash)
}

// RemoteStatus represents the sync status with the remote branch.
type RemoteStatus struct {
	Ahead  int
	Behind int
	Info   string // e.g., "ahead 2", "behind 3", "ahead 1, behind 2"
}

// CheckRemoteStatus checks if the branch is ahead/behind the remote.
// If fetch is true, fetches from remote first.
func CheckRemoteStatus(dir string, r runner.CommandRunner, fetch bool) RemoteStatus {
	if fetch {
		// Fetch with timeout to avoid hanging
		_, _ = r.RunWithTimeout(dir, 5*time.Second, "git", "fetch", "--quiet")
	}

	output, err := r.Run(dir, "git", "status", "-sb")
	if err != nil {
		return RemoteStatus{}
	}

	lines := strings.Split(output, "\n")
	if len(lines) == 0 {
		return RemoteStatus{}
	}

	firstLine := lines[0]
	if !strings.Contains(firstLine, "[") {
		return RemoteStatus{}
	}

	// Extract the tracking info
	re := regexp.MustCompile(`\[([^\]]+)\]`)
	matches := re.FindStringSubmatch(firstLine)
	if len(matches) <= 1 {
		return RemoteStatus{}
	}

	info := matches[1]
	status := RemoteStatus{Info: info}

	// Parse ahead/behind counts
	aheadRe := regexp.MustCompile(`ahead (\d+)`)
	behindRe := regexp.MustCompile(`behind (\d+)`)

	if m := aheadRe.FindStringSubmatch(info); len(m) > 1 {
		fmt.Sscanf(m[1], "%d", &status.Ahead)
	}
	if m := behindRe.FindStringSubmatch(info); len(m) > 1 {
		fmt.Sscanf(m[1], "%d", &status.Behind)
	}

	return status
}

// CountLines counts the number of non-empty lines in a string.
func CountLines(s string) int {
	if s == "" {
		return 0
	}
	return len(strings.Split(strings.TrimSpace(s), "\n"))
}
