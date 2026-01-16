package resume

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/vibes-project/vibes/internal/beads"
	"github.com/vibes-project/vibes/internal/git"
	"github.com/vibes-project/vibes/internal/runner"
)

// Options configures the resume command behavior
type Options struct {
	Dir     string               // Target directory (defaults to cwd)
	Verbose bool                 // Include full protocol details
	NoFetch bool                 // Skip fetching from remote
	Runner  runner.CommandRunner // Command runner (defaults to runner.Default)
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
	branch := git.GetCurrentBranch(dir, r)
	task := beads.DetectCurrentTask(dir, branch, r)
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
	uncommitted := git.GetWorkingTreeStatus(dir, r)
	if uncommitted != "" {
		out.WriteString(fmt.Sprintf("- **Uncommitted changes**: %s\n", uncommitted))
	} else {
		out.WriteString("- **Uncommitted changes**: None (working tree clean)\n")
	}

	// Recent commits on branch
	commits := git.GetBranchCommits(dir, branch, r)
	if commits != "" {
		commitCount := git.CountLines(commits)
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
	pendingItems := getPendingItems(dir, task, r, !opts.NoFetch)
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

func getPendingItems(dir string, task beads.TaskInfo, r runner.CommandRunner, fetch bool) []string {
	var items []string

	// Check for stashed changes
	stashCount := git.GetStashCount(dir, r)
	if stashCount > 0 {
		items = append(items, fmt.Sprintf("⚠️ %d stashed change(s) - consider applying or dropping", stashCount))
	}

	// Check if branch is behind remote
	remoteStatus := git.CheckRemoteStatus(dir, r, fetch)
	if remoteStatus.Behind > 0 {
		items = append(items, fmt.Sprintf("⚠️ Branch is %s - consider pulling", remoteStatus.Info))
	} else if remoteStatus.Ahead > 0 {
		items = append(items, fmt.Sprintf("📤 Branch is %s - remember to push", remoteStatus.Info))
	}

	// Hint about checking inbox if task has a review thread
	if task.ID != "" {
		items = append(items, fmt.Sprintf("💬 Check inbox for messages in %s-review thread", task.ID))
	}

	return items
}

func getProtocol(task beads.TaskInfo, verbose bool) string {
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
       patterns=["<your-file-patterns>"],
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

	return `1. Check inbox for pending messages or review feedback
2. Verify file reservations are still valid
3. Pull if behind remote: ` + "`git pull`" + `
4. Continue implementation from current state
5. When complete: ` + "`claude \"$(vibes done)\"`" + `

Continue working on the current task.
`
}
