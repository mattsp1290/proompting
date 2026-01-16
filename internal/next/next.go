package next

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// Options configures the next command behavior
type Options struct {
	Dir     string // Target directory (defaults to cwd)
	Verbose bool   // Include full protocol details
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

	var out strings.Builder

	// Header
	projectName := filepath.Base(dir)
	out.WriteString(fmt.Sprintf("# Next Task for %s\n\n", projectName))

	// Git context
	gitContext := getGitContext(dir)
	if gitContext != "" {
		out.WriteString("## Project Context\n")
		out.WriteString(gitContext)
		out.WriteString("\n")
	}

	// Get recommended task from beads
	taskInfo := getTaskRecommendation(dir)
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

func getGitContext(dir string) string {
	var out strings.Builder

	// Current branch
	branch := runGitCmd(dir, "rev-parse", "--abbrev-ref", "HEAD")
	if branch != "" {
		out.WriteString(fmt.Sprintf("- **Branch**: %s\n", branch))
	}

	// Status summary
	status := runGitCmd(dir, "status", "--porcelain")
	if status == "" {
		out.WriteString("- **Status**: Clean working tree\n")
	} else {
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
			out.WriteString(fmt.Sprintf("- **Status**: %s\n", strings.Join(parts, ", ")))
		}
	}

	// Recent commit
	recentCommit := runGitCmd(dir, "log", "-1", "--format=%s (%ar)")
	if recentCommit != "" {
		out.WriteString(fmt.Sprintf("- **Recent**: \"%s\"\n", recentCommit))
	}

	return out.String()
}

func getTaskRecommendation(dir string) string {
	// Check if beads is initialized
	beadsDir := filepath.Join(dir, ".beads")
	if _, err := os.Stat(beadsDir); os.IsNotExist(err) {
		return ""
	}

	// Try bv --robot-triage first (more intelligent recommendations)
	if output := runBeadsCmd(dir, "bv", "--robot-triage"); output != "" {
		return output
	}

	// Fall back to bd ready
	if output := runBeadsCmd(dir, "bd", "ready"); output != "" {
		return output
	}

	return "Beads initialized but no ready tasks found. Create tasks with `bd create \"Task name\" -p 1`\n"
}

func runGitCmd(dir string, args ...string) string {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = nil

	if err := cmd.Run(); err != nil {
		return ""
	}

	return strings.TrimSpace(stdout.String())
}

func runBeadsCmd(dir string, command string, args ...string) string {
	path, err := exec.LookPath(command)
	if err != nil {
		return ""
	}

	cmd := exec.Command(path, args...)
	cmd.Dir = dir
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = nil

	// Set a timeout for beads commands
	done := make(chan error, 1)
	go func() {
		done <- cmd.Run()
	}()

	select {
	case err := <-done:
		if err != nil {
			return ""
		}
		return strings.TrimSpace(stdout.String())
	case <-time.After(10 * time.Second):
		cmd.Process.Kill()
		return ""
	}
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
       patterns=["src/path/**"],
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
