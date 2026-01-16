package done

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/vibes-project/vibes/internal/beads"
	"github.com/vibes-project/vibes/internal/git"
	"github.com/vibes-project/vibes/internal/runner"
)

// Options configures the done command behavior
type Options struct {
	Dir     string               // Target directory (defaults to cwd)
	Verbose bool                 // Include full protocol details
	Runner  runner.CommandRunner // Command runner (defaults to runner.Default)
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
	branch := git.GetCurrentBranch(dir, r)
	task := beads.DetectCurrentTask(dir, branch, r)
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
	commits := git.GetBranchCommits(dir, branch, r)
	if commits != "" {
		out.WriteString(fmt.Sprintf("- **Commits on branch**: %d commits\n", git.CountLines(commits)))
	}

	// Working tree status
	status := git.GetWorkingTreeStatus(dir, r)
	if status != "" {
		out.WriteString(fmt.Sprintf("- **Working tree**: %s\n", status))
	} else {
		out.WriteString("- **Working tree**: Clean\n")
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
