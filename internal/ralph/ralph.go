package ralph

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

// Mode defines the operation mode for the Ralph loop.
type Mode int

const (
	// ModeSingleTask works on the next Beads task (default).
	ModeSingleTask Mode = iota
	// ModeGoal works toward a stated goal.
	ModeGoal
	// ModeAutopilot works through the entire task graph.
	ModeAutopilot
)

// Options configures the ralph command behavior.
type Options struct {
	Dir           string               // Target directory (defaults to cwd)
	Verbose       bool                 // Include full protocol details
	Mode          Mode                 // Operation mode
	Goal          string               // For ModeGoal: the goal to work toward
	MaxIterations int                  // Suggested iteration limit (0 = unlimited)
	Runner        runner.CommandRunner // Command runner (defaults to runner.Default)
}

// Run executes the ralph command and returns the prompt to stdout.
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
	out.WriteString(fmt.Sprintf("# Ralph Loop: %s\n\n", projectName))

	// Mode section
	out.WriteString(buildModeSection(opts))

	// Project context
	out.WriteString("## Project Context\n")
	out.WriteString(buildProjectContext(dir, r))
	out.WriteString("\n")

	// Current objective based on mode
	out.WriteString("## Current Objective\n")
	out.WriteString(buildTaskSection(dir, opts, r))
	out.WriteString("\n")

	// Completion requirements
	out.WriteString("## Completion Requirements (CRITICAL)\n")
	out.WriteString(buildCompletionRequirements(dir, opts.Verbose))
	out.WriteString("\n")

	// Checkpoint protocol
	out.WriteString("## Checkpoint Commits\n")
	out.WriteString(buildCheckpointProtocol(opts.Verbose))
	out.WriteString("\n")

	// Iteration protocol
	out.WriteString("## Iteration Protocol\n")
	out.WriteString(buildIterationProtocol(opts.Verbose))

	fmt.Print(out.String())
	return nil
}

func buildModeSection(opts Options) string {
	var mode string
	switch opts.Mode {
	case ModeGoal:
		mode = fmt.Sprintf("Goal: \"%s\"", opts.Goal)
	case ModeAutopilot:
		mode = "Autopilot"
	default:
		mode = "Single Task"
	}

	var out strings.Builder
	out.WriteString(fmt.Sprintf("## Mode: %s\n", mode))
	if opts.MaxIterations > 0 {
		out.WriteString(fmt.Sprintf("- Max iterations: %d [suggested limit]\n", opts.MaxIterations))
	}
	out.WriteString("\n")
	return out.String()
}

func buildProjectContext(dir string, r runner.CommandRunner) string {
	var out strings.Builder

	// Current branch
	branch := git.GetCurrentBranch(dir, r)
	if branch != "" {
		out.WriteString(fmt.Sprintf("- Branch: %s\n", branch))
	}

	// Status summary
	status := git.GetWorkingTreeStatus(dir, r)
	if status == "" {
		out.WriteString("- Status: Clean working tree\n")
	} else {
		out.WriteString(fmt.Sprintf("- Status: %s\n", status))
	}

	// Recent commit - sanitize to avoid shell-special characters
	recentCommit := git.GetRecentCommit(dir, r)
	if recentCommit != "" {
		// Remove parentheses which can cause shell issues
		sanitized := sanitizeForShell(recentCommit)
		out.WriteString(fmt.Sprintf("- Recent: %s\n", sanitized))
	}

	return out.String()
}

// sanitizeForShell removes or escapes characters that cause shell parsing issues.
func sanitizeForShell(s string) string {
	// Replace parentheses with brackets to avoid shell interpretation
	s = strings.ReplaceAll(s, "(", "[")
	s = strings.ReplaceAll(s, ")", "]")
	// Remove backticks which can cause command substitution
	s = strings.ReplaceAll(s, "`", "'")
	// Remove dollar signs which can cause variable expansion
	s = strings.ReplaceAll(s, "$", "")
	return s
}

func buildTaskSection(dir string, opts Options, r runner.CommandRunner) string {
	switch opts.Mode {
	case ModeGoal:
		return buildGoalSection(opts.Goal)
	case ModeAutopilot:
		return buildAutopilotSection(dir, r)
	default:
		return buildSingleTaskSection(dir, r)
	}
}

func buildSingleTaskSection(dir string, r runner.CommandRunner) string {
	// Check if beads is initialized
	if !beads.IsInitialized(dir) {
		return "No beads task graph found. Work on immediate project needs or run `bd init` to initialize Beads.\n"
	}

	// Try bv --robot-triage first (more intelligent recommendations)
	if output, err := r.RunWithTimeout(dir, 10*time.Second, "bv", "--robot-triage"); err == nil && output != "" {
		return output + "\n\nFocus on completing the highest priority task above.\n"
	}

	// Fall back to bd ready
	if output, err := r.RunWithTimeout(dir, 10*time.Second, "bd", "ready"); err == nil && output != "" {
		return output + "\n\nSelect and complete the most appropriate task from above.\n"
	}

	return "Beads initialized but no ready tasks found. Work on immediate project needs or create tasks with `bd create \"Task name\" -p 1`.\n"
}

func buildGoalSection(goal string) string {
	var out strings.Builder
	out.WriteString(fmt.Sprintf("Goal: %s\n\n", goal))
	out.WriteString("Work iteratively toward this goal. Each iteration should make concrete progress.\n")
	out.WriteString("Break down the goal into logical steps and execute them one at a time.\n")
	return out.String()
}

func buildAutopilotSection(dir string, r runner.CommandRunner) string {
	var out strings.Builder

	// Check if beads is initialized
	if !beads.IsInitialized(dir) {
		out.WriteString("No beads task graph found. Run 'bd init' to initialize Beads for autopilot mode.\n")
		return out.String()
	}

	out.WriteString("Work through the entire task graph autonomously.\n\n")

	// Get task graph overview
	if output, err := r.RunWithTimeout(dir, 10*time.Second, "bv", "--robot-triage"); err == nil && output != "" {
		out.WriteString("### Task Overview\n")
		out.WriteString(output)
		out.WriteString("\n\n")
	} else if output, err := r.RunWithTimeout(dir, 10*time.Second, "bd", "ready"); err == nil && output != "" {
		out.WriteString("### Ready Tasks\n")
		out.WriteString(output)
		out.WriteString("\n\n")
	}

	out.WriteString("Process tasks in priority order. After completing each task:\n")
	out.WriteString("1. Mark it closed: `bd update <id> --status closed`\n")
	out.WriteString("2. Check for newly unblocked tasks\n")
	out.WriteString("3. Continue with the next highest priority task\n")
	return out.String()
}

func buildCompletionRequirements(dir string, verbose bool) string {
	var out strings.Builder

	testCmd := detectTestCommand(dir)

	out.WriteString("Both conditions must be met for completion:\n\n")

	out.WriteString("1. Verification signals must pass:\n")
	out.WriteString("   " + testCmd + "\n\n")

	out.WriteString("2. Explicit completion promise:\n")
	out.WriteString("   When the objective is fully complete, output: <promise>COMPLETE</promise>\n")

	if verbose {
		out.WriteString("\nCompletion Criteria Details:\n")
		out.WriteString("- Tests must pass [exit code 0]\n")
		out.WriteString("- Build must succeed [if applicable]\n")
		out.WriteString("- The <promise>COMPLETE</promise> tag signals you are confident the work is done\n")
		out.WriteString("- Do NOT output the promise tag until tests/build pass\n")
		out.WriteString("- Do NOT output the promise tag if there is more work to do\n")
	}

	return out.String()
}

// detectTestCommand auto-detects the appropriate test/build commands for the project.
func detectTestCommand(dir string) string {
	// Check for Go projects
	if fileExists(filepath.Join(dir, "go.mod")) {
		return "go test ./... && go build ./..."
	}

	// Check for Node.js projects
	if fileExists(filepath.Join(dir, "package.json")) {
		// Check for yarn
		if fileExists(filepath.Join(dir, "yarn.lock")) {
			return "yarn test"
		}
		// Check for pnpm
		if fileExists(filepath.Join(dir, "pnpm-lock.yaml")) {
			return "pnpm test"
		}
		return "npm test"
	}

	// Check for Python projects
	if fileExists(filepath.Join(dir, "pyproject.toml")) {
		return "pytest"
	}
	if fileExists(filepath.Join(dir, "setup.py")) {
		return "pytest"
	}

	// Check for Rust projects
	if fileExists(filepath.Join(dir, "Cargo.toml")) {
		return "cargo test && cargo build"
	}

	// Check for Make projects
	if fileExists(filepath.Join(dir, "Makefile")) {
		return "make test"
	}

	// Default: just verify build artifacts or skip
	return "# No test runner detected - verify manually or add tests"
}

func buildCheckpointProtocol(verbose bool) string {
	var out strings.Builder

	out.WriteString("After each successful iteration [tests pass], create a checkpoint commit:\n")
	out.WriteString("   git add -A && git commit -m \"ralph: iteration N - [brief summary]\"\n")

	if verbose {
		out.WriteString("\nCommit Guidelines:\n")
		out.WriteString("- Replace N with the iteration number [1, 2, 3, ...]\n")
		out.WriteString("- Keep summary brief [under 50 chars]\n")
		out.WriteString("- Examples:\n")
		out.WriteString("  - ralph: iteration 1 - add user model\n")
		out.WriteString("  - ralph: iteration 2 - implement auth endpoint\n")
		out.WriteString("  - ralph: iteration 3 - fix validation bug\n")
		out.WriteString("- Only commit when tests pass\n")
		out.WriteString("- Each commit should represent a stable, working state\n")
	}

	return out.String()
}

func buildIterationProtocol(verbose bool) string {
	if verbose {
		return `Each iteration follows this cycle:

1. ASSESS current state
   - Review previous iteration results
   - Check test status
   - Identify what needs to be done next

2. EXECUTE one increment
   - Make focused, incremental changes
   - Keep changes small and testable
   - Do not try to do too much at once

3. VERIFY the changes
   - Run tests/build commands
   - Check for errors or regressions
   - Fix any issues before proceeding

4. CHECKPOINT [if tests pass]
   - Commit changes with iteration summary
   - This creates a stable restore point

5. EVALUATE completion
   - Is the objective fully achieved?
   - If yes: output <promise>COMPLETE</promise>
   - If no: continue to next iteration

Important: Do not skip steps. Each iteration must verify before checkpointing.
`
	}

	return `1. ASSESS - Review current state and what is needed next
2. EXECUTE - Make one focused, incremental change
3. VERIFY - Run tests/build to confirm changes work
4. CHECKPOINT - Commit if tests pass
5. EVALUATE - Output <promise>COMPLETE</promise> when done, else continue

Begin working now.
`
}

// fileExists checks if a file exists at the given path.
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
