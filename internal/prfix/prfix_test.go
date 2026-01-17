package prfix

import (
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

// mockError implements error interface for testing
type mockError struct{}

func (e *mockError) Error() string {
	return "mock error"
}

func TestGetExistingPR(t *testing.T) {
	t.Run("returns PR info when PR exists", func(t *testing.T) {
		mock := &MockRunner{
			RunWithTimeoutFunc: func(dir string, timeout time.Duration, command string, args ...string) (string, error) {
				if command == "gh" && len(args) >= 2 && args[0] == "pr" && args[1] == "view" {
					return `{"number":42,"title":"Test PR","url":"https://github.com/test/repo/pull/42","state":"OPEN","mergeable":"MERGEABLE","baseRefName":"main","headRefName":"feature/test"}`, nil
				}
				return "", nil
			},
		}

		result := getExistingPR("/test", "feature/test", mock)
		if result == nil {
			t.Fatal("expected PR info, got nil")
		}
		if result.Number != 42 {
			t.Errorf("expected PR number 42, got %d", result.Number)
		}
		if result.Title != "Test PR" {
			t.Errorf("expected title 'Test PR', got %s", result.Title)
		}
		if result.Mergeable != "MERGEABLE" {
			t.Errorf("expected mergeable 'MERGEABLE', got %s", result.Mergeable)
		}
	})

	t.Run("returns nil when no PR exists", func(t *testing.T) {
		mock := &MockRunner{
			RunWithTimeoutFunc: func(dir string, timeout time.Duration, command string, args ...string) (string, error) {
				return "", &mockError{}
			},
		}

		result := getExistingPR("/test", "feature/test", mock)
		if result != nil {
			t.Errorf("expected nil, got %+v", result)
		}
	})

	t.Run("returns nil on invalid JSON", func(t *testing.T) {
		mock := &MockRunner{
			RunWithTimeoutFunc: func(dir string, timeout time.Duration, command string, args ...string) (string, error) {
				return "not valid json", nil
			},
		}

		result := getExistingPR("/test", "feature/test", mock)
		if result != nil {
			t.Errorf("expected nil, got %+v", result)
		}
	})
}

func TestGetChecks(t *testing.T) {
	t.Run("returns checks when available", func(t *testing.T) {
		mock := &MockRunner{
			RunWithTimeoutFunc: func(dir string, timeout time.Duration, command string, args ...string) (string, error) {
				if command == "gh" && len(args) >= 2 && args[0] == "pr" && args[1] == "checks" {
					return `[{"name":"test","status":"COMPLETED","conclusion":"SUCCESS","detailsUrl":"https://example.com"},{"name":"lint","status":"COMPLETED","conclusion":"FAILURE","detailsUrl":"https://example.com/lint"}]`, nil
				}
				return "", nil
			},
		}

		result := getChecks("/test", 42, mock)
		if len(result) != 2 {
			t.Fatalf("expected 2 checks, got %d", len(result))
		}
		if result[0].Name != "test" {
			t.Errorf("expected first check name 'test', got %s", result[0].Name)
		}
		if result[1].Conclusion != "FAILURE" {
			t.Errorf("expected second check conclusion 'FAILURE', got %s", result[1].Conclusion)
		}
	})

	t.Run("returns nil on error", func(t *testing.T) {
		mock := &MockRunner{
			RunWithTimeoutFunc: func(dir string, timeout time.Duration, command string, args ...string) (string, error) {
				return "", &mockError{}
			},
		}

		result := getChecks("/test", 42, mock)
		if result != nil {
			t.Errorf("expected nil, got %+v", result)
		}
	})
}

func TestCategorizeChecks(t *testing.T) {
	checks := []CheckInfo{
		{Name: "passing", Status: "COMPLETED", Conclusion: "SUCCESS"},
		{Name: "failing", Status: "COMPLETED", Conclusion: "FAILURE"},
		{Name: "pending", Status: "IN_PROGRESS", Conclusion: ""},
		{Name: "skipped", Status: "COMPLETED", Conclusion: "SKIPPED"},
		{Name: "neutral", Status: "COMPLETED", Conclusion: "NEUTRAL"},
	}

	failing, passing, pending := categorizeChecks(checks)

	if len(failing) != 1 {
		t.Errorf("expected 1 failing check, got %d", len(failing))
	}
	if failing[0].Name != "failing" {
		t.Errorf("expected failing check name 'failing', got %s", failing[0].Name)
	}

	if len(passing) != 3 {
		t.Errorf("expected 3 passing checks, got %d", len(passing))
	}

	if len(pending) != 1 {
		t.Errorf("expected 1 pending check, got %d", len(pending))
	}
	if pending[0].Name != "pending" {
		t.Errorf("expected pending check name 'pending', got %s", pending[0].Name)
	}
}

func TestGetMergeableStatus(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"MERGEABLE", "✅ Yes"},
		{"CONFLICTING", "❌ Conflicts"},
		{"UNKNOWN", "⏳ Checking..."},
		{"other", "other"},
	}

	for _, tt := range tests {
		result := getMergeableStatus(tt.input)
		if result != tt.expected {
			t.Errorf("getMergeableStatus(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

func TestGetReviewEmoji(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"APPROVED", "✅"},
		{"CHANGES_REQUESTED", "❌"},
		{"COMMENTED", "💬"},
		{"PENDING", "⏳"},
		{"DISMISSED", "🚫"},
		{"other", "•"},
	}

	for _, tt := range tests {
		result := getReviewEmoji(tt.input)
		if result != tt.expected {
			t.Errorf("getReviewEmoji(%q) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}

func TestDetermineIssues(t *testing.T) {
	t.Run("detects merge conflicts", func(t *testing.T) {
		pr := &PRInfo{Mergeable: "CONFLICTING"}
		issues := determineIssues(pr, nil, nil, nil, nil)

		found := false
		for _, issue := range issues {
			if strings.Contains(issue, "Merge conflicts") {
				found = true
				break
			}
		}
		if !found {
			t.Error("expected merge conflicts issue")
		}
	})

	t.Run("detects CI failures", func(t *testing.T) {
		pr := &PRInfo{Mergeable: "MERGEABLE"}
		failingChecks := []CheckInfo{{Name: "test"}, {Name: "lint"}}
		issues := determineIssues(pr, failingChecks, nil, nil, nil)

		found := false
		for _, issue := range issues {
			if strings.Contains(issue, "CI failures") {
				found = true
				if !strings.Contains(issue, "test") || !strings.Contains(issue, "lint") {
					t.Error("expected failing check names in issue")
				}
				break
			}
		}
		if !found {
			t.Error("expected CI failures issue")
		}
	})

	t.Run("detects changes requested", func(t *testing.T) {
		pr := &PRInfo{Mergeable: "MERGEABLE"}
		reviews := []ReviewInfo{{Author: "reviewer", State: "CHANGES_REQUESTED"}}
		issues := determineIssues(pr, nil, nil, reviews, nil)

		found := false
		for _, issue := range issues {
			if strings.Contains(issue, "Changes requested") {
				found = true
				break
			}
		}
		if !found {
			t.Error("expected changes requested issue")
		}
	})

	t.Run("detects review comments", func(t *testing.T) {
		pr := &PRInfo{Mergeable: "MERGEABLE"}
		comments := []ReviewComment{{Body: "fix this"}, {Body: "and this"}}
		issues := determineIssues(pr, nil, nil, nil, comments)

		found := false
		for _, issue := range issues {
			if strings.Contains(issue, "review comment") {
				found = true
				break
			}
		}
		if !found {
			t.Error("expected review comments issue")
		}
	})

	t.Run("returns empty when all good", func(t *testing.T) {
		pr := &PRInfo{Mergeable: "MERGEABLE"}
		reviews := []ReviewInfo{{Author: "reviewer", State: "APPROVED"}}
		issues := determineIssues(pr, nil, nil, reviews, nil)

		if len(issues) != 0 {
			t.Errorf("expected no issues, got %v", issues)
		}
	})

	t.Run("mentions pending checks when no other issues", func(t *testing.T) {
		pr := &PRInfo{Mergeable: "MERGEABLE"}
		pendingChecks := []CheckInfo{{Name: "build"}}
		issues := determineIssues(pr, nil, pendingChecks, nil, nil)

		found := false
		for _, issue := range issues {
			if strings.Contains(issue, "still running") {
				found = true
				break
			}
		}
		if !found {
			t.Error("expected pending checks mention")
		}
	})
}

func TestGetProtocol(t *testing.T) {
	pr := &PRInfo{Number: 42, HeadRef: "feature/test", BaseRef: "main"}

	t.Run("no issues protocol", func(t *testing.T) {
		result := getProtocol(pr, nil, false)

		if !strings.Contains(result, "ready to merge") {
			t.Error("expected ready to merge message")
		}
		if !strings.Contains(result, "gh pr merge 42") {
			t.Error("expected merge command")
		}
	})

	t.Run("no issues verbose protocol", func(t *testing.T) {
		result := getProtocol(pr, nil, true)

		if !strings.Contains(result, "**Final review**") {
			t.Error("expected bold headers in verbose mode")
		}
		if !strings.Contains(result, "--squash") {
			t.Error("expected squash flag in verbose mode")
		}
		if !strings.Contains(result, "git branch -d") {
			t.Error("expected cleanup instructions")
		}
	})

	t.Run("with issues protocol", func(t *testing.T) {
		issues := []string{"CI failures"}
		result := getProtocol(pr, issues, false)

		if !strings.Contains(result, "gh pr checks 42") {
			t.Error("expected checks command")
		}
		if !strings.Contains(result, "Address the issues") {
			t.Error("expected address issues instruction")
		}
	})

	t.Run("with issues verbose protocol", func(t *testing.T) {
		issues := []string{"Merge conflicts"}
		result := getProtocol(pr, issues, true)

		if !strings.Contains(result, "**Investigate failures**") {
			t.Error("expected bold headers")
		}
		if !strings.Contains(result, "git rebase") {
			t.Error("expected rebase instructions for conflicts")
		}
		if !strings.Contains(result, "--force-with-lease") {
			t.Error("expected safe force push")
		}
	})
}

func TestRun(t *testing.T) {
	t.Run("no PR found", func(t *testing.T) {
		tmpDir := t.TempDir()

		mock := &MockRunner{
			RunFunc: func(dir string, command string, args ...string) (string, error) {
				if command == "git" && len(args) >= 2 && args[0] == "rev-parse" && args[1] == "--abbrev-ref" {
					return "feature/test", nil
				}
				return "", nil
			},
			RunWithTimeoutFunc: func(dir string, timeout time.Duration, command string, args ...string) (string, error) {
				// gh pr view returns error when no PR
				return "", &mockError{}
			},
		}

		opts := Options{
			Dir:    tmpDir,
			Runner: mock,
		}

		err := Run(opts)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("with existing PR", func(t *testing.T) {
		tmpDir := t.TempDir()

		mock := &MockRunner{
			RunFunc: func(dir string, command string, args ...string) (string, error) {
				if command == "git" && len(args) >= 2 && args[0] == "rev-parse" && args[1] == "--abbrev-ref" {
					return "feature/test", nil
				}
				return "", nil
			},
			RunWithTimeoutFunc: func(dir string, timeout time.Duration, command string, args ...string) (string, error) {
				if command == "gh" && len(args) >= 2 && args[0] == "pr" && args[1] == "view" {
					return `{"number":42,"title":"Test PR","url":"https://github.com/test/repo/pull/42","state":"OPEN","mergeable":"MERGEABLE","baseRefName":"main","headRefName":"feature/test"}`, nil
				}
				if command == "gh" && len(args) >= 2 && args[0] == "pr" && args[1] == "checks" {
					return `[{"name":"test","status":"COMPLETED","conclusion":"SUCCESS"}]`, nil
				}
				return "", nil
			},
		}

		opts := Options{
			Dir:    tmpDir,
			Runner: mock,
		}

		err := Run(opts)
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("with failing checks", func(t *testing.T) {
		tmpDir := t.TempDir()

		mock := &MockRunner{
			RunFunc: func(dir string, command string, args ...string) (string, error) {
				if command == "git" && len(args) >= 2 && args[0] == "rev-parse" && args[1] == "--abbrev-ref" {
					return "feature/test", nil
				}
				return "", nil
			},
			RunWithTimeoutFunc: func(dir string, timeout time.Duration, command string, args ...string) (string, error) {
				if command == "gh" && len(args) >= 2 && args[0] == "pr" && args[1] == "view" {
					return `{"number":42,"title":"Test PR","url":"https://github.com/test/repo/pull/42","state":"OPEN","mergeable":"CONFLICTING","baseRefName":"main","headRefName":"feature/test"}`, nil
				}
				if command == "gh" && len(args) >= 2 && args[0] == "pr" && args[1] == "checks" {
					return `[{"name":"test","status":"COMPLETED","conclusion":"FAILURE"}]`, nil
				}
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
}
