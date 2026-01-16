package main

import (
	"embed"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/vibes-project/vibes/internal/done"
	"github.com/vibes-project/vibes/internal/next"
	"github.com/vibes-project/vibes/internal/resume"
	"github.com/vibes-project/vibes/internal/setup"
	"github.com/vibes-project/vibes/internal/styles"
)

//go:embed proompts
var proomptFS embed.FS

var (
	version = "dev"

	migrateTasks  bool
	skipProompts  bool
	nextVerbose   bool
	doneVerbose   bool
	resumeVerbose bool
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

	// Next command - outputs prompt for claude
	nextCmd := &cobra.Command{
		Use:   "next",
		Short: "Output the next task as a prompt for Claude",
		Long: `Outputs a ready-to-use prompt containing the next recommended task from Beads,
current git context, and the start-task protocol.

Usage with Claude:
  claude "$(vibes next)"

This eliminates the manual workflow of running bv --robot-triage,
copying output, and combining with start-task.md.`,
		Args: cobra.NoArgs,
		RunE: runNext,
	}
	nextCmd.Flags().BoolVarP(&nextVerbose, "verbose", "v", false, "Include full protocol details")
	rootCmd.AddCommand(nextCmd)

	// Done command - outputs completion prompt for claude
	doneCmd := &cobra.Command{
		Use:   "done",
		Short: "Output a completion prompt for the current task",
		Long: `Outputs a ready-to-use prompt for completing the current task, including
work summary, recent commits, and the completion protocol.

Usage with Claude:
  claude "$(vibes done)"

This helps you wrap up work by:
- Detecting the current task from branch name or in-progress beads
- Showing recent commits on the branch
- Providing the completion protocol (release reservations, update status, etc.)`,
		Args: cobra.NoArgs,
		RunE: runDone,
	}
	doneCmd.Flags().BoolVarP(&doneVerbose, "verbose", "v", false, "Include full protocol details")
	rootCmd.AddCommand(doneCmd)

	// Resume command - outputs prompt to continue work
	resumeCmd := &cobra.Command{
		Use:   "resume",
		Short: "Output a prompt to resume work on the current task",
		Long: `Outputs a ready-to-use prompt for resuming work after a break or in a new session.
Includes current work context, uncommitted changes, recent commits, and pending items.

Usage with Claude:
  claude "$(vibes resume)"

This helps you continue seamlessly by:
- Detecting the current task from branch name or in-progress beads
- Showing uncommitted changes and recent commits
- Checking for pending messages or review feedback
- Providing the resume protocol`,
		Args: cobra.NoArgs,
		RunE: runResume,
	}
	resumeCmd.Flags().BoolVarP(&resumeVerbose, "verbose", "v", false, "Include full protocol details")
	rootCmd.AddCommand(resumeCmd)

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

func runNext(cmd *cobra.Command, args []string) error {
	opts := next.Options{
		Verbose: nextVerbose,
	}
	return next.Run(opts)
}

func runDone(cmd *cobra.Command, args []string) error {
	opts := done.Options{
		Verbose: doneVerbose,
	}
	return done.Run(opts)
}

func runResume(cmd *cobra.Command, args []string) error {
	opts := resume.Options{
		Verbose: resumeVerbose,
	}
	return resume.Run(opts)
}
