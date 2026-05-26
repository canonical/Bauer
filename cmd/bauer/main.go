package main

import (
	"bauer/internal/artifacts"
	"bauer/internal/config"
	"bauer/internal/copilotcli"
	"bauer/internal/gdocs"
	"bauer/internal/github"
	"bauer/internal/orchestrator"
	"bauer/internal/source"
	"context"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"
)

func main() {
	fs := flag.NewFlagSet(os.Args[0], flag.ExitOnError)

	docID := fs.String("doc-id", "", "Google Doc ID to extract feedback from (required, or set BAUER_DOC_ID)")
	credentialsPath := fs.String("credentials", "", "Path to service account credentials JSON\n\t(falls back to BAUER_CREDENTIALS_PATH \u2192 GOOGLE_APPLICATION_CREDENTIALS \u2192 credentials.json)")
	chunkSize := fs.Int("chunk-size", 0, "Total number of chunks (default: 1, or 5 if --page-refresh)")
	pageRefresh := fs.Bool("page-refresh", false, "Use page-refresh-instructions template (default chunk-size: 5)")
	model := fs.String("model", "", "Copilot model for sessions (default: gpt-5-mini-high)")
	summaryModel := fs.String("summary-model", "", "Copilot model for summary session (default: gpt-5-mini-high)")
	dryRun := fs.Bool("dry-run", false, "In standalone mode: skip Copilot, write chunk files only.\n\tIn --open-pr mode: apply changes locally, skip PR creation.")
	artifactsDir := fs.String("artifacts-dir", "", "Directory for run artifacts (default: ./bauer-artifacts)")
	openPR := fs.Bool("open-pr", false, "Apply changes and open a pull request (mutually exclusive with --open-issue)")
	openIssue := fs.Bool("open-issue", false, "Generate a plan and open a GitHub issue without applying changes (mutually exclusive with --open-pr)")
	branchPrefix := fs.String("branch-prefix", "", "Prefix for created branches (default: bauer)")
	githubRepo := fs.String("github-repo", "", "GitHub repository in owner/repo format (required for --open-pr and --open-issue)")
	figmaURL := fs.String("figma-url", "", "Figma file or design URL for design reference (requires BAUER_FIGMA_TOKEN)")

	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [flags]\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Flags:\n\n")
		fs.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nEnvironment variables:\n\n")
		fmt.Fprintf(os.Stderr, "\tBAUER_DOC_ID                    Override for --doc-id\n")
		fmt.Fprintf(os.Stderr, "\tBAUER_CREDENTIALS_PATH          Override for --credentials\n")
		fmt.Fprintf(os.Stderr, "\tGOOGLE_APPLICATION_CREDENTIALS  Fallback credentials path\n")
		fmt.Fprintf(os.Stderr, "\tBAUER_MODEL                     Override for --model\n")
		fmt.Fprintf(os.Stderr, "\tBAUER_SUMMARY_MODEL             Override for --summary-model\n")
		fmt.Fprintf(os.Stderr, "\tBAUER_DRY_RUN                   Override for --dry-run (true/false)\n")
		fmt.Fprintf(os.Stderr, "\tBAUER_ARTIFACTS_DIR             Override for --artifacts-dir\n")
		fmt.Fprintf(os.Stderr, "\tBAUER_BRANCH_PREFIX             Override for --branch-prefix\n")
		fmt.Fprintf(os.Stderr, "\tBAUER_GITHUB_REPO               Override for --github-repo\n")
		fmt.Fprintf(os.Stderr, "\tBAUER_FIGMA_TOKEN               Figma API token (required when --figma-url is supplied)\n")
	}

	if err := fs.Parse(os.Args[1:]); err != nil {
		os.Exit(1)
	}

	// Mutual exclusion check -- immediately after flag parsing, before any I/O or env resolution.
	if err := validateFlags(*openPR, *openIssue); err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}

	// Build CLIFlags -- *bool fields are only set for explicitly-provided flags
	// so they don't override env vars when the user didn't pass the flag.
	flags := config.CLIFlags{
		DocID:           *docID,
		CredentialsPath: *credentialsPath,
		ChunkSize:       *chunkSize,
		Model:           *model,
		SummaryModel:    *summaryModel,
		ArtifactsDir:    *artifactsDir,
		BranchPrefix:    *branchPrefix,
		GitHubRepo:      *githubRepo,
		FigmaURL:        *figmaURL,
	}
	fs.Visit(func(f *flag.Flag) {
		switch f.Name {
		case "dry-run":
			flags.DryRun = config.BoolPtr(*dryRun)
		case "page-refresh":
			flags.PageRefresh = config.BoolPtr(*pageRefresh)
		case "open-pr":
			flags.OpenPR = config.BoolPtr(*openPR)
		case "open-issue":
			flags.OpenIssue = config.BoolPtr(*openIssue)
		}
	})

	cfg, err := resolveCLIConfig(flags)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	cfg.ApplyDefaults()
	if err := cfg.Validate(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	ctx := context.Background()

	cwd, err := os.Getwd()
	if err != nil {
		fmt.Fprintln(os.Stderr, "ERROR: failed to get working directory:", err)
		os.Exit(1)
	}

	copilotAgent, err := copilotcli.NewClient(cwd)
	if err != nil {
		fmt.Fprintln(os.Stderr, "ERROR: failed to create Copilot client:", err)
		os.Exit(1)
	}

	sources := source.NewManager(cfg.CredentialsPath)
	arts := artifacts.NewManager(cfg.ArtifactsDir)
	orch := orchestrator.New(copilotAgent, sources, arts)

	switch {
	case *openIssue:
		if err := runOpenIssue(ctx, cfg, orch); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	case *openPR:
		if err := runOpenPR(ctx, cfg, orch, cwd); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	default:
		if _, err := orch.Execute(ctx, cfg); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	}
}

// validateFlags checks mutual exclusion and other flag constraints.
// Called immediately after flag parsing, before any I/O or env resolution.
func validateFlags(openPR, openIssue bool) error {
	if openPR && openIssue {
		return fmt.Errorf("--open-pr and --open-issue are mutually exclusive\n  Use --open-pr to apply changes and open a PR.\n  Use --open-issue to generate a plan and open an issue without applying changes.")
	}
	return nil
}

// resolveCLIConfig builds a Config from CLI flags, falling back to environment variables
// and then hardcoded defaults. FlagsSource has highest priority.
func resolveCLIConfig(flags config.CLIFlags) (*config.Config, error) {
	return config.NewResolver(
		config.NewFlagsSource(flags),
		config.NewEnvVarSource(),
		config.NewDefaultsSource(),
	).Resolve()
}

// openPRExecutionConfig returns a copy of cfg with DryRun disabled.
func openPRExecutionConfig(original *config.Config) *config.Config {
	copy := *original
	copy.DryRun = config.BoolPtr(false)
	return &copy
}

// runOpenIssue generates a documentation improvement plan and opens a GitHub issue.
// It runs the orchestrator in dry-run mode (extraction + prompt generation only, no Copilot).
func runOpenIssue(ctx context.Context, cfg *config.Config, orch orchestrator.Orchestrator) error {
	token, err := github.GetGitHubToken()
	if err != nil {
		return fmt.Errorf("GitHub token not found: %w\nSet BAUER_GITHUB_TOKEN, GITHUB_TOKEN, or GH_TOKEN, or run 'gh auth login'", err)
	}

	if cfg.GitHubRepo == "" {
		return fmt.Errorf("--github-repo (or BAUER_GITHUB_REPO) is required for --open-issue mode")
	}

	// Run with dry-run=true: extract + generate prompts, but skip Copilot.
	issueCfg := *cfg
	issueCfg.DryRun = config.BoolPtr(true)

	result, err := orch.Execute(ctx, &issueCfg)
	if err != nil {
		return fmt.Errorf("orchestration failed: %w", err)
	}

	if result.ExtractionBundle == nil || result.ExtractionBundle.Document == nil {
		return fmt.Errorf("no document data returned by orchestrator")
	}
	doc := result.ExtractionBundle.Document

	title := fmt.Sprintf("docs: %s \u2014 documentation suggestions review", doc.DocumentTitle)
	body := buildIssueBody(doc, cfg, result.RunID)

	issueURL, err := github.CreateIssue(ctx, token, cfg.GitHubRepo, title, body)
	if err != nil {
		return fmt.Errorf("failed to create GitHub issue: %w", err)
	}

	fmt.Printf("GitHub issue created: %s\n", issueURL)
	return nil
}

// buildIssueBody constructs the markdown body for the documentation suggestions issue.
func buildIssueBody(doc *gdocs.ProcessingResult, cfg *config.Config, runID string) string {
	var sb strings.Builder

	docURL := fmt.Sprintf("https://docs.google.com/document/d/%s", doc.DocumentID)
	generated := time.Now().UTC().Format(time.RFC3339)

	type suggEntry struct {
		Section    string
		Brief      string
		ChangeType string
	}
	var copyChanges, designChanges []suggEntry

	for _, loc := range doc.GroupedSuggestions {
		section := loc.Location.ParentHeading
		if section == "" {
			section = loc.Location.Section
			if section == "" {
				section = "Document"
			}
		}
		for _, s := range loc.Suggestions {
			brief := s.Change.NewText
			if len(brief) > 100 {
				brief = brief[:97] + "..."
			}
			if brief == "" {
				brief = s.Change.OriginalText
				if len(brief) > 100 {
					brief = brief[:97] + "..."
				}
			}
			entry := suggEntry{Section: section, Brief: brief, ChangeType: s.Change.Type}
			if s.Change.Type == "insert" {
				designChanges = append(designChanges, entry)
			} else {
				copyChanges = append(copyChanges, entry)
			}
		}
	}

	totalSuggestions := len(copyChanges) + len(designChanges)
	sectionCount := len(doc.GroupedSuggestions)

	fmt.Fprintf(&sb, "## Documentation Suggestions Review\n\n")
	fmt.Fprintf(&sb, "Source: [Google Doc](%s)\n", docURL)
	fmt.Fprintf(&sb, "Generated: %s\n", generated)
	if runID != "" {
		fmt.Fprintf(&sb, "Run ID: `%s`\n", runID)
	}
	fmt.Fprintf(&sb, "\n### Summary\n\n")
	fmt.Fprintf(&sb, "%d suggestion(s) extracted from the document across %d section(s).\n", totalSuggestions, sectionCount)
	fmt.Fprintf(&sb, "\n### Suggestions by Type\n\n")
	fmt.Fprintf(&sb, "**Copy changes** (%d):\n", len(copyChanges))
	if len(copyChanges) == 0 {
		fmt.Fprintf(&sb, "- _(none)_\n")
	}
	for _, c := range copyChanges {
		if c.Brief != "" {
			fmt.Fprintf(&sb, "- Section \"%s\": %s\n", c.Section, c.Brief)
		} else {
			fmt.Fprintf(&sb, "- Section \"%s\": (%s change)\n", c.Section, c.ChangeType)
		}
	}
	fmt.Fprintf(&sb, "\n**Design/content additions** (%d):\n", len(designChanges))
	if len(designChanges) == 0 {
		fmt.Fprintf(&sb, "- _(none)_\n")
	}
	for _, c := range designChanges {
		if c.Brief != "" {
			fmt.Fprintf(&sb, "- Section \"%s\": %s\n", c.Section, c.Brief)
		} else {
			fmt.Fprintf(&sb, "- Section \"%s\": (insertion)\n", c.Section)
		}
	}
	if cfg.FigmaURL != "" {
		fmt.Fprintf(&sb, "\n### Design Reference\n\n")
		fmt.Fprintf(&sb, "Figma: %s\n", cfg.FigmaURL)
	}
	fmt.Fprintf(&sb, "\n### Next Steps\n\n")
	fmt.Fprintf(&sb, "Review these suggestions, then run:\n\n")
	fmt.Fprintf(&sb, "```sh\n")
	fmt.Fprintf(&sb, "bauer --doc-id %s --open-pr --github-repo %s\n", doc.DocumentID, cfg.GitHubRepo)
	fmt.Fprintf(&sb, "```\n")
	fmt.Fprintf(&sb, "\nto apply them automatically via Copilot.\n")

	return sb.String()
}

// runOpenPR runs Copilot to apply documentation changes, then creates a branch and opens a PR.
func runOpenPR(ctx context.Context, cfg *config.Config, orch orchestrator.Orchestrator, repoDir string) error {
	token, err := github.GetGitHubToken()
	if err != nil {
		return fmt.Errorf("GitHub token not found: %w\nSet BAUER_GITHUB_TOKEN, GITHUB_TOKEN, or GH_TOKEN, or run 'gh auth login'", err)
	}

	if cfg.GitHubRepo == "" {
		return fmt.Errorf("--github-repo (or BAUER_GITHUB_REPO) is required for --open-pr mode")
	}

	repo, err := github.ParseGitHubRepo(cfg.GitHubRepo)
	if err != nil {
		return fmt.Errorf("invalid --github-repo %q: %w", cfg.GitHubRepo, err)
	}

	// Set token in environment so that gh CLI can authenticate.
	if err := github.SetupGitHubAuth(token); err != nil {
		return fmt.Errorf("failed to configure GitHub auth: %w", err)
	}

	// Run Copilot (disable dry-run for the execution phase).
	execCfg := openPRExecutionConfig(cfg)
	result, err := orch.Execute(ctx, execCfg)
	if err != nil {
		return fmt.Errorf("orchestration failed: %w", err)
	}

	// Determine branch name from the artifact run ID.
	branchPrefix := cfg.BranchPrefix
	if branchPrefix == "" {
		branchPrefix = "bauer"
	}
	runID := result.RunID
	if runID == "" {
		runID = time.Now().UTC().Format("2006-01-02T15-04-05Z")
	}
	branchName := branchPrefix + "/" + runID

	// Create the new branch.
	if _, err := github.RunGit(ctx, repoDir, "checkout", "-b", branchName); err != nil {
		return fmt.Errorf("failed to create branch %q: %w", branchName, err)
	}

	// Stage all changes.
	if _, err := github.RunGit(ctx, repoDir, "add", "-A"); err != nil {
		return fmt.Errorf("failed to stage changes: %w", err)
	}

	// Commit. If there is nothing to commit, report and exit cleanly.
	commitMsg := "docs(bauer): apply documentation suggestions"
	out, err := github.RunGit(ctx, repoDir, "commit", "-m", commitMsg)
	if err != nil {
		if strings.Contains(out, "nothing to commit") {
			fmt.Println("No changes to commit. Exiting.")
			return nil
		}
		return fmt.Errorf("failed to commit: %w", err)
	}

	// Push the branch.
	if _, err := github.RunGit(ctx, repoDir, "push", "origin", branchName); err != nil {
		return fmt.Errorf("failed to push branch %q: %w", branchName, err)
	}

	// Create the pull request.
	prBody := buildPRBody(result, branchName)
	prURL, err := github.CreatePR(repo.Owner, repo.Name, github.CreatePROptions{
		Title:      "docs: apply documentation suggestions from Copilot",
		Body:       prBody,
		BaseBranch: "main",
		HeadBranch: branchName,
	})
	if err != nil {
		return fmt.Errorf("failed to create PR: %w", err)
	}

	fmt.Printf("Pull request created: %s\n", prURL)
	return nil
}

// buildPRBody constructs the markdown body for the documentation suggestions PR.
func buildPRBody(result *orchestrator.OrchestrationResult, branchName string) string {
	var sb strings.Builder

	fmt.Fprintf(&sb, "## Documentation Suggestions \u2014 Automated Apply\n\n")
	fmt.Fprintf(&sb, "Applied by: Bauer + GitHub Copilot\n")
	fmt.Fprintf(&sb, "Branch: `%s`\n", branchName)
	if result.RunID != "" {
		fmt.Fprintf(&sb, "Run ID: `%s`\n", result.RunID)
	}
	fmt.Fprintf(&sb, "Timestamp: %s\n", time.Now().UTC().Format(time.RFC3339))

	if result.ExtractionBundle != nil && result.ExtractionBundle.Document != nil {
		doc := result.ExtractionBundle.Document
		docURL := fmt.Sprintf("https://docs.google.com/document/d/%s", doc.DocumentID)
		fmt.Fprintf(&sb, "\n### Source Document\n\n")
		fmt.Fprintf(&sb, "[%s](%s)\n", doc.DocumentTitle, docURL)
		fmt.Fprintf(&sb, "\n%d suggestion(s) from %d section(s) were applied.\n",
			countAllSuggestions(doc), len(doc.GroupedSuggestions))
	}

	if len(result.CopilotOutputs) > 0 {
		fmt.Fprintf(&sb, "\n### Copilot Execution Summary\n\n")
		fmt.Fprintf(&sb, "%d chunk(s) processed in %s.\n",
			len(result.CopilotOutputs), result.CopilotDuration.Round(time.Second))
	}

	if result.Summary != "" {
		fmt.Fprintf(&sb, "\n### Summary\n\n")
		fmt.Fprintf(&sb, "%s\n", result.Summary)
	}

	return sb.String()
}

// countAllSuggestions returns the total number of suggestions across all location groups.
func countAllSuggestions(doc *gdocs.ProcessingResult) int {
	total := 0
	for _, loc := range doc.GroupedSuggestions {
		total += len(loc.Suggestions)
	}
	return total
}
