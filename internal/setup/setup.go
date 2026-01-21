package setup

import (
	"embed"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/huh"
	"github.com/vibes-project/vibes/internal/styles"
)

// Options configures the setup behavior
type Options struct {
	TargetDir    string
	MigrateTasks bool
	SkipProompts bool
	SourceFS     embed.FS
}

// Result tracks what was done during setup
type Result struct {
	ProomptsCopied   bool
	BeadsInitialized bool
	GitignoreUpdated bool
	HookInstalled    bool
}

// Run executes the full setup process
func Run(opts Options) (*Result, error) {
	result := &Result{}

	// Resolve target directory
	targetDir, err := filepath.Abs(opts.TargetDir)
	if err != nil {
		return nil, fmt.Errorf("resolving target directory: %w", err)
	}

	// Validate target
	if err := validateTarget(targetDir); err != nil {
		return nil, err
	}

	fmt.Println(styles.Header("Setting up AI Agent Infrastructure"))
	fmt.Println(styles.Info("Target: " + targetDir))
	fmt.Println()

	// Step 1: Copy proompts
	if !opts.SkipProompts {
		copied, err := copyProompts(opts.SourceFS, targetDir)
		if err != nil {
			return result, fmt.Errorf("copying proompts: %w", err)
		}
		result.ProomptsCopied = copied
	} else {
		fmt.Println(styles.Header("Step 1: Proompts Directory"))
		fmt.Println(styles.Info("Skipping proompts copy (--skip-proompts)"))
	}

	// Step 2: Initialize Beads
	initialized, err := initBeads(targetDir)
	if err != nil {
		return result, fmt.Errorf("initializing beads: %w", err)
	}
	result.BeadsInitialized = initialized

	// Step 2b: Migrate tasks if requested
	if opts.MigrateTasks {
		if err := migrateTasks(targetDir); err != nil {
			fmt.Println(styles.Info("Migration note: " + err.Error()))
		}
	}

	// Step 3: Check MCP Agent Mail
	checkAgentMail()

	// Step 4: Check Beads Viewer
	checkBeadsViewer()

	// Step 5: Update .gitignore
	updated, err := updateGitignore(targetDir)
	if err != nil {
		return result, fmt.Errorf("updating gitignore: %w", err)
	}
	result.GitignoreUpdated = updated

	// Step 6: Pre-commit hook (optional)
	installed, err := installPreCommitHook(targetDir)
	if err != nil {
		fmt.Println(styles.Info("Skipped pre-commit hook"))
	} else {
		result.HookInstalled = installed
	}

	// Print summary
	printSummary(targetDir)

	return result, nil
}

// HasVibesSetup checks if a directory already has vibes installed
func HasVibesSetup(dir string) bool {
	proompts := filepath.Join(dir, "proompts")
	beads := filepath.Join(dir, ".beads")

	_, errP := os.Stat(proompts)
	_, errB := os.Stat(beads)

	return errP == nil || errB == nil
}

// IsGitRepo checks if a directory is a git repository
func IsGitRepo(dir string) bool {
	gitDir := filepath.Join(dir, ".git")
	info, err := os.Stat(gitDir)
	return err == nil && info.IsDir()
}

func validateTarget(targetDir string) error {
	info, err := os.Stat(targetDir)
	if os.IsNotExist(err) {
		return fmt.Errorf("directory '%s' does not exist", targetDir)
	}
	if err != nil {
		return fmt.Errorf("checking directory: %w", err)
	}
	if !info.IsDir() {
		return fmt.Errorf("'%s' is not a directory", targetDir)
	}
	if !IsGitRepo(targetDir) {
		return fmt.Errorf("directory '%s' is not a git repository", targetDir)
	}
	return nil
}

func copyProompts(sourceFS embed.FS, targetDir string) (bool, error) {
	fmt.Println(styles.Header("Step 1: Proompts Directory"))

	targetProompts := filepath.Join(targetDir, "proompts")

	// Check if already exists
	if _, err := os.Stat(targetProompts); err == nil {
		fmt.Println(styles.Info("Proompts directory already exists"))

		var overwrite bool
		form := huh.NewForm(
			huh.NewGroup(
				huh.NewConfirm().
					Title("Overwrite existing files?").
					Value(&overwrite),
			),
		)

		if err := form.Run(); err != nil {
			return false, err
		}

		if !overwrite {
			fmt.Println(styles.Info("Keeping existing proompts"))
			return false, nil
		}
	} else {
		if err := os.MkdirAll(targetProompts, 0755); err != nil {
			return false, err
		}
	}

	// Copy files from embedded FS
	err := fs.WalkDir(sourceFS, "proompts", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Get relative path from "proompts"
		relPath, err := filepath.Rel("proompts", path)
		if err != nil {
			return err
		}

		destPath := filepath.Join(targetProompts, relPath)

		if d.IsDir() {
			return os.MkdirAll(destPath, 0755)
		}

		// Read source file
		content, err := sourceFS.ReadFile(path)
		if err != nil {
			return err
		}

		// Write destination file
		return os.WriteFile(destPath, content, 0644)
	})

	if err != nil {
		return false, err
	}

	fmt.Println(styles.Success("Created proompts directory"))
	return true, nil
}

func initBeads(targetDir string) (bool, error) {
	fmt.Println(styles.Header("Step 2: Beads Task Graph"))

	beadsDir := filepath.Join(targetDir, ".beads")
	if _, err := os.Stat(beadsDir); err == nil {
		fmt.Println(styles.Info("Beads already initialized"))
		return false, nil
	}

	// Check if bd is available
	bdPath, err := exec.LookPath("bd")
	if err != nil {
		fmt.Println(styles.Info("Beads CLI (bd) not found"))
		fmt.Println("  Install with: npm install -g @beads/bd")
		fmt.Println("  Or: go install github.com/steveyegge/beads/cmd/bd@latest")
		fmt.Println()
		fmt.Println("  After installing, run: cd " + targetDir + " && bd init")
		return false, nil
	}

	// Run bd init
	cmd := exec.Command(bdPath, "init")
	cmd.Dir = targetDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return false, fmt.Errorf("running bd init: %w", err)
	}

	fmt.Println(styles.Success("Initialized Beads (.beads/)"))
	return true, nil
}

func migrateTasks(targetDir string) error {
	fmt.Println(styles.Header("Step 2b: Migrate tasks.yaml to Beads"))

	// Look for tasks.yaml
	var tasksYaml string
	candidates := []string{
		filepath.Join(targetDir, "tasks.yaml"),
		filepath.Join(targetDir, "proompts", "tasks.yaml"),
	}

	for _, path := range candidates {
		if _, err := os.Stat(path); err == nil {
			tasksYaml = path
			break
		}
	}

	if tasksYaml == "" {
		fmt.Println(styles.Info("No tasks.yaml found to migrate"))
		fmt.Println("  Looked in: " + targetDir + "/tasks.yaml")
		fmt.Println("             " + targetDir + "/proompts/tasks.yaml")
		return nil
	}

	fmt.Println(styles.Info("Found tasks.yaml at: " + tasksYaml))

	// Check for Claude CLI
	claudePath, err := exec.LookPath("claude")
	if err != nil {
		fmt.Println(styles.Info("Claude Code CLI not found"))
		fmt.Println("  Install with: npm install -g @anthropic-ai/claude-code")
		return nil
	}

	var runMigration bool
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewConfirm().
				Title("Run migration now?").
				Value(&runMigration),
		),
	)

	if err := form.Run(); err != nil {
		return err
	}

	if !runMigration {
		fmt.Println(styles.Info("Migration skipped"))
		return nil
	}

	// Run migration script if it exists
	// For now, just note that Claude is available
	_ = claudePath
	fmt.Println(styles.Info("Migration would run here with Claude CLI"))
	return nil
}

func checkAgentMail() {
	fmt.Println(styles.Header("Step 3: MCP Agent Mail"))

	client := &http.Client{Timeout: 2 * time.Second}
	resp, err := client.Get("http://localhost:8765/health")
	if err == nil {
		defer resp.Body.Close()
		if resp.StatusCode == http.StatusOK {
			fmt.Println(styles.Success("Agent Mail server is running on :8765"))
			return
		}
	}

	fmt.Println(styles.Info("Agent Mail server not detected"))
	fmt.Println("  Install with:")
	fmt.Println(`  curl -fsSL "https://raw.githubusercontent.com/Dicklesworthstone/mcp_agent_mail/main/scripts/install.sh" | bash -s -- --yes`)
	fmt.Println()
	fmt.Println("  Start server with: am")
	fmt.Println("  Web UI: http://localhost:8765")
}

func checkBeadsViewer() {
	fmt.Println(styles.Header("Step 4: Beads Viewer (bv)"))

	if _, err := exec.LookPath("bv"); err == nil {
		fmt.Println(styles.Success("Beads Viewer (bv) is installed"))
		return
	}

	fmt.Println(styles.Info("Beads Viewer (bv) not found"))
	fmt.Println("  Install with: go install github.com/Dicklesworthstone/beads_viewer@latest")
	fmt.Println()
	fmt.Println("  This provides robot flags for AI agents:")
	fmt.Println("    bv --robot-triage    # Intelligent task recommendations")
	fmt.Println("    bv --robot-plan      # Parallel execution tracks")
	fmt.Println("    bv --robot-insights  # PageRank, critical path")
}

func updateGitignore(targetDir string) (bool, error) {
	fmt.Println(styles.Header("Step 5: Git Configuration"))

	gitignorePath := filepath.Join(targetDir, ".gitignore")

	// Read existing content
	var existing []byte
	if _, err := os.Stat(gitignorePath); err == nil {
		existing, err = os.ReadFile(gitignorePath)
		if err != nil {
			return false, err
		}
	} else {
		fmt.Println(styles.Success("Created .gitignore"))
	}

	content := string(existing)
	lines := strings.Split(content, "\n")
	lineSet := make(map[string]bool)
	for _, line := range lines {
		lineSet[strings.TrimSpace(line)] = true
	}

	entries := []string{".beads/.cache/"}
	added := false

	for _, entry := range entries {
		if !lineSet[entry] {
			if content != "" && !strings.HasSuffix(content, "\n") {
				content += "\n"
			}
			content += entry + "\n"
			fmt.Println(styles.Success("Added " + entry + " to .gitignore"))
			added = true
		}
	}

	if added {
		if err := os.WriteFile(gitignorePath, []byte(content), 0644); err != nil {
			return false, err
		}
	}

	return added, nil
}

func installPreCommitHook(targetDir string) (bool, error) {
	fmt.Println(styles.Header("Step 6: Pre-commit Hook"))

	var install bool
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewConfirm().
				Title("Install file reservation check hook?").
				Value(&install),
		),
	)

	if err := form.Run(); err != nil {
		return false, err
	}

	if !install {
		return false, fmt.Errorf("user declined")
	}

	hookPath := filepath.Join(targetDir, ".git", "hooks", "pre-commit")
	hookContent := `#!/bin/bash
# Pre-commit hook: Check for file reservation conflicts
# Part of Beads + MCP Agent Mail integration

# Skip if agent mail server isn't running
if ! curl -s http://localhost:8765/health &> /dev/null 2>&1; then
    exit 0
fi

# Get staged files
STAGED_FILES=$(git diff --cached --name-only)

if [ -z "$STAGED_FILES" ]; then
    exit 0
fi

# Check for conflicts (implement based on your MCP integration)
# This is a placeholder - actual implementation would query Agent Mail API
# echo "Checking file reservations for: $STAGED_FILES"

# For now, just pass
exit 0
`

	if err := os.WriteFile(hookPath, []byte(hookContent), 0755); err != nil {
		return false, err
	}

	fmt.Println(styles.Success("Installed pre-commit hook"))
	return true, nil
}

func printSummary(targetDir string) {
	fmt.Println()
	fmt.Println(styles.Header("Setup Complete"))
	fmt.Println()
	fmt.Println("Directory structure:")
	fmt.Println("  " + targetDir + "/")
	fmt.Println("  ├── proompts/              # Prompts and documentation")
	fmt.Println("  │   ├── initial-prompt.md")
	fmt.Println("  │   ├── start-task.md")
	fmt.Println("  │   ├── request-review.md")
	fmt.Println("  │   ├── act-on-review.md")
	fmt.Println("  │   └── docs/")
	fmt.Println("  ├── .beads/                # Beads task graph (if initialized)")
	fmt.Println("  └── .gitignore             # Updated")
	fmt.Println()
	fmt.Println("Quick Start:")
	fmt.Println("  1. Create task graph:  Use proompts/initial-prompt.md")
	fmt.Println("     OR migrate existing: vibes --migrate " + targetDir)
	fmt.Println("  2. Start working:      bv --robot-triage && bd ready")
	fmt.Println("  3. Get next task:      Use proompts/start-task.md")
	fmt.Println("  4. Request review:     Use proompts/request-review.md")
	fmt.Println("  5. Act on feedback:    Use proompts/act-on-review.md")
	fmt.Println()
	fmt.Println("Web UI (when Agent Mail running): http://localhost:8765")
	fmt.Println()
	fmt.Println(styles.Success("The vibes are going! Good luck with the project."))
}

// CopyFile copies a single file from src to dst
func CopyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, sourceFile)
	return err
}

// CopyDir recursively copies a directory from src to dst
func CopyDir(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}

		destPath := filepath.Join(dst, relPath)

		if info.IsDir() {
			return os.MkdirAll(destPath, info.Mode())
		}

		return CopyFile(path, destPath)
	})
}
