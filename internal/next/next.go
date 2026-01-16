package next

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

// Options configures the next command behavior
type Options struct {
	Dir     string               // Target directory (defaults to cwd)
	Verbose bool                 // Include full protocol details
	Runner  runner.CommandRunner // Command runner (defaults to runner.Default)
}

// Run executes the next command and returns the prompt to stdout
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
	out.WriteString(fmt.Sprintf("# Next Task for %s\n\n", projectName))

	// Git context
	gitContext := getGitContext(dir, r)
	if gitContext != "" {
		out.WriteString("## Project Context\n")
		out.WriteString(gitContext)
		out.WriteString("\n")
	}

	// Get recommended task from beads
	taskInfo := getTaskRecommendation(dir, r)
	out.WriteString("## Recommended Task\n")
	if taskInfo != "" {
		out.WriteString(taskInfo)
	} else {
		out.WriteString("No beads task graph found. Run `bd init` to initialize, or use `vibes` to set up the project.\n")
	}
	out.WriteString("\n")

	// Protocol
	out.WriteString("## Protocol\n")
	out.WriteString(getProtocol(opts.Verbose))

	fmt.Print(out.String())
	return nil
}

func getGitContext(dir string, r runner.CommandRunner) string {
	var out strings.Builder

	// Current branch
	branch := git.GetCurrentBranch(dir, r)
	if branch != "" {
		out.WriteString(fmt.Sprintf("- **Branch**: %s\n", branch))
	}

	// Status summary
	status := git.GetWorkingTreeStatus(dir, r)
	if status == "" {
		out.WriteString("- **Status**: Clean working tree\n")
	} else {
		out.WriteString(fmt.Sprintf("- **Status**: %s\n", status))
	}

	// Recent commit
	recentCommit := git.GetRecentCommit(dir, r)
	if recentCommit != "" {
		out.WriteString(fmt.Sprintf("- **Recent**: \"%s\"\n", recentCommit))
	}

	return out.String()
}

func getTaskRecommendation(dir string, r runner.CommandRunner) string {
	// Check if beads is initialized
	if !beads.IsInitialized(dir) {
		return ""
	}

	// Try bv --robot-triage first (more intelligent recommendations)
	if output, err := r.RunWithTimeout(dir, 10*time.Second, "bv", "--robot-triage"); err == nil && output != "" {
		return output
	}

	// Fall back to bd ready
	if output, err := r.RunWithTimeout(dir, 10*time.Second, "bd", "ready"); err == nil && output != "" {
		return output
	}

	return "Beads initialized but no ready tasks found. Create tasks with `bd create \"Task name\" -p 1`\n"
}

func getProtocol(verbose bool) string {
	if verbose {
		return `1. **Claim the work**:
   ` + "```bash" + `
   bd update bd-XXXX --status in_progress
   bd show bd-XXXX
   ` + "```" + `

2. **Reserve files** via MCP Agent Mail:
   ` + "```" + `
   file_reservation_paths(
       project_key="project-name",
       agent_name="YourAgentIdentity",
       patterns=["<your-file-patterns>"],
       ttl_seconds=3600,
       exclusive=true
   )
   ` + "```" + `

3. **Announce start** in the bead's thread

4. **Execute** the implementation

5. **Complete**:
   ` + "```bash" + `
   bd update bd-XXXX --status closed
   ` + "```" + `

Begin working on the highest priority task now.
`
	}

	return `1. Claim: ` + "`bd update <id> --status in_progress`" + `
2. Reserve files via MCP Agent Mail (if available)
3. Execute the implementation
4. Complete: ` + "`bd update <id> --status closed`" + `

Begin working on the highest priority task now.
`
}
