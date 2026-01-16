// Package beads provides shared bead parsing operations for vibes commands.
package beads

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/vibes-project/vibes/internal/runner"
)

// TaskInfo holds information about a bead task.
type TaskInfo struct {
	ID          string
	Title       string
	Status      string
	Branch      string
	ProjectName string
}

// IsInitialized checks if beads is initialized in the given directory.
func IsInitialized(dir string) bool {
	beadsDir := filepath.Join(dir, ".beads")
	_, err := os.Stat(beadsDir)
	return err == nil
}

// ExtractIDFromBranch extracts a bead ID from a branch name.
// Matches patterns like: feature/bd-123-description, bd-456, BEAD-789
func ExtractIDFromBranch(branch string) string {
	patterns := []string{
		`(bd-\d+)`,
		`(BEAD-\d+)`,
		`(bead-\d+)`,
	}

	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		if matches := re.FindStringSubmatch(branch); len(matches) > 1 {
			return matches[1]
		}
	}
	return ""
}

// ParseListLine parses a line from `bd list` output.
// Format: "bd-123  Some task title  [status]"
func ParseListLine(line string) (id, title string) {
	line = strings.TrimSpace(line)
	if line == "" {
		return "", ""
	}

	// Look for bd-XXXX pattern at the start
	re := regexp.MustCompile(`^(bd-\d+)\s+(.+?)(?:\s+\[.+\])?$`)
	if matches := re.FindStringSubmatch(line); len(matches) >= 3 {
		return matches[1], strings.TrimSpace(matches[2])
	}

	// Simpler fallback: just get the ID
	reSimple := regexp.MustCompile(`^(bd-\d+)`)
	if matches := reSimple.FindStringSubmatch(line); len(matches) > 1 {
		return matches[1], ""
	}

	return "", ""
}

// ExtractTitleFromShow extracts the title from `bd show` output.
func ExtractTitleFromShow(output string) string {
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "Title:") {
			return strings.TrimSpace(strings.TrimPrefix(line, "Title:"))
		}
	}
	return ""
}

// ExtractStatusFromShow extracts the status from `bd show` output.
func ExtractStatusFromShow(output string) string {
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "Status:") {
			return strings.TrimSpace(strings.TrimPrefix(line, "Status:"))
		}
	}
	return ""
}

// DetectCurrentTask attempts to detect the current task from beads or branch name.
func DetectCurrentTask(dir string, branch string, r runner.CommandRunner) TaskInfo {
	task := TaskInfo{Branch: branch}

	if !IsInitialized(dir) {
		// Try to extract from branch name as fallback
		task.ID = ExtractIDFromBranch(branch)
		return task
	}

	// Try to find in-progress tasks
	output, err := r.RunWithTimeout(dir, 5*time.Second, "bd", "list", "--status", "in_progress")
	if err == nil && output != "" {
		// Parse first in-progress task
		lines := strings.Split(output, "\n")
		for _, line := range lines {
			if id, title := ParseListLine(line); id != "" {
				task.ID = id
				task.Title = title
				task.Status = "in_progress"
				return task
			}
		}
	}

	// Fallback: try to extract bead ID from branch name
	if beadID := ExtractIDFromBranch(branch); beadID != "" {
		task.ID = beadID
		// Try to get the title and status
		if output, err := r.RunWithTimeout(dir, 5*time.Second, "bd", "show", beadID); err == nil {
			task.Title = ExtractTitleFromShow(output)
			task.Status = ExtractStatusFromShow(output)
		}
	}

	return task
}
