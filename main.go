package main

import (
	"embed"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/vibes-project/vibes/internal/setup"
	"github.com/vibes-project/vibes/internal/styles"
)

//go:embed proompts
var proomptFS embed.FS

var (
	version = "dev"

	migrateTasks bool
	skipProompts bool
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "vibes [target-directory]",
		Short: "Set up AI agent infrastructure in a git project",
		Long: `Vibes sets up proompts, Beads, and MCP Agent Mail integration in a git project.

When run with no arguments in a git repository that doesn't have vibes installed,
it will automatically set up the AI agent infrastructure in the current directory.

Examples:
  vibes                    # Set up in current directory
  vibes /path/to/project   # Set up in specified directory
  vibes --migrate .        # Set up and migrate existing tasks.yaml`,
		Args:    cobra.MaximumNArgs(1),
		Version: version,
		RunE:    runSetup,
	}

	rootCmd.Flags().BoolVar(&migrateTasks, "migrate", false, "Migrate existing tasks.yaml to Beads")
	rootCmd.Flags().BoolVar(&skipProompts, "skip-proompts", false, "Don't copy proompts directory")

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func runSetup(cmd *cobra.Command, args []string) error {
	// Determine target directory
	var targetDir string
	if len(args) > 0 {
		targetDir = args[0]
	} else {
		// Use current directory
		cwd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("getting current directory: %w", err)
		}
		targetDir = cwd
	}

	// Check if it's a git repo
	if !setup.IsGitRepo(targetDir) {
		fmt.Println(styles.Error("Directory is not a git repository"))
		fmt.Println("Run this command in a git repository or specify a target directory.")
		return fmt.Errorf("not a git repository")
	}

	// Check if vibes is already set up (when no args provided)
	if len(args) == 0 && setup.HasVibesSetup(targetDir) && !migrateTasks {
		fmt.Println(styles.Info("Vibes is already set up in this directory."))
		fmt.Println()
		fmt.Println("Options:")
		fmt.Println("  vibes --migrate       # Migrate tasks.yaml to Beads")
		fmt.Println("  vibes --skip-proompts # Reinitialize without overwriting proompts")
		fmt.Println("  vibes /other/path     # Set up in a different directory")
		return nil
	}

	// Run setup
	opts := setup.Options{
		TargetDir:    targetDir,
		MigrateTasks: migrateTasks,
		SkipProompts: skipProompts,
		SourceFS:     proomptFS,
	}

	_, err := setup.Run(opts)
	return err
}
