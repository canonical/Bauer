package models

// WorkflowPost is the request body for triggering the PR-creation workflow.
//
// Secrets (service account credentials and the GitHub token) are never accepted
// in the request body. They are resolved server-side from configuration and the
// process environment.
type WorkflowPost struct {
	// DocID is the Google Doc ID to extract suggestions from.
	DocID string `json:"doc_id"`

	// GitHubRepo is the target repository ("owner/repo" or HTTPS URL).
	GitHubRepo string `json:"github_repo"`

	// BranchPrefix is the branch naming prefix. Defaults to "bauer" when empty.
	BranchPrefix string `json:"branch_prefix"`

	// PageRefresh enables page-refresh instruction mode.
	PageRefresh bool `json:"page_refresh"`

	// ParseOnly enables parse-only mode, which generates a json file
	// containing the parsed suggestions without creating a branch or PR.
	ParseOnly bool `json:"parse_only"`
}
