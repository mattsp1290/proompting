package resume

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/vibes-project/vibes/internal/runner"
)

// Options configures the resume command behavior
type Options struct {
	Dir     string               // Target directory (defaults to cwd)
	Verbose bool                 // Include full protocol details
	Runner  runner.CommandRunner // Command runner (defaults to runner.Default)
}

// TaskInfo holds information about the current task
type TaskInfo struct {
	ID          string
	Title       string
	Status      string
	Branch      string
	ProjectName string
}

// Run executes the resume command and returns the prompt to stdout
func Run(opts Options) error {
	dir := opts.Dir
	if dir == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("getting current directory: %w", err)
		}
		dir = cwd
	}

	r := opts.Runner
	if r == nil {
		r = &runner.Default{}
	}

	var out strings.Builder

	// Header
	projectName := filepath.Base(dir)
	out.WriteString(fmt.Sprintf("# Resume Work in %s\n\n", projectName))

	// Get current branch and task context
	branch := getCurrentBranch(dir, r)
	task := detectCurrentTask(dir, branch, r)
	task.ProjectName = projectName

	// Current work section
	out.WriteString("## Current Work\n")
	if branch != "" {
		out.WriteString(fmt.Sprintf("- **Branch**: %s\n", branch))
	}
	if task.ID != "" {
		if task.Title != "" {
			statusStr := ""
			if task.Status != "" {
				statusStr = fmt.Sprintf(" [%s]", task.Status)
			}
			out.WriteString(fmt.Sprintf("- **Task**: %s \"%s\"%s\n", task.ID, task.Title, statusStr))
		} else {
			out.WriteString(fmt.Sprintf("- **Task**: %s\n", task.ID))
		}
	}
	out.WriteString("\n")

	// Work in progress section
	out.WriteString("## Work in Progress\n")

	// Uncommitted changes
	uncommitted := getUncommittedChanges(dir, r)
	if uncommitted != "" {
		out.WriteString(fmt.Sprintf("- **Uncommitted changes**: %s\n", uncommitted))
	} else {
		out.WriteString("- **Uncommitted changes**: None (working tree clean)\n")
	}

	// Recent commits on branch
	commits := getBranchCommits(dir, branch, r)
	if commits != "" {
		commitCount := countLines(commits)
		out.WriteString(fmt.Sprintf("- **Commits on branch**: %d\n", commitCount))
	}
	out.WriteString("\n")

	// Show recent commits
	if commits != "" {
		out.WriteString("## Recent Commits\n")
		out.WriteString("```\n")
		out.WriteString(commits)
		out.WriteString("\n```\n\n")
	}

	// Pending attention section
	pendingItems := getPendingItems(dir, task, r)
	if len(pendingItems) > 0 {
		out.WriteString("## Pending Attention\n")
		for _, item := range pendingItems {
			out.WriteString(fmt.Sprintf("- %s\n", item))
		}
		out.WriteString("\n")
	}

	// Protocol
	out.WriteString("## Protocol\n")
	out.WriteString(getProtocol(task, opts.Verbose))

	fmt.Print(out.String())
	return nil
}

func getCurrentBranch(dir string, r runner.CommandRunner) string {
	branch, err := r.Run(dir, "git", "rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		return ""
	}
	return branch
}

func detectCurrentTask(dir string, branch string, r runner.CommandRunner) TaskInfo {
	task := TaskInfo{Branch: branch}

	// Check if beads is initialized
	beadsDir := filepath.Join(dir, ".beads")
	if _, err := os.Stat(beadsDir); os.IsNotExist(err) {
		// Try to extract from branch name as fallback
		task.ID = extractBeadIDFromBranch(branch)
		return task
	}

	// Try to find in-progress tasks
	output, err := r.RunWithTimeout(dir, 5*time.Second, "bd", "list", "--status", "in_progress")
	if err == nil && output != "" {
		// Parse first in-progress task
		lines := strings.Split(output, "\n")
		for _, line := range lines {
			if id, title := parseBeadLine(line); id != "" {
				task.ID = id
				task.Title = title
				task.Status = "in_progress"
				return task
			}
		}
	}

	// Fallback: try to extract bead ID from branch name
	if beadID := extractBeadIDFromBranch(branch); beadID != "" {
		task.ID = beadID
		// Try to get the title and status
		if output, err := r.RunWithTimeout(dir, 5*time.Second, "bd", "show", beadID); err == nil {
			task.Title = extractTitleFromShow(output)
			task.Status = extractStatusFromShow(output)
		}
	}

	return task
}

func extractBeadIDFromBranch(branch string) string {
	// Match patterns like: feature/bd-123-description, bd-456, BEAD-789
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

func parseBeadLine(line string) (id, title string) {
	// Parse format like: "bd-123  Some task title  [status]"
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

func extractTitleFromShow(output string) string {
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "Title:") {
			return strings.TrimSpace(strings.TrimPrefix(line, "Title:"))
		}
	}
	return ""
}

func extractStatusFromShow(output string) string {
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "Status:") {
			return strings.TrimSpace(strings.TrimPrefix(line, "Status:"))
		}
	}
	return ""
}

func getUncommittedChanges(dir string, r runner.CommandRunner) string {
	status, err := r.Run(dir, "git", "status", "--porcelain")
	if err != nil {
		return ""
	}

	if status == "" {
		return ""
	}

	lines := strings.Split(strings.TrimSpace(status), "\n")
	modified := 0
	untracked := 0
	staged := 0

	for _, line := range lines {
		if len(line) < 2 {
			continue
		}
		index := line[0]
		worktree := line[1]
		if index == '?' {
			untracked++
		} else if index != ' ' {
			staged++
		}
		if worktree != ' ' && worktree != '?' {
			modified++
		}
	}

	parts := []string{}
	if staged > 0 {
		parts = append(parts, fmt.Sprintf("%d staged", staged))
	}
	if modified > 0 {
		parts = append(parts, fmt.Sprintf("%d modified", modified))
	}
	if untracked > 0 {
		parts = append(parts, fmt.Sprintf("%d untracked", untracked))
	}

	if len(parts) > 0 {
		return strings.Join(parts, ", ")
	}
	return ""
}

func getBranchCommits(dir string, branch string, r runner.CommandRunner) string {
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

func countLines(s string) int {
	if s == "" {
		return 0
	}
	return len(strings.Split(strings.TrimSpace(s), "\n"))
}

func getPendingItems(dir string, task TaskInfo, r runner.CommandRunner) []string {
	var items []string

	// Check for stashed changes
	stash, err := r.Run(dir, "git", "stash", "list")
	if err == nil && stash != "" {
		stashCount := countLines(stash)
		items = append(items, fmt.Sprintf("⚠️ %d stashed change(s) - consider applying or dropping", stashCount))
	}

	// Check if branch is behind remote
	behindAhead := checkRemoteStatus(dir, r)
	if behindAhead != "" {
		items = append(items, behindAhead)
	}

	// Hint about checking inbox if task has a review thread
	if task.ID != "" {
		items = append(items, fmt.Sprintf("💬 Check inbox for messages in %s-review thread", task.ID))
	}

	return items
}

func checkRemoteStatus(dir string, r runner.CommandRunner) string {
	// Fetch first to get latest remote state (with timeout to avoid hanging)
	_, _ = r.RunWithTimeout(dir, 5*time.Second, "git", "fetch", "--quiet")

	// Check status relative to upstream
	output, err := r.Run(dir, "git", "status", "-sb")
	if err != nil {
		return ""
	}

	// Look for [ahead N] or [behind N] in the first line
	lines := strings.Split(output, "\n")
	if len(lines) == 0 {
		return ""
	}

	firstLine := lines[0]
	if strings.Contains(firstLine, "[") {
		// Extract the tracking info
		re := regexp.MustCompile(`\[([^\]]+)\]`)
		if matches := re.FindStringSubmatch(firstLine); len(matches) > 1 {
			info := matches[1]
			if strings.Contains(info, "behind") {
				return fmt.Sprintf("⚠️ Branch is %s - consider pulling", info)
			}
			if strings.Contains(info, "ahead") {
				return fmt.Sprintf("📤 Branch is %s - remember to push", info)
			}
		}
	}

	return ""
}

func getProtocol(task TaskInfo, verbose bool) string {
	taskID := task.ID
	if taskID == "" {
		taskID = "<task-id>"
	}

	projectKey := task.ProjectName
	if projectKey == "" {
		projectKey = "project-name"
	}

	if verbose {
		return fmt.Sprintf(`1. **Check for updates**
   - Review any pending messages in your inbox
   - Check if file reservations are still valid
   - Pull latest changes if behind remote

2. **Verify current state**:
   `+"```bash"+`
   git status
   bd show %s
   `+"```"+`

3. **Re-reserve files if needed** (via MCP Agent Mail):
   `+"```"+`
   file_reservation_paths(
       project_key="%s",
       agent_name="YourAgentIdentity",
       patterns=["src/path/**"],
       ttl_seconds=3600,
       exclusive=true
   )
   `+"```"+`

4. **Continue implementation** from current state

5. **When complete**:
   `+"```bash"+`
   claude "$(vibes done)"
   `+"```"+`

Continue working on the current task.
`, taskID, projectKey)
	}

	return fmt.Sprintf(`1. Check inbox for pending messages or review feedback
2. Verify file reservations are still valid
3. Pull if behind remote: `+"`git pull`"+`
4. Continue implementation from current state
5. When complete: `+"`claude \"$(vibes done)\"`"+`

Continue working on the current task.
`)
}
