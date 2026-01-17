package stuck

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/vibes-project/vibes/internal/beads"
	"github.com/vibes-project/vibes/internal/git"
	"github.com/vibes-project/vibes/internal/runner"
)

// Options configures the stuck command behavior
type Options struct {
	Dir         string               // Target directory (defaults to cwd)
	Verbose     bool                 // Include full protocol details
	Description string               // Optional problem description from user
	Runner      runner.CommandRunner // Command runner (defaults to runner.Default)
}

// Run executes the stuck command and returns the prompt to stdout
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
	out.WriteString(fmt.Sprintf("# Help Debugging in %s\n\n", projectName))

	// Get current branch and task context
	branch := git.GetCurrentBranch(dir, r)
	task := beads.DetectCurrentTask(dir, branch, r)
	task.ProjectName = projectName

	// Current context section
	out.WriteString("## Current Context\n")
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

	// Working tree status
	status := git.GetWorkingTreeStatus(dir, r)
	if status != "" {
		out.WriteString(fmt.Sprintf("- **Working tree**: %s\n", status))
	} else {
		out.WriteString("- **Working tree**: Clean\n")
	}
	out.WriteString("\n")

	// Recent changes section
	diff := getDiff(dir, r)
	if diff != "" {
		out.WriteString("## Recent Changes\n")
		out.WriteString("```diff\n")
		out.WriteString(truncateOutput(diff, 100))
		out.WriteString("\n```\n\n")
	}

	// Recent commits
	commits := git.GetBranchCommits(dir, branch, r)
	if commits != "" {
		out.WriteString("## Recent Commits\n")
		out.WriteString("```\n")
		out.WriteString(commits)
		out.WriteString("\n```\n\n")
	}

	// Try to detect errors
	errorOutput := detectErrors(dir, r)
	if errorOutput != "" {
		out.WriteString("## Detected Errors\n")
		out.WriteString("```\n")
		out.WriteString(truncateOutput(errorOutput, 50))
		out.WriteString("\n```\n\n")
	}

	// Problem description
	if opts.Description != "" {
		out.WriteString("## Problem\n")
		out.WriteString(fmt.Sprintf("%s\n\n", opts.Description))
	}

	// Protocol
	out.WriteString("## Debugging Protocol\n")
	out.WriteString(getProtocol(opts.Verbose))

	fmt.Print(out.String())
	return nil
}

// getDiff returns the combined staged and unstaged diff, limited to recent changes
func getDiff(dir string, r runner.CommandRunner) string {
	// Get staged diff
	staged, _ := r.Run(dir, "git", "diff", "--cached", "--stat")

	// Get unstaged diff
	unstaged, _ := r.Run(dir, "git", "diff", "--stat")

	// Get actual diff content (limited)
	diffContent, _ := r.Run(dir, "git", "diff", "HEAD")

	var parts []string
	if staged != "" {
		parts = append(parts, "Staged:\n"+staged)
	}
	if unstaged != "" {
		parts = append(parts, "Unstaged:\n"+unstaged)
	}
	if diffContent != "" {
		parts = append(parts, diffContent)
	}

	return strings.Join(parts, "\n\n")
}

// detectErrors attempts to find recent errors by running common test/build commands
func detectErrors(dir string, r runner.CommandRunner) string {
	var errors []string

	// Check for Go projects
	if fileExists(filepath.Join(dir, "go.mod")) {
		// Try go build
		output, err := r.RunWithTimeout(dir, 30*time.Second, "go", "build", "./...")
		if err != nil && output != "" {
			errors = append(errors, "Go build errors:\n"+output)
		}

		// Try go vet
		output, err = r.RunWithTimeout(dir, 30*time.Second, "go", "vet", "./...")
		if err != nil && output != "" {
			errors = append(errors, "Go vet issues:\n"+output)
		}
	}

	// Check for Node.js projects
	if fileExists(filepath.Join(dir, "package.json")) {
		// Check for TypeScript errors
		if fileExists(filepath.Join(dir, "tsconfig.json")) {
			output, err := r.RunWithTimeout(dir, 30*time.Second, "npx", "tsc", "--noEmit")
			if err != nil && output != "" {
				errors = append(errors, "TypeScript errors:\n"+output)
			}
		}
	}

	// Check for Python projects
	if fileExists(filepath.Join(dir, "pyproject.toml")) || fileExists(filepath.Join(dir, "setup.py")) {
		// Try python syntax check on changed files
		changedPy, _ := r.Run(dir, "git", "diff", "--name-only", "--diff-filter=M", "*.py")
		if changedPy != "" {
			for _, f := range strings.Split(changedPy, "\n") {
				if f != "" {
					output, err := r.RunWithTimeout(dir, 10*time.Second, "python", "-m", "py_compile", f)
					if err != nil && output != "" {
						errors = append(errors, "Python syntax error in "+f+":\n"+output)
					}
				}
			}
		}
	}

	return strings.Join(errors, "\n\n")
}

// fileExists checks if a file exists
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// truncateOutput limits output to a certain number of lines
func truncateOutput(s string, maxLines int) string {
	lines := strings.Split(s, "\n")
	if len(lines) <= maxLines {
		return s
	}
	return strings.Join(lines[:maxLines], "\n") + fmt.Sprintf("\n... (%d more lines)", len(lines)-maxLines)
}

func getProtocol(verbose bool) string {
	if verbose {
		return `1. **Analyze the situation**
   - Review the recent changes shown above
   - Examine any detected errors
   - Understand what was being attempted

2. **Diagnose the root cause**
   - Identify where the problem originates
   - Check for common issues:
     - Typos or syntax errors
     - Missing imports or dependencies
     - Logic errors in conditionals
     - Incorrect function signatures
     - Race conditions or timing issues

3. **Investigate further if needed**
   - Read relevant source files
   - Check test files for expected behavior
   - Look at similar working code for patterns

4. **Propose a fix**
   - Explain what's wrong
   - Show the specific code change needed
   - Explain why the fix works

5. **Verify the fix**
   - Run relevant tests
   - Check for any new issues introduced

6. **If still stuck**
   - Ask clarifying questions
   - Suggest alternative approaches
   - Recommend additional debugging steps

Please help diagnose and fix the issue.
`
	}

	return `1. Analyze the recent changes and any detected errors
2. Diagnose the root cause of the problem
3. Investigate relevant files if needed
4. Propose a specific fix with explanation
5. Verify the fix works (run tests if applicable)

Please help diagnose and fix the issue.
`
}
