package prfix

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
	Number    int    `json:"number"`
	Title     string `json:"title"`
	URL       string `json:"url"`
	State     string `json:"state"`
	Mergeable string `json:"mergeable"`
	BaseRef   string `json:"baseRefName"`
	HeadRef   string `json:"headRefName"`
}

// CheckInfo holds information about a CI check
type CheckInfo struct {
	Name       string `json:"name"`
	Status     string `json:"status"`
	Conclusion string `json:"conclusion"`
	DetailsURL string `json:"detailsUrl"`
}

// ReviewInfo holds information about a PR review
type ReviewInfo struct {
	Author string `json:"author"`
	State  string `json:"state"`
	Body   string `json:"body"`
}

// ReviewAuthor holds author info for a review
type ReviewAuthor struct {
	Login string `json:"login"`
}

// ReviewComment holds information about a review comment
type ReviewComment struct {
	Author ReviewAuthor `json:"author"`
	Body   string       `json:"body"`
	Path   string       `json:"path"`
	Line   int          `json:"line"`
}

// Options configures the pr-fix command behavior
type Options struct {
	Dir     string               // Target directory (defaults to cwd)
	Verbose bool                 // Include full protocol details
	Runner  runner.CommandRunner // Command runner (defaults to runner.Default)
}

// Run executes the pr-fix command and returns the prompt to stdout
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

	// Get current branch
	branch := git.GetCurrentBranch(dir, r)
	if branch == "" {
		out.WriteString(fmt.Sprintf("# Fix PR Issues for %s\n\n", projectName))
		out.WriteString("⚠️ Could not determine current branch.\n")
		fmt.Print(out.String())
		return nil
	}

	// Get existing PR
	pr := getExistingPR(dir, branch, r)
	if pr == nil {
		out.WriteString(fmt.Sprintf("# Fix PR Issues for %s\n\n", projectName))
		out.WriteString("## No PR Found\n")
		out.WriteString(fmt.Sprintf("No pull request found for branch `%s`.\n\n", branch))
		out.WriteString("Create one first:\n")
		out.WriteString("```bash\n")
		out.WriteString("claude \"$(vibes pr)\"\n")
		out.WriteString("```\n")
		fmt.Print(out.String())
		return nil
	}

	// Get task context
	task := beads.DetectCurrentTask(dir, branch, r)
	task.ProjectName = projectName

	// Header
	out.WriteString(fmt.Sprintf("# Fix PR #%d Issues\n\n", pr.Number))

	// PR Status section
	out.WriteString("## PR Status\n")
	out.WriteString(fmt.Sprintf("- **PR**: #%d %s\n", pr.Number, pr.Title))
	out.WriteString(fmt.Sprintf("- **URL**: %s\n", pr.URL))
	out.WriteString(fmt.Sprintf("- **State**: %s\n", pr.State))
	out.WriteString(fmt.Sprintf("- **Branch**: %s → %s\n", pr.HeadRef, pr.BaseRef))

	// Mergeable status
	mergeStatus := getMergeableStatus(pr.Mergeable)
	out.WriteString(fmt.Sprintf("- **Mergeable**: %s\n", mergeStatus))

	// Task context
	if task.ID != "" {
		if task.Title != "" {
			out.WriteString(fmt.Sprintf("- **Task**: %s \"%s\"\n", task.ID, task.Title))
		} else {
			out.WriteString(fmt.Sprintf("- **Task**: %s\n", task.ID))
		}
	}
	out.WriteString("\n")

	// CI Checks section
	checks := getChecks(dir, pr.Number, r)
	failingChecks, passingChecks, pendingChecks := categorizeChecks(checks)

	out.WriteString("## CI Checks\n")
	if len(checks) == 0 {
		out.WriteString("No CI checks configured.\n")
	} else {
		// Summary line
		out.WriteString(fmt.Sprintf("- ✅ Passing: %d\n", len(passingChecks)))
		out.WriteString(fmt.Sprintf("- ❌ Failing: %d\n", len(failingChecks)))
		out.WriteString(fmt.Sprintf("- ⏳ Pending: %d\n", len(pendingChecks)))
		out.WriteString("\n")

		// Show failing checks in detail
		if len(failingChecks) > 0 {
			out.WriteString("### Failing Checks\n")
			out.WriteString("```\n")
			for _, check := range failingChecks {
				out.WriteString(fmt.Sprintf("❌ %s\n", check.Name))
				if check.DetailsURL != "" {
					out.WriteString(fmt.Sprintf("   %s\n", check.DetailsURL))
				}
			}
			out.WriteString("```\n")
		}

		// Show pending checks
		if len(pendingChecks) > 0 {
			out.WriteString("### Pending Checks\n")
			out.WriteString("```\n")
			for _, check := range pendingChecks {
				out.WriteString(fmt.Sprintf("⏳ %s\n", check.Name))
			}
			out.WriteString("```\n")
		}
	}
	out.WriteString("\n")

	// Reviews section
	reviews := getReviews(dir, pr.Number, r)
	comments := getReviewComments(dir, pr.Number, r)

	out.WriteString("## Reviews\n")
	if len(reviews) == 0 && len(comments) == 0 {
		out.WriteString("No reviews yet.\n")
	} else {
		// Show review states
		for _, review := range reviews {
			emoji := getReviewEmoji(review.State)
			out.WriteString(fmt.Sprintf("- %s **%s**: %s\n", emoji, review.Author, review.State))
		}

		// Show review comments
		if len(comments) > 0 {
			out.WriteString("\n### Review Comments\n")
			for _, comment := range comments {
				out.WriteString(fmt.Sprintf("\n**@%s** on `%s", comment.Author.Login, comment.Path))
				if comment.Line > 0 {
					out.WriteString(fmt.Sprintf(":%d", comment.Line))
				}
				out.WriteString("`:\n")
				// Indent the comment body
				lines := strings.Split(comment.Body, "\n")
				for _, line := range lines {
					out.WriteString(fmt.Sprintf("> %s\n", line))
				}
			}
		}
	}
	out.WriteString("\n")

	// Determine what needs to be fixed
	issues := determineIssues(pr, failingChecks, pendingChecks, reviews, comments)

	// Instructions section
	out.WriteString("## Issues to Address\n")
	if len(issues) == 0 {
		out.WriteString("✅ **No blocking issues found!**\n\n")
		out.WriteString("The PR looks ready to merge. You can:\n")
		out.WriteString("```bash\n")
		out.WriteString(fmt.Sprintf("gh pr merge %d\n", pr.Number))
		out.WriteString("```\n")
	} else {
		for i, issue := range issues {
			out.WriteString(fmt.Sprintf("%d. %s\n", i+1, issue))
		}
		out.WriteString("\n")
	}

	// Protocol
	out.WriteString("## Protocol\n")
	out.WriteString(getProtocol(pr, issues, opts.Verbose))

	fmt.Print(out.String())
	return nil
}

// getExistingPR checks if a PR already exists for the given branch
func getExistingPR(dir string, branch string, r runner.CommandRunner) *PRInfo {
	output, err := r.RunWithTimeout(dir, 10*time.Second, "gh", "pr", "view", "--json", "number,title,url,state,mergeable,baseRefName,headRefName")
	if err != nil || output == "" {
		return nil
	}

	var pr PRInfo
	if err := json.Unmarshal([]byte(output), &pr); err != nil {
		return nil
	}

	return &pr
}

// getChecks retrieves CI check status for the PR
func getChecks(dir string, prNumber int, r runner.CommandRunner) []CheckInfo {
	output, err := r.RunWithTimeout(dir, 10*time.Second, "gh", "pr", "checks", fmt.Sprintf("%d", prNumber), "--json", "name,status,conclusion,detailsUrl")
	if err != nil || output == "" {
		return nil
	}

	var checks []CheckInfo
	if err := json.Unmarshal([]byte(output), &checks); err != nil {
		return nil
	}

	return checks
}

// categorizeChecks separates checks into failing, passing, and pending
func categorizeChecks(checks []CheckInfo) (failing, passing, pending []CheckInfo) {
	for _, check := range checks {
		switch {
		case check.Status != "COMPLETED":
			pending = append(pending, check)
		case check.Conclusion == "SUCCESS" || check.Conclusion == "SKIPPED" || check.Conclusion == "NEUTRAL":
			passing = append(passing, check)
		default:
			failing = append(failing, check)
		}
	}
	return
}

// getReviews retrieves review information for the PR
func getReviews(dir string, prNumber int, r runner.CommandRunner) []ReviewInfo {
	output, err := r.RunWithTimeout(dir, 10*time.Second, "gh", "pr", "view", fmt.Sprintf("%d", prNumber), "--json", "reviews")
	if err != nil || output == "" {
		return nil
	}

	var result struct {
		Reviews []struct {
			Author struct {
				Login string `json:"login"`
			} `json:"author"`
			State string `json:"state"`
			Body  string `json:"body"`
		} `json:"reviews"`
	}
	if err := json.Unmarshal([]byte(output), &result); err != nil {
		return nil
	}

	var reviews []ReviewInfo
	for _, r := range result.Reviews {
		reviews = append(reviews, ReviewInfo{
			Author: r.Author.Login,
			State:  r.State,
			Body:   r.Body,
		})
	}
	return reviews
}

// getReviewComments retrieves review comments for the PR
func getReviewComments(dir string, prNumber int, r runner.CommandRunner) []ReviewComment {
	output, err := r.RunWithTimeout(dir, 10*time.Second, "gh", "pr", "view", fmt.Sprintf("%d", prNumber), "--json", "reviewRequests,comments")
	if err != nil || output == "" {
		// Try getting comments via the API
		output, err = r.RunWithTimeout(dir, 10*time.Second, "gh", "api", fmt.Sprintf("repos/{owner}/{repo}/pulls/%d/comments", prNumber))
		if err != nil || output == "" {
			return nil
		}
	}

	var comments []ReviewComment
	if err := json.Unmarshal([]byte(output), &comments); err != nil {
		// Try alternate format
		var altComments []struct {
			User struct {
				Login string `json:"login"`
			} `json:"user"`
			Body string `json:"body"`
			Path string `json:"path"`
			Line int    `json:"line"`
		}
		if err := json.Unmarshal([]byte(output), &altComments); err != nil {
			return nil
		}
		for _, c := range altComments {
			comments = append(comments, ReviewComment{
				Author: ReviewAuthor{Login: c.User.Login},
				Body:   c.Body,
				Path:   c.Path,
				Line:   c.Line,
			})
		}
	}

	return comments
}

// getMergeableStatus returns a human-readable mergeable status
func getMergeableStatus(mergeable string) string {
	switch strings.ToUpper(mergeable) {
	case "MERGEABLE":
		return "✅ Yes"
	case "CONFLICTING":
		return "❌ Conflicts"
	case "UNKNOWN":
		return "⏳ Checking..."
	default:
		return mergeable
	}
}

// getReviewEmoji returns an emoji for the review state
func getReviewEmoji(state string) string {
	switch strings.ToUpper(state) {
	case "APPROVED":
		return "✅"
	case "CHANGES_REQUESTED":
		return "❌"
	case "COMMENTED":
		return "💬"
	case "PENDING":
		return "⏳"
	case "DISMISSED":
		return "🚫"
	default:
		return "•"
	}
}

// determineIssues analyzes the PR state and returns a list of issues to address
func determineIssues(pr *PRInfo, failingChecks, pendingChecks []CheckInfo, reviews []ReviewInfo, comments []ReviewComment) []string {
	var issues []string

	// Merge conflicts
	if strings.ToUpper(pr.Mergeable) == "CONFLICTING" {
		issues = append(issues, "**Merge conflicts** - Resolve conflicts with the base branch")
	}

	// Failing CI
	if len(failingChecks) > 0 {
		checkNames := make([]string, len(failingChecks))
		for i, c := range failingChecks {
			checkNames[i] = c.Name
		}
		issues = append(issues, fmt.Sprintf("**CI failures** - Fix: %s", strings.Join(checkNames, ", ")))
	}

	// Changes requested
	for _, review := range reviews {
		if strings.ToUpper(review.State) == "CHANGES_REQUESTED" {
			issues = append(issues, fmt.Sprintf("**Changes requested** by @%s", review.Author))
			break // Only mention once
		}
	}

	// Review comments to address
	if len(comments) > 0 {
		issues = append(issues, fmt.Sprintf("**%d review comment(s)** to address", len(comments)))
	}

	// Pending checks (informational)
	if len(pendingChecks) > 0 && len(issues) == 0 {
		issues = append(issues, fmt.Sprintf("**%d check(s) still running** - Wait for completion", len(pendingChecks)))
	}

	return issues
}

func getProtocol(pr *PRInfo, issues []string, verbose bool) string {
	if len(issues) == 0 {
		// No issues - ready to merge
		if verbose {
			return fmt.Sprintf(`The PR is ready to merge!

1. **Final review** - Skim through changes one more time
2. **Merge the PR**:
   `+"```bash"+`
   gh pr merge %d --squash
   `+"```"+`
3. **Clean up** local branch:
   `+"```bash"+`
   git checkout main && git pull && git branch -d %s
   `+"```"+`

Proceed with merging when ready.
`, pr.Number, pr.HeadRef)
		}
		return fmt.Sprintf(`The PR is ready to merge!

1. Final review of changes
2. Merge: `+"`gh pr merge %d --squash`"+`
3. Clean up: `+"`git checkout main && git pull`"+`

Proceed with merging when ready.
`, pr.Number)
	}

	if verbose {
		return fmt.Sprintf(`1. **Investigate failures**:
   `+"```bash"+`
   gh pr checks %d
   gh pr view %d --comments
   `+"```"+`

2. **For merge conflicts**:
   `+"```bash"+`
   git fetch origin %s
   git rebase origin/%s
   # Resolve conflicts in each file
   git add <resolved-files>
   git rebase --continue
   git push --force-with-lease
   `+"```"+`

3. **For CI failures**:
   - Check the logs at the details URL
   - Fix the failing tests or linting issues
   - Commit and push the fixes

4. **For review comments**:
   - Address each comment
   - Reply to comments explaining changes
   - Request re-review if needed

5. **Push fixes and verify**:
   `+"```bash"+`
   git push
   gh pr checks %d --watch
   `+"```"+`

6. **When all checks pass**, run:
   `+"```bash"+`
   claude "$(vibes pr-fix)"
   `+"```"+`

Address the issues listed above.
`, pr.Number, pr.Number, pr.BaseRef, pr.BaseRef, pr.Number)
	}

	return fmt.Sprintf(`1. Investigate: `+"`gh pr checks %d`"+` and `+"`gh pr view %d --comments`"+`
2. For conflicts: rebase on %s and resolve
3. For CI failures: check logs, fix issues, push
4. For review comments: address and reply
5. Push fixes: `+"`git push`"+`
6. Re-check: `+"`claude \"$(vibes pr-fix)\"`"+`

Address the issues listed above.
`, pr.Number, pr.Number, pr.BaseRef)
}
