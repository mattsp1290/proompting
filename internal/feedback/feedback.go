package feedback

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/vibes-project/vibes/internal/beads"
	"github.com/vibes-project/vibes/internal/git"
	"github.com/vibes-project/vibes/internal/runner"
)

// Options configures the feedback command behavior
type Options struct {
	Dir     string               // Target directory (defaults to cwd)
	Verbose bool                 // Include full protocol details
	Runner  runner.CommandRunner // Command runner (defaults to runner.Default)
}

// Run executes the feedback command and returns the prompt to stdout
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
	out.WriteString(fmt.Sprintf("# Act on Review Feedback in %s\n\n", projectName))

	// Get current branch and task context
	branch := git.GetCurrentBranch(dir, r)
	baseBranch := getBaseBranch(dir, r)
	task := beads.DetectCurrentTask(dir, branch, r)
	task.ProjectName = projectName

	// Context section
	out.WriteString("## Current Context\n")
	if branch != "" {
		out.WriteString(fmt.Sprintf("- **Branch**: %s\n", branch))
	}
	if task.ID != "" {
		if task.Title != "" {
			out.WriteString(fmt.Sprintf("- **Task**: %s \"%s\"\n", task.ID, task.Title))
		} else {
			out.WriteString(fmt.Sprintf("- **Task**: %s\n", task.ID))
		}
		out.WriteString(fmt.Sprintf("- **Review Thread**: %s-review\n", task.ID))
	}

	// Working tree status
	status := git.GetWorkingTreeStatus(dir, r)
	if status != "" {
		out.WriteString(fmt.Sprintf("- **Working tree**: %s\n", status))
	} else {
		out.WriteString("- **Working tree**: Clean\n")
	}
	out.WriteString("\n")

	// Recent commits on branch
	commits := git.GetBranchCommits(dir, branch, r)
	if commits != "" {
		out.WriteString("## Recent Commits\n")
		out.WriteString("```\n")
		out.WriteString(commits)
		out.WriteString("\n```\n\n")
	}

	// Changes since base branch
	diffStats := getDiffStats(dir, baseBranch, r)
	if diffStats != "" {
		out.WriteString("## Changes Summary\n")
		out.WriteString(fmt.Sprintf("- **Base**: %s\n", baseBranch))
		out.WriteString(fmt.Sprintf("- **Stats**: %s\n", diffStats))
		out.WriteString("\n")
	}

	// Inbox hint
	out.WriteString("## Check Review Feedback\n")
	out.WriteString(getInboxHint(task, opts.Verbose))
	out.WriteString("\n")

	// Protocol
	out.WriteString("## Protocol\n")
	out.WriteString(getProtocol(task, opts.Verbose))

	fmt.Print(out.String())
	return nil
}

// getBaseBranch determines the base branch (main or master)
func getBaseBranch(dir string, r runner.CommandRunner) string {
	// Check if main exists
	_, err := r.Run(dir, "git", "rev-parse", "--verify", "main")
	if err == nil {
		return "main"
	}

	// Check if master exists
	_, err = r.Run(dir, "git", "rev-parse", "--verify", "master")
	if err == nil {
		return "master"
	}

	// Default to main
	return "main"
}

// getDiffStats returns a summary of the diff (files changed, insertions, deletions)
func getDiffStats(dir string, baseBranch string, r runner.CommandRunner) string {
	output, err := r.Run(dir, "git", "diff", "--stat", baseBranch+"...HEAD")
	if err != nil || output == "" {
		return ""
	}

	// Get the last line which contains the summary
	lines := strings.Split(strings.TrimSpace(output), "\n")
	if len(lines) == 0 {
		return ""
	}

	summary := lines[len(lines)-1]
	return strings.TrimSpace(summary)
}

func getInboxHint(task beads.TaskInfo, verbose bool) string {
	threadID := "<task-id>-review"
	if task.ID != "" {
		threadID = task.ID + "-review"
	}

	projectKey := task.ProjectName
	if projectKey == "" {
		projectKey = "project-name"
	}

	if verbose {
		return fmt.Sprintf(`Check your inbox for review feedback:

`+"```"+`
# Check inbox for messages
resource://inbox/YourAgentIdentity

# Get messages from the review thread
get_thread_messages(
    project_key="%s",
    thread_id="%s"
)
`+"```"+`

Look for feedback categories:
- **Blocking**: Must fix before merge
- **Suggestion**: Should consider, discuss if disagree
- **Question**: Respond with clarification
- **Nitpick**: Optional style/preference
`, projectKey, threadID)
	}

	return fmt.Sprintf(`- Check inbox: `+"`resource://inbox/YourAgentIdentity`"+`
- Get thread: `+"`get_thread_messages(project_key, \"%s\")`"+`
`, threadID)
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
		return fmt.Sprintf(`1. **Retrieve review feedback** from the thread

2. **Triage feedback** by category:
   | Category | Action | Priority |
   |----------|--------|----------|
   | Blocking | Must fix before merge | Critical |
   | Suggestion | Consider, discuss if disagree | High |
   | Question | Respond with clarification | Medium |
   | Nitpick | Optional style fix | Low |

3. **Re-reserve files** if needed:
   `+"```"+`
   file_reservation_paths(
       project_key="%s",
       agent_name="YourAgentIdentity",
       patterns=["<your-file-patterns>"],
       ttl_seconds=3600,
       exclusive=true
   )
   `+"```"+`

4. **Address blocking issues first**, then suggestions

5. **Respond to questions** in the review thread

6. **Commit fixes** with descriptive messages:
   `+"```bash"+`
   git commit -m "fix: address review feedback

   - Fixed <blocking issue>
   - Improved <suggestion>

   Bead: %s"
   `+"```"+`

7. **Post resolution summary** to the review thread:
   `+"```"+`
   send_message(
       project_key="%s",
       from_agent="YourAgentIdentity",
       thread_id="%s-review",
       subject="Review Feedback Addressed",
       body="All items addressed. Ready for re-review."
   )
   `+"```"+`

8. **Request re-review** if changes were significant

9. **When approved**, continue to PR:
   `+"```bash"+`
   claude "$(vibes pr)"
   `+"```"+`

Address the review feedback now.
`, projectKey, taskID, projectKey, taskID)
	}

	return fmt.Sprintf(`1. Retrieve feedback from %s-review thread
2. Triage: blocking > suggestions > questions > nitpicks
3. Re-reserve files if needed
4. Fix blocking issues first
5. Commit: `+"`git commit -m \"fix: address review feedback\"`"+`
6. Post resolution summary to thread
7. Request re-review if significant changes
8. When approved: `+"`claude \"$(vibes pr)\"`"+`

Address the review feedback now.
`, taskID)
}
