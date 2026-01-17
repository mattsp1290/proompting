package pr

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/vibes-project/vibes/internal/beads"
	"github.com/vibes-project/vibes/internal/git"
	"github.com/vibes-project/vibes/internal/runner"
)

// PRInfo holds information about an existing pull request
type PRInfo struct {
	Number int    `json:"number"`
	Title  string `json:"title"`
	URL    string `json:"url"`
	State  string `json:"state"`
}

// Options configures the pr command behavior
type Options struct {
	Dir     string               // Target directory (defaults to cwd)
	Verbose bool                 // Include full protocol details
	Runner  runner.CommandRunner // Command runner (defaults to runner.Default)
}

// Run executes the pr command and returns the prompt to stdout
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

	projectName := filepath.Base(dir)

	// Get current branch and task context
	branch := git.GetCurrentBranch(dir, r)
	baseBranch := getBaseBranch(dir, r)
	task := beads.DetectCurrentTask(dir, branch, r)
	task.ProjectName = projectName

	// Check if we're on the base branch (early exit)
	if branch == baseBranch || branch == "main" || branch == "master" {
		out.WriteString(fmt.Sprintf("# Create Pull Request for %s\n\n", projectName))
		out.WriteString("## Branch Info\n")
		out.WriteString(fmt.Sprintf("- **Current**: %s\n", branch))
		out.WriteString(fmt.Sprintf("- **Base**: %s\n", baseBranch))
		out.WriteString("\n⚠️ You are on the base branch. Create a feature branch first:\n")
		out.WriteString("```bash\n")
		out.WriteString("git checkout -b feature/your-feature-name\n")
		out.WriteString("```\n")
		fmt.Print(out.String())
		return nil
	}

	// Check for existing PR
	existingPR := getExistingPR(dir, branch, r)

	// Header - changes based on whether PR exists
	if existingPR != nil {
		out.WriteString(fmt.Sprintf("# Pull Request #%d for %s\n\n", existingPR.Number, projectName))
		out.WriteString("## Existing PR\n")
		out.WriteString(fmt.Sprintf("- **PR**: #%d %s\n", existingPR.Number, existingPR.Title))
		out.WriteString(fmt.Sprintf("- **Status**: %s\n", existingPR.State))
		out.WriteString(fmt.Sprintf("- **URL**: %s\n", existingPR.URL))
		out.WriteString("\n")
	} else {
		out.WriteString(fmt.Sprintf("# Create Pull Request for %s\n\n", projectName))
	}

	// Branch info section
	out.WriteString("## Branch Info\n")
	if branch != "" {
		out.WriteString(fmt.Sprintf("- **Current**: %s\n", branch))
	}
	out.WriteString(fmt.Sprintf("- **Base**: %s\n", baseBranch))

	// Commits ahead
	commits := git.GetBranchCommits(dir, branch, r)
	if commits != "" {
		commitCount := git.CountLines(commits)
		out.WriteString(fmt.Sprintf("- **Commits**: %d ahead of %s\n", commitCount, baseBranch))
	}

	// Diff stats
	diffStats := getDiffStats(dir, baseBranch, r)
	if diffStats != "" {
		out.WriteString(fmt.Sprintf("- **Changes**: %s\n", diffStats))
	}

	// Working tree status
	status := git.GetWorkingTreeStatus(dir, r)
	if status != "" {
		out.WriteString(fmt.Sprintf("- **Working tree**: %s (uncommitted)\n", status))
	}
	out.WriteString("\n")

	// Task context section (if available)
	if task.ID != "" {
		out.WriteString("## Task Context\n")
		if task.Title != "" {
			out.WriteString(fmt.Sprintf("- **Bead**: %s \"%s\"\n", task.ID, task.Title))
		} else {
			out.WriteString(fmt.Sprintf("- **Bead**: %s\n", task.ID))
		}
		out.WriteString("\n")
	}

	// Commits section
	if commits != "" {
		out.WriteString("## Commits\n")
		out.WriteString("```\n")
		out.WriteString(commits)
		out.WriteString("\n```\n\n")
	}

	// Files changed section
	filesChanged := getFilesChanged(dir, baseBranch, r)
	if filesChanged != "" {
		out.WriteString("## Files Changed\n")
		out.WriteString("```\n")
		out.WriteString(filesChanged)
		out.WriteString("\n```\n\n")
	}

	// Protocol
	out.WriteString("## Protocol\n")
	if existingPR != nil {
		out.WriteString(getExistingPRProtocol(existingPR, opts.Verbose))
	} else {
		out.WriteString(getProtocol(task, baseBranch, opts.Verbose))
	}

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
	// Clean up the summary line
	summary = strings.TrimSpace(summary)
	return summary
}

// getFilesChanged returns a list of files changed compared to base branch
func getFilesChanged(dir string, baseBranch string, r runner.CommandRunner) string {
	output, err := r.Run(dir, "git", "diff", "--name-status", baseBranch+"...HEAD")
	if err != nil || output == "" {
		return ""
	}
	return strings.TrimSpace(output)
}

func getProtocol(task beads.TaskInfo, baseBranch string, verbose bool) string {
	taskContext := ""
	if task.ID != "" {
		if task.Title != "" {
			taskContext = fmt.Sprintf("\n   - Reference: %s \"%s\"", task.ID, task.Title)
		} else {
			taskContext = fmt.Sprintf("\n   - Reference: %s", task.ID)
		}
	}

	if verbose {
		return fmt.Sprintf(`1. **Review changes** for any issues:
   - Security vulnerabilities
   - Performance problems
   - Missing error handling
   - Code style consistency

2. **Check for uncommitted work**:
   `+"```bash"+`
   git status
   git diff
   `+"```"+`

3. **Create PR title and description**:
   - Title: concise summary (50 chars max)
   - Description: what changed and why%s

4. **Create the pull request**:
   `+"```bash"+`
   gh pr create --base %s --title "Your PR title" --body "$(cat <<'EOF'
## Summary
<bullet points of changes>

## Test plan
<how to verify the changes>

🤖 Generated with [Claude Code](https://claude.com/claude-code)
EOF
)"
   `+"```"+`

5. **Verify PR was created**:
   `+"```bash"+`
   gh pr view --web
   `+"```"+`

Please review the changes and create the pull request.
`, taskContext, baseBranch)
	}

	return fmt.Sprintf(`1. Review changes for issues (security, performance, style)
2. Check for uncommitted work: `+"`git status`"+`
3. Create PR with descriptive title and summary%s
4. Run: `+"`gh pr create --base %s`"+`

Please review the changes and create the pull request.
`, taskContext, baseBranch)
}

// getExistingPR checks if a PR already exists for the given branch
func getExistingPR(dir string, branch string, r runner.CommandRunner) *PRInfo {
	output, err := r.RunWithTimeout(dir, 10*time.Second, "gh", "pr", "list", "--head", branch, "--json", "number,title,url,state", "--limit", "1")
	if err != nil || output == "" {
		return nil
	}

	var prs []PRInfo
	if err := json.Unmarshal([]byte(output), &prs); err != nil {
		return nil
	}

	if len(prs) == 0 {
		return nil
	}

	return &prs[0]
}

// getExistingPRProtocol returns the protocol for an existing PR
func getExistingPRProtocol(pr *PRInfo, verbose bool) string {
	if verbose {
		return fmt.Sprintf(`A pull request already exists for this branch.

1. **Review the PR status**:
   `+"```bash"+`
   gh pr view %d
   gh pr checks %d
   `+"```"+`

2. **Check for review feedback**:
   `+"```bash"+`
   gh pr view %d --comments
   `+"```"+`

3. **If changes are needed**, commit and push:
   `+"```bash"+`
   git add -A && git commit -m "address review feedback"
   git push
   `+"```"+`

4. **View the PR in browser**:
   `+"```bash"+`
   gh pr view %d --web
   `+"```"+`

5. **When ready to merge**:
   `+"```bash"+`
   gh pr merge %d
   `+"```"+`

The PR is ready for review or updates.
`, pr.Number, pr.Number, pr.Number, pr.Number, pr.Number)
	}

	return fmt.Sprintf(`A pull request already exists for this branch.

1. View PR: `+"`gh pr view %d`"+`
2. Check status: `+"`gh pr checks %d`"+`
3. Push updates: `+"`git push`"+` (if changes made)
4. Open in browser: `+"`gh pr view %d --web`"+`

The PR is ready for review or updates.
`, pr.Number, pr.Number, pr.Number)
}
