package done

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/vibes-project/vibes/internal/runner"
)

// Options configures the done command behavior
type Options struct {
	Dir     string               // Target directory (defaults to cwd)
	Verbose bool                 // Include full protocol details
	Runner  runner.CommandRunner // Command runner (defaults to runner.Default)
}

// TaskInfo holds information about the current task
type TaskInfo struct {
	ID          string
	Title       string
	Branch      string
	ProjectName string
}

// Run executes the done command and returns the prompt to stdout
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
	out.WriteString(fmt.Sprintf("# Complete Current Work in %s\n\n", projectName))

	// Get current branch and work summary
	branch := getCurrentBranch(dir, r)
	task := detectCurrentTask(dir, branch, r)
	task.ProjectName = projectName

	out.WriteString("## Work Summary\n")
	if branch != "" {
		out.WriteString(fmt.Sprintf("- **Branch**: %s\n", branch))
	}
	if task.ID != "" {
		if task.Title != "" {
			out.WriteString(fmt.Sprintf("- **Task**: %s \"%s\"\n", task.ID, task.Title))
		} else {
			out.WriteString(fmt.Sprintf("- **Task**: %s\n", task.ID))
		}
	}

	// Commits on this branch
	commits := getBranchCommits(dir, branch, r)
	if commits != "" {
		out.WriteString(fmt.Sprintf("- **Commits on branch**: %d commits\n", countLines(commits)))
	}

	// Working tree status
	status := getWorkingTreeStatus(dir, r)
	if status != "" {
		out.WriteString(fmt.Sprintf("- **Working tree**: %s\n", status))
	}
	out.WriteString("\n")

	// Recent commits section
	if commits != "" {
		out.WriteString("## Recent Commits\n")
		out.WriteString("```\n")
		out.WriteString(commits)
		out.WriteString("\n```\n\n")
	}

	// Protocol
	out.WriteString("## Completion Protocol\n")
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
		// Format is typically: "bd-XXXX  Title here  [in_progress]"
		lines := strings.Split(output, "\n")
		for _, line := range lines {
			if id, title := parseBeadLine(line); id != "" {
				task.ID = id
				task.Title = title
				return task
			}
		}
	}

	// Fallback: try to extract bead ID from branch name
	if beadID := extractBeadIDFromBranch(branch); beadID != "" {
		task.ID = beadID
		// Try to get the title
		if output, err := r.RunWithTimeout(dir, 5*time.Second, "bd", "show", beadID); err == nil {
			task.Title = extractTitleFromShow(output)
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
	// Look for a title line in bd show output
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "Title:") {
			return strings.TrimSpace(strings.TrimPrefix(line, "Title:"))
		}
		// Also try without prefix - first non-empty line after ID might be title
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
	// Try main first, then master
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

func getWorkingTreeStatus(dir string, r runner.CommandRunner) string {
	status, err := r.Run(dir, "git", "status", "--porcelain")
	if err != nil {
		return ""
	}

	if status == "" {
		return "Clean"
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
	return "Clean"
}

func countLines(s string) int {
	if s == "" {
		return 0
	}
	return len(strings.Split(strings.TrimSpace(s), "\n"))
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
		return fmt.Sprintf(`1. **Verify work is complete**
   - All tests pass
   - Code is committed (or commit now)
   - Changes are ready for review

2. **Release file reservations** (if using MCP Agent Mail):
   `+"```"+`
   release_file_paths(
       project_key="%s",
       agent_name="YourAgentIdentity"
   )
   `+"```"+`

3. **Mark task complete**:
   `+"```bash"+`
   bd update %s --status closed
   `+"```"+`

4. **Check for unblocked tasks**:
   `+"```bash"+`
   bd ready
   `+"```"+`

5. **Continue to next task** (optional):
   `+"```bash"+`
   claude "$(vibes next)"
   `+"```"+`

Please complete the current work following this protocol.
`, projectKey, taskID)
	}

	return fmt.Sprintf(`1. Verify: Tests pass, code committed
2. Release file reservations (if applicable)
3. Complete: `+"`bd update %s --status closed`"+`
4. Check unblocked: `+"`bd ready`"+`
5. Continue: `+"`claude \"$(vibes next)\"`"+`

Please complete the current work following this protocol.
`, taskID)
}
