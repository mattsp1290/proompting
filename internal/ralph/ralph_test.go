package ralph

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// MockRunner is a mock implementation of runner.CommandRunner for testing
type MockRunner struct {
	RunFunc            func(dir string, command string, args ...string) (string, error)
	RunWithTimeoutFunc func(dir string, timeout time.Duration, command string, args ...string) (string, error)
}

func (m *MockRunner) Run(dir string, command string, args ...string) (string, error) {
	if m.RunFunc != nil {
		return m.RunFunc(dir, command, args...)
	}
	return "", nil
}

func (m *MockRunner) RunWithTimeout(dir string, timeout time.Duration, command string, args ...string) (string, error) {
	if m.RunWithTimeoutFunc != nil {
		return m.RunWithTimeoutFunc(dir, timeout, command, args...)
	}
	return "", nil
}

func TestModeSelection(t *testing.T) {
	t.Run("default mode is SingleTask", func(t *testing.T) {
		opts := Options{}
		if opts.Mode != ModeSingleTask {
			t.Errorf("expected default mode to be ModeSingleTask, got %v", opts.Mode)
		}
	})

	t.Run("mode constants are distinct", func(t *testing.T) {
		if ModeSingleTask == ModeGoal {
			t.Error("ModeSingleTask should not equal ModeGoal")
		}
		if ModeGoal == ModeAutopilot {
			t.Error("ModeGoal should not equal ModeAutopilot")
		}
		if ModeSingleTask == ModeAutopilot {
			t.Error("ModeSingleTask should not equal ModeAutopilot")
		}
	})
}

func TestBuildModeSection(t *testing.T) {
	t.Run("single task mode", func(t *testing.T) {
		opts := Options{Mode: ModeSingleTask}
		result := buildModeSection(opts)

		if !strings.Contains(result, "Single Task") {
			t.Errorf("expected 'Single Task' in output, got: %s", result)
		}
	})

	t.Run("goal mode", func(t *testing.T) {
		opts := Options{Mode: ModeGoal, Goal: "Add dark mode support"}
		result := buildModeSection(opts)

		if !strings.Contains(result, "Goal:") {
			t.Errorf("expected 'Goal:' in output, got: %s", result)
		}
		if !strings.Contains(result, "Add dark mode support") {
			t.Errorf("expected goal text in output, got: %s", result)
		}
	})

	t.Run("autopilot mode", func(t *testing.T) {
		opts := Options{Mode: ModeAutopilot}
		result := buildModeSection(opts)

		if !strings.Contains(result, "Autopilot") {
			t.Errorf("expected 'Autopilot' in output, got: %s", result)
		}
	})

	t.Run("with max iterations", func(t *testing.T) {
		opts := Options{Mode: ModeSingleTask, MaxIterations: 30}
		result := buildModeSection(opts)

		if !strings.Contains(result, "Max iterations") {
			t.Errorf("expected max iterations in output, got: %s", result)
		}
		if !strings.Contains(result, "30") {
			t.Errorf("expected '30' in output, got: %s", result)
		}
	})

	t.Run("without max iterations", func(t *testing.T) {
		opts := Options{Mode: ModeSingleTask, MaxIterations: 0}
		result := buildModeSection(opts)

		if strings.Contains(result, "Max iterations") {
			t.Errorf("expected no max iterations when 0, got: %s", result)
		}
	})
}

func TestDetectTestCommand(t *testing.T) {
	tests := []struct {
		name     string
		setup    func(dir string)
		expected string
	}{
		{
			name: "Go project",
			setup: func(dir string) {
				os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module test"), 0644)
			},
			expected: "go test ./... && go build ./...",
		},
		{
			name: "Node project with npm",
			setup: func(dir string) {
				os.WriteFile(filepath.Join(dir, "package.json"), []byte("{}"), 0644)
			},
			expected: "npm test",
		},
		{
			name: "Node project with yarn",
			setup: func(dir string) {
				os.WriteFile(filepath.Join(dir, "package.json"), []byte("{}"), 0644)
				os.WriteFile(filepath.Join(dir, "yarn.lock"), []byte(""), 0644)
			},
			expected: "yarn test",
		},
		{
			name: "Node project with pnpm",
			setup: func(dir string) {
				os.WriteFile(filepath.Join(dir, "package.json"), []byte("{}"), 0644)
				os.WriteFile(filepath.Join(dir, "pnpm-lock.yaml"), []byte(""), 0644)
			},
			expected: "pnpm test",
		},
		{
			name: "Python project with pyproject.toml",
			setup: func(dir string) {
				os.WriteFile(filepath.Join(dir, "pyproject.toml"), []byte(""), 0644)
			},
			expected: "pytest",
		},
		{
			name: "Python project with setup.py",
			setup: func(dir string) {
				os.WriteFile(filepath.Join(dir, "setup.py"), []byte(""), 0644)
			},
			expected: "pytest",
		},
		{
			name: "Rust project",
			setup: func(dir string) {
				os.WriteFile(filepath.Join(dir, "Cargo.toml"), []byte(""), 0644)
			},
			expected: "cargo test && cargo build",
		},
		{
			name: "Makefile project",
			setup: func(dir string) {
				os.WriteFile(filepath.Join(dir, "Makefile"), []byte(""), 0644)
			},
			expected: "make test",
		},
		{
			name: "No recognized project type",
			setup: func(dir string) {
				// Empty directory
			},
			expected: "# No test runner detected",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			tc.setup(tmpDir)

			result := detectTestCommand(tmpDir)

			if !strings.Contains(result, tc.expected) {
				t.Errorf("expected '%s' in output, got: %s", tc.expected, result)
			}
		})
	}
}

func TestBuildCompletionRequirements(t *testing.T) {
	t.Run("non-verbose", func(t *testing.T) {
		tmpDir := t.TempDir()
		os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte("module test"), 0644)

		result := buildCompletionRequirements(tmpDir, false)

		if !strings.Contains(result, "go test") {
			t.Errorf("expected test command, got: %s", result)
		}
		if !strings.Contains(result, "<promise>COMPLETE</promise>") {
			t.Errorf("expected promise tag, got: %s", result)
		}
		if !strings.Contains(result, "Both conditions must be met") {
			t.Errorf("expected dual condition note, got: %s", result)
		}
	})

	t.Run("verbose includes details", func(t *testing.T) {
		tmpDir := t.TempDir()

		result := buildCompletionRequirements(tmpDir, true)

		if !strings.Contains(result, "Completion Criteria Details") {
			t.Errorf("expected verbose details, got: %s", result)
		}
		if !strings.Contains(result, "exit code 0") {
			t.Errorf("expected exit code note, got: %s", result)
		}
	})
}

func TestSanitizeForShell(t *testing.T) {
	t.Run("replaces parentheses", func(t *testing.T) {
		result := sanitizeForShell("message (2 hours ago)")
		if strings.Contains(result, "(") || strings.Contains(result, ")") {
			t.Errorf("expected parentheses to be replaced, got: %s", result)
		}
		if !strings.Contains(result, "[2 hours ago]") {
			t.Errorf("expected brackets instead of parentheses, got: %s", result)
		}
	})

	t.Run("removes backticks", func(t *testing.T) {
		result := sanitizeForShell("use `command` here")
		if strings.Contains(result, "`") {
			t.Errorf("expected backticks to be removed, got: %s", result)
		}
	})

	t.Run("removes dollar signs", func(t *testing.T) {
		result := sanitizeForShell("costs $100")
		if strings.Contains(result, "$") {
			t.Errorf("expected dollar sign to be removed, got: %s", result)
		}
	})
}

func TestBuildCheckpointProtocol(t *testing.T) {
	t.Run("non-verbose", func(t *testing.T) {
		result := buildCheckpointProtocol(false)

		if !strings.Contains(result, "git add -A && git commit") {
			t.Errorf("expected git command, got: %s", result)
		}
		if !strings.Contains(result, "ralph: iteration N") {
			t.Errorf("expected commit message format, got: %s", result)
		}
	})

	t.Run("verbose includes guidelines", func(t *testing.T) {
		result := buildCheckpointProtocol(true)

		if !strings.Contains(result, "Commit Guidelines") {
			t.Errorf("expected guidelines header, got: %s", result)
		}
		if !strings.Contains(result, "under 50 chars") {
			t.Errorf("expected character limit note, got: %s", result)
		}
		if !strings.Contains(result, "Examples") {
			t.Errorf("expected examples, got: %s", result)
		}
	})
}

func TestBuildIterationProtocol(t *testing.T) {
	t.Run("non-verbose", func(t *testing.T) {
		result := buildIterationProtocol(false)

		if !strings.Contains(result, "ASSESS") {
			t.Errorf("expected ASSESS step, got: %s", result)
		}
		if !strings.Contains(result, "EXECUTE") {
			t.Errorf("expected EXECUTE step, got: %s", result)
		}
		if !strings.Contains(result, "VERIFY") {
			t.Errorf("expected VERIFY step, got: %s", result)
		}
		if !strings.Contains(result, "CHECKPOINT") {
			t.Errorf("expected CHECKPOINT step, got: %s", result)
		}
		if !strings.Contains(result, "EVALUATE") {
			t.Errorf("expected EVALUATE step, got: %s", result)
		}
		if !strings.Contains(result, "Begin working now") {
			t.Errorf("expected call to action, got: %s", result)
		}
	})

	t.Run("verbose includes details", func(t *testing.T) {
		result := buildIterationProtocol(true)

		if !strings.Contains(result, "Each iteration follows this cycle") {
			t.Errorf("expected cycle explanation, got: %s", result)
		}
		if !strings.Contains(result, "Do not skip steps") {
			t.Errorf("expected warning, got: %s", result)
		}
	})
}

func TestBuildTaskSection(t *testing.T) {
	t.Run("single task mode without beads", func(t *testing.T) {
		tmpDir := t.TempDir()
		mock := &MockRunner{}

		opts := Options{Mode: ModeSingleTask}
		result := buildTaskSection(tmpDir, opts, mock)

		if !strings.Contains(result, "No beads task graph found") {
			t.Errorf("expected no beads message, got: %s", result)
		}
	})

	t.Run("single task mode with beads", func(t *testing.T) {
		tmpDir := t.TempDir()
		os.MkdirAll(filepath.Join(tmpDir, ".beads"), 0755)

		mock := &MockRunner{
			RunWithTimeoutFunc: func(dir string, timeout time.Duration, command string, args ...string) (string, error) {
				if command == "bv" {
					return "Task 1: Important task\nPriority: High", nil
				}
				return "", nil
			},
		}

		opts := Options{Mode: ModeSingleTask}
		result := buildTaskSection(tmpDir, opts, mock)

		if !strings.Contains(result, "Task 1: Important task") {
			t.Errorf("expected task info, got: %s", result)
		}
	})

	t.Run("goal mode", func(t *testing.T) {
		tmpDir := t.TempDir()
		mock := &MockRunner{}

		opts := Options{Mode: ModeGoal, Goal: "Implement user authentication"}
		result := buildTaskSection(tmpDir, opts, mock)

		if !strings.Contains(result, "Implement user authentication") {
			t.Errorf("expected goal text, got: %s", result)
		}
		if !strings.Contains(result, "Work iteratively") {
			t.Errorf("expected iterative guidance, got: %s", result)
		}
	})

	t.Run("autopilot mode", func(t *testing.T) {
		tmpDir := t.TempDir()
		os.MkdirAll(filepath.Join(tmpDir, ".beads"), 0755)

		mock := &MockRunner{
			RunWithTimeoutFunc: func(dir string, timeout time.Duration, command string, args ...string) (string, error) {
				if command == "bv" {
					return "Task 1\nTask 2\nTask 3", nil
				}
				return "", nil
			},
		}

		opts := Options{Mode: ModeAutopilot}
		result := buildTaskSection(tmpDir, opts, mock)

		if !strings.Contains(result, "task graph autonomously") {
			t.Errorf("expected autopilot description, got: %s", result)
		}
		if !strings.Contains(result, "Task Overview") {
			t.Errorf("expected task overview, got: %s", result)
		}
	})
}

func TestBuildProjectContext(t *testing.T) {
	t.Run("clean repo", func(t *testing.T) {
		mock := &MockRunner{
			RunFunc: func(dir string, command string, args ...string) (string, error) {
				if len(args) >= 2 && args[0] == "rev-parse" {
					return "main", nil
				}
				if len(args) >= 1 && args[0] == "status" {
					return "", nil
				}
				if len(args) >= 1 && args[0] == "log" {
					return "Initial commit (2 hours ago)", nil
				}
				return "", nil
			},
		}

		result := buildProjectContext("/test/dir", mock)

		if !strings.Contains(result, "main") {
			t.Errorf("expected branch name, got: %s", result)
		}
		if !strings.Contains(result, "Clean working tree") {
			t.Errorf("expected clean status, got: %s", result)
		}
	})
}

func TestRun(t *testing.T) {
	t.Run("single task mode", func(t *testing.T) {
		tmpDir := t.TempDir()
		mock := &MockRunner{
			RunFunc: func(dir string, command string, args ...string) (string, error) {
				return "", nil
			},
		}

		opts := Options{
			Dir:    tmpDir,
			Mode:   ModeSingleTask,
			Runner: mock,
		}

		err := Run(opts)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("goal mode", func(t *testing.T) {
		tmpDir := t.TempDir()
		mock := &MockRunner{
			RunFunc: func(dir string, command string, args ...string) (string, error) {
				return "", nil
			},
		}

		opts := Options{
			Dir:    tmpDir,
			Mode:   ModeGoal,
			Goal:   "Test goal",
			Runner: mock,
		}

		err := Run(opts)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("autopilot mode", func(t *testing.T) {
		tmpDir := t.TempDir()
		mock := &MockRunner{
			RunFunc: func(dir string, command string, args ...string) (string, error) {
				return "", nil
			},
		}

		opts := Options{
			Dir:    tmpDir,
			Mode:   ModeAutopilot,
			Runner: mock,
		}

		err := Run(opts)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("verbose mode", func(t *testing.T) {
		tmpDir := t.TempDir()
		mock := &MockRunner{
			RunFunc: func(dir string, command string, args ...string) (string, error) {
				return "", nil
			},
		}

		opts := Options{
			Dir:     tmpDir,
			Verbose: true,
			Runner:  mock,
		}

		err := Run(opts)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("with nil runner uses default", func(t *testing.T) {
		tmpDir := t.TempDir()

		opts := Options{
			Dir:    tmpDir,
			Runner: nil,
		}

		// Should not panic
		_ = Run(opts)
	})

	t.Run("with max iterations", func(t *testing.T) {
		tmpDir := t.TempDir()
		mock := &MockRunner{
			RunFunc: func(dir string, command string, args ...string) (string, error) {
				return "", nil
			},
		}

		opts := Options{
			Dir:           tmpDir,
			MaxIterations: 50,
			Runner:        mock,
		}

		err := Run(opts)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})
}

func TestFileExists(t *testing.T) {
	t.Run("existing file", func(t *testing.T) {
		tmpDir := t.TempDir()
		filePath := filepath.Join(tmpDir, "test.txt")
		os.WriteFile(filePath, []byte("test"), 0644)

		if !fileExists(filePath) {
			t.Error("expected fileExists to return true for existing file")
		}
	})

	t.Run("non-existing file", func(t *testing.T) {
		if fileExists("/nonexistent/path/to/file.txt") {
			t.Error("expected fileExists to return false for non-existing file")
		}
	})
}
