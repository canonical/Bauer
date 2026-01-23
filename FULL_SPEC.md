# BAU CLI - Full Technical Specification

> **Document Version:** 1.2  
> **Status:** Draft  
> **Last Updated:** 2026-01-21

## Table of Contents

- [Executive Summary](#executive-summary)
- [Requirements Analysis](#requirements-analysis)
  - [R1: CLI Interface](#r1-cli-interface)
  - [R2: Credentials Management](#r2-credentials-management)
  - [R3: Technical Plan Generation](#r3-technical-plan-generation)
  - [R4: GitHub Copilot Integration (via SDK)](#r4-github-copilot-integration-via-sdk)
  - [R5: Git Change Detection](#r5-git-change-detection)
  - [R6: Pull Request Creation](#r6-pull-request-creation)
- [Architecture Design](#architecture-design)
- [POC Cleanup Integration](#poc-cleanup-integration)
- [Template Files Specification](#template-files-specification)
- [Copilot SDK Configuration](#copilot-sdk-configuration)
- [Security Considerations](#security-considerations)
- [Learning Resources](#learning-resources)
- [Detailed Implementation Plan](#detailed-implementation-plan)
- [Risk Assessment](#risk-assessment)
- [Missing Details & Assumptions](#missing-details--assumptions)
- [Phased Implementation](#phased-implementation)
- [Testing Strategy](#testing-strategy)
- [Appendices](#appendices)

---

## Executive Summary

This document specifies the transformation of the Google Docs Content & Feedback Extraction POC into a full-featured CLI tool called **BAU** (Build Automation Utility). The tool will:

1. Extract suggestions and comments from a Google Doc via CLI arguments
2. Generate a technical implementation plan from the extracted feedback
3. Use the GitHub Copilot SDK (programmatic SDK for Copilot CLI) to implement the plan automatically
4. Detect changes and create a Pull Request on GitHub

The goal is to create an end-to-end automation pipeline that takes document feedback and produces implemented code changes with minimal human intervention. The integration with Copilot is implemented using the official Copilot SDK for Go (github.com/github/copilot-sdk/go) — BAU will depend on this SDK to create and manage sessions, stream output, and expose tools to Copilot.

---

## Requirements Analysis

### R1: CLI Interface

#### Description

The program should run as a CLI tool, accepting the Google Doc ID as an argument rather than using hardcoded values.

#### Approaches

| Approach                       | Description                                         | Pros                                                                                        | Cons                                                            |
| ------------------------------ | --------------------------------------------------- | ------------------------------------------------------------------------------------------- | --------------------------------------------------------------- |
| **A. Standard `flag` package** | Use Go's built-in flag package for argument parsing | - No external dependencies<br>- Simple to implement<br>- Familiar to Go developers          | - Limited features (no subcommands)<br>- Manual help formatting |
| **B. Cobra library**           | Use spf13/cobra for CLI framework                   | - Industry standard<br>- Subcommand support<br>- Auto-generated help<br>- Shell completions | - External dependency<br>- More boilerplate for simple CLIs     |

#### Recommendation: **Approach A (Standard `flag` package)**

**Rationale:**

- Simple requirements that don't yet justify the complexity of a full CLI framework.
- Standard library keeps dependencies minimal.
- Easy to upgrade to Cobra later if subcommands are needed.

#### Proposed CLI Interface

```bash
# Basic usage
bau --doc-id "1b9F1Av8tRNG8xkPHgjvtBKrQogXRDaRb0Lw7pEZxr9I" --credentials ./creds.json

# Dry run (skips Copilot execution and PR creation)
bau \
  --doc-id "1b9F1Av8tRNG8xkPHgjvtBKrQogXRDaRb0Lw7pEZxr9I" \
  --credentials ./creds.json \
  --dry-run
```

#### CLI Flags Specification (Updated January 2025)

**Current Flags (Phase 1 & 2):**

```
Required:
  --doc-id string          Google Doc ID to extract feedback from
  --credentials string     Path to service account JSON file

Optional:
  --dry-run               Skip Copilot execution and PR creation (default: true)
  --chunk-size int        Maximum locations per chunk (default: 10)
  --output-dir string     Output directory for prompts (default: "bauer-output")
```

**Removed Flags:**
- ❌ `--target-repo` (user runs from repo directory)
- ❌ `--target-branch` (user checks out correct branch)

**Original Specification:**

| Flag            | Short | Type   | Required | Default    | Description                                                    |
| --------------- | ----- | ------ | -------- | ---------- | -------------------------------------------------------------- |
| `--doc-id`      | `-d`  | string | Yes      | -          | Google Doc ID to extract feedback from                         |
| `--credentials` | `-c`  | string | Yes      | -          | Path to service account JSON                                   |
| `--dry-run`     | `-n`  | bool   | No       | false      | Run extraction and planning only; skip Copilot and PR creation |
| `--template`    | -     | string | No       | -          | Custom template file for technical plan                        |
| `--output`      | -     | string | No       | `./output` | Output directory                                               |
| `--model-name`  | -     | string | No       | `gpt-5`    | Copilot model to use for sessions                              |
| `--chunk-size`  | -     | int    | No       | `30`       | Number of GroupedSuggestions sent per chunk (CHUNK_SIZE)       |

---

### R2: Credentials Management

#### Description

How should users provide Google Cloud service account credentials to the tool?

#### Approaches

| Approach                                | Description                              | Pros                                                     | Cons                                                                 |
| --------------------------------------- | ---------------------------------------- | -------------------------------------------------------- | -------------------------------------------------------------------- |
| **A. User-specified path only**         | Require `--credentials` flag every time  | - Explicit<br>- No magic<br>- Flexible                   | - Tedious for repeated use<br>- No sensible default                  |
| **B. Environment variable**             | Use `GOOGLE_CREDENTIALS_PATH` env var    | - Standard practice<br>- Works in CI/CD<br>- Secure      | - Must be set up<br>- Can be forgotten                               |
| **C. Default location**                 | Look in `~/.config/bau/credentials.json` | - Convenient<br>- Follows XDG spec<br>- Set once, forget | - Security concerns if not secured<br>- User might not know location |
| **D. Multiple sources with precedence** | Try multiple locations in order          | - Most flexible<br>- Best UX<br>- Covers all use cases   | - More complex logic<br>- Might be confusing                         |

#### Recommendation: **Approach A (User-specified path only)**

**Rationale:**

- Explicit and simple for MVP.
- Avoids complexity of precedence logic.

#### Future Potential Improvements

- **Environment variable support:** `GOOGLE_CREDENTIALS_PATH` or `GOOGLE_APPLICATION_CREDENTIALS`.
- **Default location:** `~/.config/bau/credentials.json`.
- **Precedence logic:** Combining flags, env vars, and default paths.

#### Credentials Resolution

```
1. --credentials flag (Required)
2. Error: No credentials provided or file not found
```

#### Security Considerations

```go
// Validate credentials file permissions on Unix systems
func validateCredentialsPermissions(path string) error {
    info, err := os.Stat(path)
    if err != nil {
        return err
    }

    // Warn if file is world-readable
    mode := info.Mode()
    if mode&0044 != 0 {
        slog.Warn("Credentials file has insecure permissions",
            "path", path,
            "mode", mode.String(),
            "recommended", "0600")
    }
    return nil
}
```

#### First-Run Setup Experience

```bash
$ bau --doc-id "..."
Error: No credentials found.

To set up credentials:
  1. Create a service account at https://console.cloud.google.com/iam-admin/serviceaccounts
  2. Download the JSON key file
  3. Save it to one of these locations:
     - ~/.config/bau/credentials.json (recommended)
     - Set GOOGLE_CREDENTIALS_PATH environment variable
     - Pass --credentials /path/to/file.json
  4. Ensure the service account has manual access to the document.

For detailed instructions, see: https://github.com/canonical/bau#authentication
```

---

### R3: Technical Plan Generation

#### Description

After extracting feedback from the Google Doc, the tool should generate a detailed technical implementation plan that can be consumed by the Copilot SDK (and therefore Copilot CLI).

#### Target File Resolution

The tool uses multiple sources to determine which file(s) Copilot should modify:

**Resolution Order:**

1. **Metadata table `Page path` field** - Primary source from the Google Doc's metadata table
2. **Metadata table `Page title` field** - Used to search for files with matching names
3. **Document title** - Fallback, parsed to extract potential file paths
4. **First H1 heading** - Additional fallback if title doesn't contain path info

**Implementation:**

```go
type TargetFileInfo struct {
    PrimaryPath      string   `json:"primary_path"`       // From metadata table
    AlternativePaths []string `json:"alternative_paths"`  // From title/headings
    SearchPatterns   []string `json:"search_patterns"`    // Glob patterns to try
    Confidence       string   `json:"confidence"`         // "high", "medium", "low"
}

func resolveTargetFile(metadata *MetadataTable, docTitle string, firstHeading string) *TargetFileInfo {
    info := &TargetFileInfo{}

    // Check metadata table for explicit path
    if metadata != nil && metadata.Raw["Page path"] != "" {
        info.PrimaryPath = metadata.Raw["Page path"]
        info.Confidence = "high"
    } else if metadata != nil && metadata.PageTitle != "" {
        // Use page title to generate search patterns
        info.SearchPatterns = append(info.SearchPatterns,
            fmt.Sprintf("**/*%s*", sanitizeForGlob(metadata.PageTitle)))
        info.Confidence = "medium"
    }

    // Add alternatives from document title
    if path := extractPathFromTitle(docTitle); path != "" {
        info.AlternativePaths = append(info.AlternativePaths, path)
    }

    // Add alternatives from first heading
    if path := extractPathFromTitle(firstHeading); path != "" {
        info.AlternativePaths = append(info.AlternativePaths, path)
    }

    if info.Confidence == "" {
        info.Confidence = "low"
    }

    return info
}
```

#### Approaches

| Approach                                 | Description                                     | Pros                                                                 | Cons                                                             |
| ---------------------------------------- | ----------------------------------------------- | -------------------------------------------------------------------- | ---------------------------------------------------------------- |
| **A. Static template with placeholders** | Use Go text/template with extracted data        | - Simple to implement<br>- Easy to customize<br>- Predictable output | - Limited flexibility<br>- Template maintenance                  |
| **B. LLM-generated plan**                | Call an LLM API to generate the plan            | - More intelligent<br>- Context-aware<br>- Better prose              | - External dependency<br>- Cost<br>- Latency<br>- Variability    |
| **C. Structured JSON output**            | Output machine-readable JSON for agents         | - Precise<br>- Deterministic<br>- Easy to parse                      | - Less readable for humans<br>- Copilot prefers natural language |
| **D. Hybrid: Template + embedded JSON**  | Markdown template with embedded structured data | - Best of both worlds<br>- Human and machine readable<br>- Flexible  | - More complex templates<br>- Two formats to maintain            |

#### Recommendation: **Approach D (Hybrid: Template + embedded JSON)**

**Rationale:**

- Copilot SDK works best with natural language prompts and attachments
- Embedded JSON provides precision for specific changes
- Humans can review the plan before execution
- Templates can be versioned and customized per project

#### Template System Design

**Template Location Resolution:**

```
1. --template flag
2. BAU_TEMPLATE_PATH environment variable
3. ./.bau/templates/default.md (project-specific)
4. ~/.config/bau/templates/default.md (user-wide)
5. Embedded default template (compiled into binary)
```

**Template Embedding (Go 1.16+ embed):**

```go
import "embed"

//go:embed templates/*.md
var embeddedTemplates embed.FS

func loadTemplate(customPath string) (*template.Template, error) {
    // Try custom path first
    if customPath != "" {
        return template.ParseFiles(customPath)
    }

    // Try project-specific
    if _, err := os.Stat(".bau/templates/default.md"); err == nil {
        return template.ParseFiles(".bau/templates/default.md")
    }

    // Fall back to embedded
    content, err := embeddedTemplates.ReadFile("templates/default.md")
    if err != nil {
        return nil, err
    }
    return template.New("default").Parse(string(content))
}
```

**Template Data Model:**

```go
type TemplatePlanData struct {
    // Document metadata
    DocumentTitle    string    `json:"document_title"`
    DocumentID       string    `json:"document_id"`
    DocumentURL      string    `json:"document_url"`
    ExtractionTime   time.Time `json:"extraction_time"`

    // Target file information
    TargetFile       *TargetFileInfo `json:"target_file"`

    // Metadata table (if present)
    Metadata         *MetadataTable `json:"metadata,omitempty"`

    // Extracted feedback
    Suggestions      []ActionableSuggestion `json:"suggestions"`
    SuggestionCount  int                    `json:"suggestion_count"`
    Comments         []Comment              `json:"comments"`
    CommentCount     int                    `json:"comment_count"`

    // Grouped by type for easier template iteration
    Insertions       []ActionableSuggestion `json:"insertions"`
    Deletions        []ActionableSuggestion `json:"deletions"`
    Replacements     []ActionableSuggestion `json:"replacements"`
    StyleChanges     []ActionableSuggestion `json:"style_changes"`

    // Target context
    TargetRepo       string `json:"target_repo"`
    TargetBranch     string `json:"target_branch"`

    // Configuration
    Config           TemplateConfig `json:"config"`
}

type TemplateConfig struct {
    IncludeRawJSON      bool   `json:"include_raw_json"`
    IncludeVerification bool   `json:"include_verification"`
    VerboseAnchors      bool   `json:"verbose_anchors"`
}
```

#### Output Files Generated

```
<output-dir>/
├── extraction.json           # Raw extracted data
├── technical-plan.md         # Generated implementation plan
├── suggestions-detailed.json # Detailed suggestion data (includes grouped suggestions)
├── copilot-prompt.md         # Copilot-optimized prompt (attachment-friendly)
└── execution-log.txt         # Log of operations (after execution)
```

---

### Chunking Location-Grouped Suggestions (ProcessingResult)

BAU will consume the new `LocationGroupedSuggestions` collection produced by the extraction/processing pipeline (returned as `ProcessingResult.GroupedSuggestions` in `internal/gdocs/types.go`). This structure first groups suggestions by their location in the document (section, heading, table), then within each location, groups atomic operations by suggestion ID. These location-based groups are the primary units we will batch and feed to Copilot sessions.

Why chunking is required

- Large documents and many suggestions may exceed Copilot model/session context or practical token limits.
- Sending every suggestion in a single, monolithic prompt reduces performance and increases risk of truncation or context loss.
- Chunking enables incremental, auditable application of changes, easier verification, and partial rollback on failure.

Overview of the chunking design

1. Input: `ProcessingResult.GroupedSuggestions []LocationGroupedSuggestions` plus `DocumentStructure` and `extraction.json`. Each `LocationGroupedSuggestions` contains:
   - `Location`: Contextual metadata (section, parent heading, table info)
   - `Suggestions`: Array of `GroupedActionableSuggestion` for that location
2. Partitioning: Location groups are natural chunks - each location group can be processed as a unit, or multiple location groups can be batched together using heuristics (see below).
3. For each chunk:
   - Build a chunk-level payload (attachments + concise prompt).
   - Send the payload to Copilot via SDK session (non-interactive or interactive per `--interactive`).
   - Wait for assistant to finish and collect events (streaming deltas + final message).
   - Verify the resulting changes locally (using verification.text_after_change and tools).
   - Commit changes into a working branch (if verification passes) or report/abort on errors.
4. Continue to next chunk; optionally resume the same session or create/resume sessions as needed.

Chunking heuristics (recommended)

- Primary grouping: **by location** - suggestions are pre-grouped by their document location (section, heading, table). Each `LocationGroupedSuggestions` represents a natural semantic unit.
- Secondary grouping: by target file path (if suggestions from multiple locations map to the same file, they can be batched together).
- Size cap: limit the number of location groups or estimated tokens per chunk to keep within model context (e.g., 2–5k tokens target per chunk; configurable).
- Location group integrity: **never split a single `LocationGroupedSuggestions` across chunks** - all suggestions within a location should be processed together as they share context.
- Atomic suggestion integrity: never split a single `GroupedActionableSuggestion` within a location (each grouped suggestion represents merged atomic operations).
- Natural ordering: location groups are already sorted by document position, making sequential processing straightforward.

Constructing the chunk payloads: prompt + attachments
Two viable payload strategies — trade-offs discussed below.

A. Full-template-per-chunk (preferred for simplicity and human-readability)

- For each chunk, render the full `technical-plan.md` template but include only the subset of location groups that belong to the chunk (with their contained suggestions).
- Attach `extraction.json` and `suggestions-detailed.json` (or the full `processing_result.json`) so the assistant can reference full context if needed.
- Advantages:
  - Copilot receives the full plan context each time, reducing ambiguity.
  - Easier for a human reviewer to inspect a single `technical-plan.md` per chunk.
  - Simpler implementation: reuse templating code, swap `Suggestions` slice.
- Disadvantages:
  - Re-sending repeated plan text increases token usage across chunks.
  - Requires careful token budgeting—must ensure repeated content does not push chunks over limits.

B. Minimal-chunk-prompt + context attachments (preferred for token efficiency)

- For each chunk, send a concise prompt that references the full plan via attachment (attached once at session start or included with each chunk) and enumerates only the chunk's location groups and their suggestions inline.
- Attach metadata/verification JSON for only the chunk (smaller attachments).
- Advantages:
  - Lower repeated token cost; more efficient for many or large plans.
  - Assistant can request additional context or call a tool to fetch more details when needed.
- Disadvantages:
  - Requires careful session state management (assistant must be able to reference attachments).
  - Slightly more complex templating: separate the global plan header from per-chunk payloads.

Recommendation

- Start with the Full-template-per-chunk approach for MVP because of simpler implementation and clearer auditability. Provide a config flag to switch to Minimal-chunk-prompt later.
- Always attach `suggestions-detailed.json` (or the chunk-level JSON) so the SDK tools or assistant can call structured tools rather than rely purely on prompt parsing.

Session & tool strategy

- Session lifetime:
  - Option 1: Use a single long-lived session and send chunk messages sequentially.
    - Pros: session retains conversation context; assistant can reference prior chunks.
    - Cons: session-level token accumulation; potential drift and higher memory/connection requirements.
  - Option 2: Use short sessions per chunk (create → send → destroy).
    - Pros: predictable token windows and isolation between chunks; easier to parallelize.
    - Cons: no conversational state across chunks; assistant cannot rely on previous replies unless attachments are shared.
- Recommended approach: single session with the ability to "reset" or resume (use `ResumeSessionWithOptions` and `SessionConfig` tools), or a hybrid where related chunks (same file) use one session and independent files use separate sessions.
- Tools:
  - Expose `check_file_exists`, `read_file_segment`, and `preview_patch` via `copilot.DefineTool`.
  - Tools allow the assistant to validate anchors precisely and request only the file ranges it needs.
  - Use a `apply_patch` tool shim internally (not exposed for full arbitrary shell execution) that returns a structured result; BAU then applies the changes locally only after verifying tool results and user consent (if interactive).

Verification and commit workflow

- After a chunk is processed by Copilot:
  - BAU should compute a dry-run diff (or use the `preview_patch` tool response) and validate against `Suggestion.Verification.TextAfterChange`. If verification fails, BAU should:
    - In interactive mode: show diffs to the user and ask for guidance (retry/skip/manual).
    - In non-interactive mode: abort the chunk, write diagnostics, and stop processing further chunks.
  - On verification success, stage and commit changes on the feature branch (commit message includes chunk metadata and suggestion IDs).
  - Optionally push after each chunk or push once after all chunks — configurable (`--commit-per-chunk` flag).

Open questions and considerations

1. Token budgeting & chunk sizing
   - How to estimate tokens reliably from suggestion content and template text? Implement a heuristic (character→token ratio) or integrate a lightweight tokenizer to estimate before sending.
2. Session persistence vs isolation
   - Do we prefer a single session (stateful) or stateless sessions per chunk? The former enables assistant continuity; the latter improves predictability. Hybrid strategies (per-file sessions) are a reasonable compromise.
3. Ordering and inter-chunk conflicts
   - If chunks target overlapping ranges or affect the same file in ways that conflict, how should BAU resolve ordering? Conservative approach: order by document position and detect overlaps; fail the run when ambiguous and surface to user.
4. Idempotency and retries
   - If a chunk partially applies (e.g., session times out), BAU must be able to resume or roll back. Use feature-branch commits per-chunk to make rollbacks straightforward.
5. Tool trust model
   - What tools do we expose to Copilot and which require explicit user consent? Prefer read-only tools by default; require interactive confirmation to expose write tools or apply patches automatically.
6. Verification strictness
   - How strict should verification be? Exact string equality is safe but brittle; fuzzy matching increases robustness but risks false positives. Make verification mode configurable (`strict`, `lenient`, `manual-review`).
7. Parallelism
   - Can independent chunks be processed in parallel? Yes if they target different files, but be careful about git commits and merge order. Prefer sequential processing for single-file changes.
8. Testing strategy
   - Create fixtures that simulate multiple chunk scenarios, overlapping edits, and tool failures. Add integration tests that run the SDK locally and validate commit outcomes.

Implementation checklist (practical)

- [ ] Add `ProcessingResult.GroupedSuggestions` (type: `[]LocationGroupedSuggestions`) consumers in `internal/planner` and `internal/copilot`.
- [ ] Implement chunker utility with configurable heuristics (by file, by token estimate).
- [ ] Render per-chunk templates (full-template mode) and attach chunk JSON files for each run.
- [ ] Implement `check_file_exists`, `read_file_segment`, `preview_patch` tools in `internal/copilot/tools.go`.
- [ ] Implement chunk-level send/verify/commit loop in `internal/copilot/executor.go`.
- [ ] Add flags: `--commit-per-chunk`, `--chunk-token-limit`, `--chunk-mode={full,minimal}`, `--verification-mode`.
- [ ] Add tests covering chunking, verification failures, and recovery.

Summary

- BAU will take `LocationGroupedSuggestions` (from `ProcessingResult.GroupedSuggestions`) and batch them into chunks before sending to Copilot via the SDK.
- Each location group contains suggestions that share the same document context (section, heading, table), making them ideal semantic units for processing.
- For MVP, BAU should send a full plan template per chunk (simpler, auditable), attach `suggestions-detailed.json`, stream the assistant response, verify changes, and commit per-chunk.
- There are several open questions around token budgeting, session strategy, and verification strictness; these should be addressed with configurable defaults and integration tests.

### R4: GitHub Copilot Integration (via SDK)

#### Description

BAU will integrate with GitHub Copilot using the official Copilot SDK for Go (github.com/github/copilot-sdk/go). The SDK provides programmatic access to the Copilot CLI server: start/stop the server, create and manage sessions, stream assistant responses, and expose tools for Copilot to call.

By adopting the SDK, BAU can:

- Start or connect to a Copilot server programmatically
- Create sessions that include attachments (the technical-plan and JSON)
- Stream responses and reasoning data
- Provide typed tools (via DefineTool) for Copilot to query repository state or run safe git operations
- Resume sessions if necessary

This section replaces previous shell-based invocation examples and the manual "exec.Command" approach with SDK-based usage patterns.

#### Integration Rationale

- The SDK is the supported programmatic integration mechanism maintained by GitHub.
- It abstracts transport (stdio/TCP), JSON-RPC plumbing, and provides helper primitives (sessions, tools).
- Using the SDK reduces fragile shell interactions and enables richer integration (tools, streaming, permissions).

#### Copilot SDK Quick-Start Example

The recommended pattern for BAU is to create a Copilot client, start it, create a session with the technical plan attached, and either run a non-interactive "apply plan" message or open an interactive loop that forwards user input to the session.

```go
package copilot

import (
    "context"
    "fmt"
    "log"
    "os"
    "time"

    copilot "github.com/github/copilot-sdk/go"
)

func runCopilotWithPlan(ctx context.Context, planPath, cwd string) error {
    // Create client with options. Use stdio transport by default.
    client := copilot.NewClient(&copilot.ClientOptions{
        CLIPath:  os.Getenv("COPILOT_CLI_PATH"), // Optional override
        Cwd:      cwd,
        UseStdio: true,
        LogLevel: "error",
    })

    // Start the client (starts CLI server if needed)
    if err := client.Start(); err != nil {
        return fmt.Errorf("failed to start Copilot client: %w", err)
    }
    defer func() {
        errs := client.Stop()
        for _, e := range errs {
            log.Printf("copilot stop error: %v", e)
        }
    }()

    // Create a session with streaming enabled (so we can show incremental output)
    session, err := client.CreateSession(&copilot.SessionConfig{
        Model:     "gpt-5",
        Streaming: true,
        // Optionally expose tools (see DefineTool examples below)
    })
    if err != nil {
        return fmt.Errorf("failed to create Copilot session: %w", err)
    }
    defer session.Destroy()

    // Subscribe to session events
    done := make(chan bool)
    session.On(func(event copilot.SessionEvent) {
        switch event.Type {
        case "assistant.message_delta":
            if event.Data.DeltaContent != nil {
                fmt.Print(*event.Data.DeltaContent) // Streaming chunk
            }
        case "assistant.message":
            if event.Data.Content != nil {
                fmt.Println("\n--- Final assistant message ---")
                fmt.Println(*event.Data.Content)
            }
        case "session.idle":
            close(done)
        case "session.error":
            log.Printf("session error: %v", event)
        }
    })

    // Send a message referencing the plan as an attachment
    _, err = session.Send(copilot.MessageOptions{
        Prompt: fmt.Sprintf("Implement the changes described in @%s. Apply changes in order.", planPath),
        Attachments: []copilot.Attachment{
            {Type: "file", Path: planPath, DisplayName: "technical-plan.md"},
        },
    })
    if err != nil {
        return fmt.Errorf("failed to send message: %w", err)
    }

    // Wait until session becomes idle (assistant finished)
    select {
    case <-done:
        return nil
    case <-time.After(10 * time.Minute):
        return fmt.Errorf("copilot session timed out")
    case <-ctx.Done():
        return ctx.Err()
    }
}
```

#### Checking Copilot Availability (SDK approach)

Instead of `exec.LookPath` + `--version`, BAU should create a `copilot.Client` with sensible `ClientOptions` and call `Start()` and `Ping()` to confirm connectivity.

```go
func initializeCopilotClient(cliPath, cwd string) (*copilot.Client, error) {
    client := copilot.NewClient(&copilot.ClientOptions{
        CLIPath:  cliPath,
        Cwd:      cwd,
        UseStdio: true,
        LogLevel: "error",
    })
    if err := client.Start(); err != nil {
        return nil, fmt.Errorf("failed to start copilot client: %w", err)
    }
    // Ping to verify responsiveness
    if _, err := client.Ping("health-check"); err != nil {
        client.Stop()
        return nil, fmt.Errorf("copilot client ping failed: %w", err)
    }
    return client, nil
}
```

If `CLIPath` is not provided, the SDK uses `COPILOT_CLI_PATH` env var or defaults to `copilot`. Starting the client will spawn the local copilot CLI process (if not connecting to an existing CLI server). This behavior centralizes detection and connection logic.

#### Execution Modes

BAU supports a single, non-interactive programmatic execution mode for applying chunks via the Copilot SDK. The MVP uses short-lived sessions per chunk (create → send → wait → destroy) and does not provide an integrated REPL. Interactive, manual sessions remain possible by running the Copilot CLI directly outside BAU, but BAU itself will not implement a built-in REPL for the MVP.

```go
type CopilotExecutionMode int

const (
    // ModeNonInteractive - Programmatically send plan/chunk and wait for assistant to finish
    ModeNonInteractive CopilotExecutionMode = iota
)
```

Rationale:
- Avoids complexity and potential security/consent issues from embedding an interactive REPL in the tool.
- Keeps BAU deterministic in its lifecycle management of Copilot sessions (short-lived, per-chunk).
- Users who want to interact manually can run `copilot` themselves and follow the generated plan files.

#### Tools: Exposing Safe Operations to Copilot

BAU can expose typed tools using `copilot.DefineTool` to allow Copilot to query repository state or request controlled actions, which increases safety and observability.

Example tool: read a file or check for target file existence.

```go
type CheckFileParams struct {
    Path string `json:"path" jsonschema:"target file path"`
}

checkFileTool := copilot.DefineTool("check_file_exists", "Check if a repository file exists",
    func(params CheckFileParams, inv copilot.ToolInvocation) (bool, error) {
        _, err := os.Stat(params.Path)
        return err == nil, nil
    })

session, _ := client.CreateSession(&copilot.SessionConfig{
    Model: "gpt-5",
    Tools: []copilot.Tool{checkFileTool},
})
```

Tools should be narrowly scoped, return structured results, and log invocations for auditing.

#### Execution Modes

```go
type CopilotExecutionMode int

const (
    ModeNonInteractive CopilotExecutionMode = iota // Programmatically send plan and wait
    ModeInteractive                                // Open REPL via SDK; user interacts via REPL
)
```

BAU will default to non-interactive programmatic execution but allow interactive REPL when `--interactive` is passed.

#### Error Handling: Target File Not Found (prompt + tool)

When creating the prompt, also provide the target-file list as an attachment and expose a `check_file_exists` tool so Copilot can query the repository before making changes. If the tool reports a missing target, Copilot should return an explicit error message and BAU will surface that to the user.

```go
initialPrompt := fmt.Sprintf(
    "Read and implement the technical plan in @%s. "+
    "Try primary path: %s. Alternatives: %v. If you cannot find the file, call the 'check_file_exists' tool and report an error if not found.",
    planPath,
    target.PrimaryPath,
    target.AlternativePaths,
)
```

---

### R5: Git Change Detection & Branch Management

[This section remains functionally the same; BAU will continue to use git via exec safely. See original implementations for details.]

(omitted here for brevity in this summary — implementation details kept in file body)

---

### R6: Pull Request Creation

[This section remains functionally the same, recommending `gh` CLI for PR creation or go-github for API fallback. The Copilot integration changes do not alter PR creation behavior.]

(omitted here for brevity in this summary — implementation details kept in file body)

---

## Architecture Design

### High-Level Flow

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                              BAU CLI                                         │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│  ┌──────────┐   ┌──────────┐   ┌──────────┐   ┌──────────┐   ┌──────────┐  │
│  │ 1. Parse │──▶│ 2.Extract│──▶│ 3.Generate│──▶│4.Execute │──▶│ 5.Create │  │
│  │   Args   │   │   Docs   │   │   Plan   │   │ (Copilot)│   │    PR    │  │
│  └──────────┘   └──────────┘   └──────────┘   └──────────┘   └──────────┘  │
│       │              │              │              │              │          │
│       ▼              ▼              ▼              ▼              ▼          │
│  ┌──────────┐   ┌──────────┐   ┌──────────┐   ┌──────────┐   ┌──────────┐  │
│  │ Validate │   │Google API│   │ Template │   │ Copilot  │   │  gh CLI  │  │
│  │  Inputs  │   │  Client  │   │  Engine  │   │  SDK     │   │  Client  │  │
│  └──────────┘   └──────────┘   └──────────┘   └──────────┘   └──────────┘  │
│                                                                              │
└─────────────────────────────────────────────────────────────────────────────┘
```

### Proposed Directory Structure

```
projects/bau/
├── cmd/
│   └── bau/
│       └── main.go                 # Entry point
├── internal/
│   ├── cli/
│   │   ├── root.go                 # Root command
│   │   ├── extract.go              # Extract subcommand
│   │   ├── plan.go                 # Plan subcommand
│   │   ├── implement.go            # Implement subcommand (uses Copilot SDK)
│   │   └── pr.go                   # PR subcommand
│   ├── config/
│   │   ├── config.go               # Configuration loading
│   │   ├── credentials.go          # Credential resolution
│   │   └── validate.go             # Input validation
│   ├── gdocs/
│   │   ├── client.go               # Docs/Drive API client
│   │   ├── extractor.go            # Document extraction
│   │   ├── suggestions.go          # Suggestion parsing
│   │   ├── comments.go             # Comment parsing
│   │   └── structure.go            # Document structure analysis
│   ├── planner/
│   │   ├── planner.go              # Plan generation
│   │   ├── template.go             # Template loading/execution
│   │   ├── funcs.go                # Template helper functions
│   │   └── target.go               # Target file resolution
│   ├── copilot/
│   │   ├── client.go               # SDK client initialization & helpers
│   │   ├── executor.go             # Execution wrapper using SDK
│   │   └── tools.go                # DefineTool helpers
│   ├── git/
│   │   ├── repo.go                 # Repository operations
│   │   ├── status.go               # Status parsing
│   │   └── branch.go               # Branch operations
│   └── github/
│       ├── cli.go                  # gh CLI wrapper
│       └── pr.go                   # PR creation
├── pkg/
│   ├── models/
│   │   └── models.go               # Shared data types
│   └── errors/
│       └── errors.go               # Custom errors
├── templates/                       # Embedded templates
│   ├── default.md                  # Default technical plan template
│   ├── copilot-prompt.md           # Copilot-optimized prompt
│   ├── pr-body.md                  # PR description template
│   └── copilot-instructions.md     # Instructions for Copilot
├── testdata/
│   ├── sample-extraction.json      # Test fixtures
│   └── sample-plan.md
├── go.mod
├── go.sum
├── README.md
├── FULL_SPEC.md                    # This document
└── Makefile
```

---

## Security Considerations

(unchanged except for note that Copilot SDK reduces shell usage; see relevant code examples in SDK sections)

[Detailed security mitigations for credentials, API scopes, git safety, process security, and data security remain as described earlier. Use tools instead of spawning shells where possible; when shelling out is necessary prefer exec.Command arg arrays and sanitize inputs.]

---

## Learning Resources

(Updated to include Copilot SDK docs)

- Copilot SDK for Go - https://pkg.go.dev/github.com/github/copilot-sdk/go
- Copilot CLI docs - https://docs.github.com/en/copilot/github-copilot-in-the-cli

[Other resources unchanged]

---

## Detailed Implementation Plan

## POC Cleanup Integration

The phased implementation plan includes all cleanup tasks from the README's "Future Improvements" section:

### Tasks from README to Integrate

| README Section                 | Task                                  | Phase   |
| ------------------------------ | ------------------------------------- | ------- |
| Remove Hardcoded Configuration | Extract `googleDocURL` to CLI flag    | Phase 1 |
| Remove Hardcoded Configuration | Extract `delegationEmail` to CLI flag | Phase 1 |
| Remove Hardcoded Configuration | Extract `useDelegation` to CLI flag   | Phase 1 |
| Remove Hardcoded Configuration | Extract credentials path to flag/env  | Phase 1 |
| Remove Hardcoded Configuration | Extract anchor length to config       | Phase 1 |
| Proposed File Structure        | Split `main.go` into packages         | Phase 1 |
| Proposed File Structure        | Create `config/` package              | Phase 1 |
| Proposed File Structure        | Create `auth/` package                | Phase 1 |
| Proposed File Structure        | Create `docs/` package                | Phase 1 |
| Proposed File Structure        | Create `drive/` package               | Phase 1 |
| Proposed File Structure        | Create `models/` package              | Phase 1 |
| Proposed File Structure        | Create `output/` package              | Phase 2 |
| Enhancement Ideas              | CLI interface (Cobra)                 | Phase 1 |
| Enhancement Ideas              | Environment variables support         | Phase 1 |
| Enhancement Ideas              | Output to file (`--output`)           | Phase 2 |
| Testing Plan                   | Unit tests for extractors             | Phase 1 |
| Testing Plan                   | Integration tests                     | Phase 3 |
| Testing Plan                   | CI/CD Integration                     | Phase 3 |

### Refactoring Map

Current `main.go` functions and their new locations:

| Current Function                  | New Location                    | Notes                        |
| --------------------------------- | ------------------------------- | ---------------------------- |
| `buildDocsService()`              | `internal/gdocs/client.go`      | Rename to `NewDocsClient()`  |
| `buildDriveService()`             | `internal/gdocs/client.go`      | Rename to `NewDriveClient()` |
| `extractDocumentID()`             | `internal/gdocs/extractor.go`   | Keep name                    |
| `fetchDocumentContent()`          | `internal/gdocs/extractor.go`   | Keep name                    |
| `extractSuggestions()`            | `internal/gdocs/suggestions.go` | Keep name                    |
| `buildDocumentStructure()`        | `internal/gdocs/structure.go`   | Keep name                    |
| `findParentHeading()`             | `internal/gdocs/structure.go`   | Keep name                    |
| `findTableLocation()`             | `internal/gdocs/structure.go`   | Keep name                    |
| `getTextAround()`                 | `internal/gdocs/structure.go`   | Keep name                    |
| `buildActionableSuggestions()`    | `internal/gdocs/suggestions.go` | Keep name                    |
| `extractCellText()`               | `internal/gdocs/structure.go`   | Keep name                    |
| `extractMetadataTable()`          | `internal/gdocs/structure.go`   | Keep name                    |
| `fetchComments()`                 | `internal/gdocs/comments.go`    | Keep name                    |
| `main()`                          | `cmd/bau/main.go`               | Minimal, calls CLI           |
| Types (Suggestion, Comment, etc.) | `pkg/models/models.go`          | Group all types              |

---

## Template Files Specification

### Template 1: `templates/default.md` (Technical Plan)

**Purpose:** Main technical implementation plan for human review and Copilot consumption.

**Required Sections:**

1. Overview header with document metadata
2. Target file information with confidence level
3. Implementation tasks (numbered, actionable)
4. Change details with anchor text
5. Verification data (embedded JSON)
6. Comments section
7. Execution notes

**Template Variables:**

- `{{ .DocumentTitle }}` - Title of the Google Doc
- `{{ .DocumentID }}` - Google Doc ID
- `{{ .DocumentURL }}` - Full URL to the doc
- `{{ .ExtractionTime }}` - ISO timestamp
- `{{ .TargetFile.PrimaryPath }}` - Most likely target file
- `{{ .TargetFile.AlternativePaths }}` - Backup paths to try
- `{{ .TargetFile.Confidence }}` - "high", "medium", "low"
- `{{ .Metadata.PageTitle }}` - From metadata table
- `{{ .Metadata.PageDescription }}` - From metadata table
- `{{ .SuggestionCount }}` - Total suggestions
- `{{ .CommentCount }}` - Total comments
- `{{ .Suggestions }}` - Array of ActionableSuggestion
- `{{ .Insertions }}` - Filtered suggestions
- `{{ .Deletions }}` - Filtered suggestions
- `{{ .Replacements }}` - Filtered suggestions
- `{{ .Comments }}` - Array of Comment
- `{{ .TargetRepo }}` - Path to repository
- `{{ .TargetBranch }}` - Branch name

**Template Functions:**

- `{{ toJSON .Anchor }}` - Serialize to JSON
- `{{ truncate .Text 50 }}` - Truncate with ellipsis
- `{{ add .Index 1 }}` - Arithmetic
- `{{ title .Type }}` - Title case

---

### Template 2: `templates/copilot-prompt.md`

**Purpose:** Optimized prompt specifically for Copilot SDK sessions.

**Characteristics:**

- Concise, action-oriented language
- Clear error handling instructions
- File references should be attached as `Attachment` via the SDK (BAU attaches `technical-plan.md` and `extraction.json`)
- Step-by-step execution order

**Example Content (for SDK use):**

```markdown
# Task: Implement Document Feedback

## Target File

Primary: `{{ .TargetFile.PrimaryPath }}`
{{- if .TargetFile.AlternativePaths }}
Alternatives: {{ range .TargetFile.AlternativePaths }}`{{ . }}` {{ end }}
{{- end }}

## Instructions

1. Use the attached file `technical-plan.md` to implement changes.
2. Use the provided `check_file_exists` tool to verify file locations before modifying.
3. Apply {{ .SuggestionCount }} changes in order below using anchor text to locate positions.
4. For each change, verify the `verification.text_after_change` matches the resulting file.

## Changes

{{ range $i, $s := .Suggestions }}

### {{ add $i 1 }}. {{ $s.Change.Type | title }}

- Find: `{{ truncate $s.Anchor.PrecedingText 40 }}`
- {{ if eq $s.Change.Type "insert" }}Insert after: `{{ $s.Change.NewText }}`
  {{- else if eq $s.Change.Type "delete" }}Delete: `{{ $s.Change.OriginalText }}`
  {{- else }}Replace `{{ $s.Change.OriginalText }}` with `{{ $s.Change.NewText }}`
  {{- end }}
  {{ end }}

## Error Handling

- If `check_file_exists` reports the file missing: STOP and report the error via session response
- If anchor not found: Report anchor and call tool `list_similar_locations` (if exposed)
- If ambiguous: Ask user via REPL (interactive) or fail with explicit error (non-interactive)
```

---

### Template 3: `templates/pr-body.md`

**Purpose:** Pull request description template.

**Required Sections:**

1. Summary with link to source doc
2. Change statistics
3. Detailed change list (collapsible)
4. Comments addressed (if any)
5. Verification checklist
6. Footer with BAU attribution

**Example Content:**

```markdown
## Summary

This PR implements feedback from the Google Doc: [{{ .DocumentTitle }}]({{ .DocumentURL }})

**Generated by BAU** on {{ .ExtractionTime.Format "2006-01-02 15:04" }}

## Changes Applied

| Type               | Count                   |
| ------------------ | ----------------------- |
| Insertions         | {{ len .Insertions }}   |
| Deletions          | {{ len .Deletions }}    |
| Replacements       | {{ len .Replacements }} |
| Comments Addressed | {{ .CommentCount }}     |

<details>
<summary>📝 Detailed Changes</summary>

{{ range $i, $s := .Suggestions }}

### Change {{ add $i 1 }}: {{ $s.Change.Type | title }}

**Location:** {{ if $s.Location.ParentHeading }}{{ $s.Location.ParentHeading }}{{ else }}Document root{{ end }}

{{ if eq $s.Change.Type "insert" -}}
**Inserted:** `{{ truncate $s.Change.NewText 100 }}`
{{- else if eq $s.Change.Type "delete" -}}
**Deleted:** `{{ truncate $s.Change.OriginalText 100 }}`
{{- else -}}
**Before:** `{{ truncate $s.Change.OriginalText 50 }}`
**After:** `{{ truncate $s.Change.NewText 50 }}`
{{- end }}

{{ end }}

</details>

{{ if .Comments }}

## Comments Addressed

{{ range .Comments }}

- **{{ .Author }}**: {{ truncate .Content 100 }}
  {{ end }}
  {{ end }}

## Verification Checklist

- [ ] Changes match the suggestions in the source document
- [ ] No unintended modifications
- [ ] Page renders correctly (if applicable)
- [ ] Tests pass (if applicable)

---

_Generated by [BAU](https://github.com/canonical/bau) - Build Automation Utility_
```

---

### Template 4: `templates/copilot-instructions.md`

**Purpose:** Project-level Copilot instructions to include in `.github/copilot-instructions.md`

**Note:** This is a template for users to customize, not used directly by BAU.

**Example Content:**

```markdown
# BAU Implementation Instructions

When implementing changes from a BAU technical plan:

## Finding Locations

1. **Use anchor text exactly** - The `preceding_text` and `following_text` fields contain exact strings
2. **Search globally first** - Don't assume the location, search the entire file
3. **Report if not found** - Never guess, always report missing anchors

## Applying Changes

| Change Type | Action                                            |
| ----------- | ------------------------------------------------- |
| `insert`    | Add `new_text` immediately after `preceding_text` |
| `delete`    | Remove `original_text` between anchors            |
| `replace`   | Replace `original_text` with `new_text`           |

## Verification

After each change, verify:

- The surrounding text matches `verification.text_after_change`
- No duplicate insertions occurred
- Formatting is preserved

## Error Handling

If you encounter issues:

1. Stop and report the specific problem
2. Show the text you found vs. expected
3. Ask for guidance before proceeding

## Commit Messages

Use format: `Apply feedback: [brief description]`
```

---

### Template 5: `templates/extraction-summary.json` (Schema)

[Unchanged]

---

## Copilot SDK Configuration

### Can We Configure Copilot Programmatically?

Yes. The Copilot SDK exposes `ClientOptions` and `SessionConfig` parameters that BAU can use to configure behavior at runtime rather than editing user config files directly. This reduces risk of modifying user config on disk unexpectedly.

Key `ClientOptions` fields:

- `CLIPath` - path to the Copilot CLI executable
- `UseStdio` - whether to use stdio transport
- `CLIUrl` - connect to an existing server instead of spawning
- `Cwd` - working directory for spawned CLI process
- `Env` - environment variables for spawned process
- `AutoStart`/`AutoRestart` pointers - control auto-start behavior

Example: create a client that spawns the CLI in a specific directory and sets a port for TCP mode (if needed).

```go
client := copilot.NewClient(&copilot.ClientOptions{
    CLIPath:  "/usr/local/bin/copilot",
    Cwd:      "/path/to/repo",
    UseStdio: true,
    LogLevel: "info",
})
if err := client.Start(); err != nil {
    return err
}
```

### MCP Server Configuration via SDK

Rather than editing `~/.copilot/mcp-config.json` manually, BAU can pass MCP server configuration when creating/resuming a session using `SessionConfig.MCPServers`. This is a safer, session-scoped approach:

```go
mcp := map[string]copilot.MCPServerConfig{
    "filesystem": copilot.MCPLocalServerConfig{
        Type:    "stdio",
        Command: "/usr/local/bin/my-filesystem-mcp",
        Args:    []string{"--serve"},
        Cwd:     "/usr/local/bin",
        Timeout: 30,
    },
}

session, err := client.CreateSession(&copilot.SessionConfig{
    Model:     "gpt-5",
    MCPServers: mcp,
})
```

This avoids persistently mutating user config and limits any changes to the lifetime of the session.

### Trust Directory Handling

Copilot still requires the user to trust a repository directory. BAU should not automatically change trust settings on behalf of the user. Instead:

- Use SDK Start() / Ping() to surface trust-related failures
- If the server refuses to operate due to untrusted directory, instruct the user to run `copilot` interactively once and approve trust, or use BAU's `bau setup-copilot` command to walk the user through the trust process.

Example check:

```go
if _, err := client.Ping("health"); err != nil {
    fmt.Println("Copilot CLI reported an issue. If this is a new repository, run 'copilot' interactively to accept the trust prompt, then re-run BAU.")
}
```

### Custom Instructions

Copilot SDK sessions respect repository-level `.github/copilot-instructions.md`. BAU can recommend creating such a file and can generate one via `bau init-copilot-instructions`. BAU should only suggest and not force-write repository files unless the user consents.

```go
func checkCopilotInstructions(repoPath string) {
    instructionsPath := filepath.Join(repoPath, ".github", "copilot-instructions.md")
    if _, err := os.Stat(instructionsPath); os.IsNotExist(err) {
        slog.Info("Tip: Create .github/copilot-instructions.md for better Copilot results",
            "path", instructionsPath)
        fmt.Println("\nRun: bau init-copilot-instructions to create a template.")
    }
}
```

### Other Considerations

1. Session Persistence:
   - Sessions created via the SDK can be resumed with `ResumeSession`/`ResumeSessionWithOptions`.
2. Streaming:
   - The SDK supports streaming via `Streaming: true` in `SessionConfig` and delivers `assistant.message_delta` events.
3. Permissions/Tools:
   - Use SDK `PermissionHandler` and `DefineTool` to implement safe, auditable tool integrations.

---

## Detailed Implementation Plan

[Unchanged; however, the `internal/copilot/` package will implement the SDK-based client init, session creation, tools, and REPL.]

---

## Progress Reporting

### Approaches

| Approach                         | Description                                          | Pros                                                        | Cons                                                     |
| -------------------------------- | ---------------------------------------------------- | ----------------------------------------------------------- | -------------------------------------------------------- |
| **A. Simple println**            | Print status messages                                | - Simple<br>- No deps                                       | - No structure<br>- Hard to parse                        |
| **B. Structured logging (slog)** | Use Go's slog package                                | - Built-in<br>- Configurable levels<br>- JSON output option | - More verbose for simple cases                          |
| **C. Progress bars**             | Use a library like progressbar                       | - Visual feedback<br>- Good UX                              | - External dependency<br>- Overkill for quick operations |
| **D. Hybrid: slog + spinners**   | Structured logging with visual spinners for long ops | - Best UX<br>- Informative<br>- Professional                | - More complex                                           |

### Recommendation: **Approach B (Structured logging with slog)**

For Phase 1, use simple structured logging. Can enhance with spinners later.

### Implementation

```go
func setupLogging(verbose bool) {
    level := slog.LevelInfo
    if verbose {
        level = slog.LevelDebug
    }

    handler := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
        Level: level,
    })
    slog.SetDefault(slog.New(handler))
}

// Usage throughout the code:
slog.Info("Extracting document", "doc_id", docID)
slog.Debug("Found suggestion", "id", suggestion.ID, "type", suggestion.Type)
slog.Warn("No changes detected after Copilot execution")
slog.Error("Failed to create PR", "error", err)
```

### Progress Output Example

```
$ bau --doc-url "https://docs.google.com/..."

BAU - Build Automation Utility v1.0.0
=====================================

[1/5] Validating environment...
  ✓ Git repository detected
  ✓ Repository is clean
  ✓ GitHub Copilot CLI found (v1.0.0)
  ✓ GitHub CLI authenticated

[2/5] Extracting from Google Doc...
  ✓ Document fetched: "Ubuntu on AWS - index.html"
  ✓ Found 5 suggestions (3 insertions, 2 deletions)
  ✓ Found 2 comments
  ✓ Target file: templates/aws/ubuntu.html (confidence: high)

[3/5] Generating technical plan...
  ✓ Saved: ./output/extraction.json
  ✓ Saved: ./output/technical-plan.md
  ✓ Saved: ./output/copilot-prompt.md

[4/5] Starting Copilot CLI...
============================================================
  Copilot will start with the technical plan.
  You can interact normally. Exit when done.
============================================================

[Copilot CLI session - user interacts here]

============================================================
  Copilot exited. Checking for changes...
============================================================

[5/5] Creating pull request...
  ✓ Branch created: bau/2025-01-15-103045-ubuntu-aws
  ✓ Changes committed: 3 files modified
  ✓ Pushed to origin
  ✓ PR created: https://github.com/org/repo/pull/123

=====================================
SUCCESS: Feedback applied and PR created!
  PR: https://github.com/org/repo/pull/123
  Branch: bau/2025-01-15-103045-ubuntu-aws
  Changes: 5 suggestions applied
=====================================
```

---

## Risk Assessment

This section evaluates the main technical, operational, and security risks for BAU under the current design (Copilot SDK integration, chunked short-session execution, CHUNK_SIZE-driven batching, and minimal verification for MVP).

### Risk Matrix

| ID  | Risk                                                  | Category     | Impact  | Likelihood | Severity | Mitigation / Response |
|-----|-------------------------------------------------------|--------------|---------|------------|----------|------------------------|
| R1  | Copilot SDK/API changes or protocol incompatibility   | External     | High    | Medium     | High     | Pin SDK version in go.mod; add defensive error handling and graceful fallback messages; update spec when SDK changes. |
| R2  | Chunk fails to apply (assistant produces no edits)    | Operational  | Medium  | Medium     | Medium   | Retry chunk once; surface diagnostics and assistant output; stop run and require user intervention. Commit-per-chunk enables easy rollback. |
| R3  | Incorrect edits applied by Copilot                    | Quality      | High    | Medium     | High     | Default: MVP disables auto-verification. Use commit-per-chunk and require PR review. Add `--verification-mode` later for stricter checks. |
| R4  | Credentials or tokens leaked in logs/output           | Security     | High    | Low        | High     | Sanitize logs, avoid printing sensitive content, mask paths and tokens. Follow existing credential handling guidelines. |
| R5  | Git conflicts or overlapping edits across chunks      | Operational  | Medium  | Low        | Medium   | Grouping preserves ordering; do not process chunks in parallel; detect overlapping ranges and fail early. |
| R6  | CLI or environment trust issues for Copilot CLI       | Operational  | High    | Medium     | High     | Use SDK Start()/Ping() to detect trust issues; instruct user to run `copilot` interactively to accept trust; do not programmatically change trust. |
| R7  | Excessive token usage causing truncated prompts       | Performance  | Medium  | Medium     | Medium   | Use CHUNK_SIZE constant to cap items per chunk; allow operators to decrease chunk size via flag. Consider token-budgeting later. |
| R8  | Network/transient SDK errors during session startup   | Infrastructure| Medium | Medium     | Medium   | Retry with small backoff for transient errors; collect session logs for diagnosis. |
| R9  | Over-reliance on assistant decisions without verification | Process  | High   | Medium     | High     | For MVP, require human PR review; plan to add `--verification-mode` and `preview_patch` tools later. |

### Risk Response Strategies

- R1 (SDK changes): Track the SDK release notes; vendor a small compatibility shim that isolates code from protocol churn; add version check during startup.
- R2 (Chunk failure): Implement retry-once policy; log assistant output and diff; stop after second failure and present user actionable diagnostics (assistant output, attachments, git status).
- R3 (Incorrect edits): Default to commit-per-chunk + PR review. Later add automated verification (exact or fuzzy) before committing.
- R4 (Credentials): Reuse existing credential handling code; redact sensitive fields in logs and in execution artifacts.
- R5 (Conflicts): Chunking preserves document order; if overlapping ranges are detected, mark as manual review and abort the automated application.
- R6 (Trust): Surface clear instructions; do not automate trust-setting.
- R7 (Token limits): CHUNK_SIZE default limits the number of grouped suggestions. Provide `--chunk-size` flag to tune behavior in edge cases.
- R8 (Transient errors): Add exponential backoff (configurable attempts) for CreateSession/Send errors; persist session logs for debugging.
- R9 (No verification): Make verification optional in future; MVP uses manual verification via PR review.

### Monitoring & Observability

- Persist per-chunk execution logs (assistant events, session IDs, attachments references) to `execution-log.txt`.
- Capture git diffs per-chunk into a `./bau-output/diffs/` folder for auditability.
- Emit structured slog entries for each chunk lifecycle event: start, sent, assistant-completed, git-changes-detected, commit-made, failed.


---

## Phased Implementation Plan

### Phase 1: CLI Foundation & POC Cleanup ✅ COMPLETE

**Objective:** Transform hardcoded POC into configurable CLI tool with proper project structure

**Status:** All tasks complete (January 2025)

**Tasks:**

| Task | Status | Description                                                                              | Notes                   |
| ---- | ------ | ---------------------------------------------------------------------------------------- | ----------------------- |
| 1.1  | ✅     | Add standard `flag` package support                                                      | Implemented             |
| 1.2  | ✅     | Implement `--doc-id` flag                                                                | Working                 |
| 1.3  | ✅     | Implement `--credentials` flag (file path only)                                          | With validation         |
| 1.4  | ✅     | Implement `--dry-run` flag (skip Copilot/PR)                                             | Working                 |
| 1.5  | ✅     | Create `internal/config/` package                                                        | Created                 |
| 1.6  | ✅     | Create `internal/gdocs/` package (refactor)                                              | Refactored              |
| 1.7  | ✅     | Create `internal/models/` package                                                        | Merged into types.go    |
| 1.8  | ✅     | Add slog-based logging with JSON prettification options                                  | Outputs to log.json     |
| 1.9  | ✅     | Unit tests for config and validation                                                     | 6/6 passing             |
| 1.10 | ✅     | Refactor `buildDocumentStructure`: extract heading extraction into a helper function     | Implemented             |
| 1.11 | ✅     | Update `extractMetadataTable`: add check for "metadata" (case-insensitive) in first cell | Working                 |
| 1.12 | ✅     | Enhance table extraction: extract table title/header (text right above the table)        | Implemented             |
| 1.13 | 🔄     | Add git repository validation                                                            | Deferred to Phase 3     |

**Deliverables:**

- [x] flag-based CLI with MVP flags (`--doc-id`, `--credentials`, `--dry-run`)
- [x] Credential loading from file path
- [x] Refactored code in proper package structure (`internal/config`, `internal/gdocs`)
- [x] Unit tests for config package (6/6 passing)
- [x] Heading extraction helper
- [x] Prettified slog output to `log.json`
- [x] Enhanced table and metadata extraction logic
- [x] Git repository detection deferred to Phase 3

**Exit Criteria:** ✅ Met

```bash
bau --doc-id "1b9F1Av8tRNG8xkPHgjvtBKrQogXRDaRb0Lw7pEZxr9I" --credentials ./creds.json
bau --help
bau --doc-id "1b9F1Av8..." --credentials ./creds.json --dry-run
```

### Phase 2: Prompt Generation & Templates ✅ COMPLETE

**Objective:** Generate structured prompts for GitHub Copilot with embedded templates

**Status:** Fully implemented with simplifications (January 2025)

**Key Design Decisions:**
1. ✅ **Package naming**: `internal/prompt` (not `template` - avoids stdlib conflicts)
2. ✅ **Chunking strategy**: Location-based (simple array slicing), not suggestion-count based
3. ✅ **Template approach**: Simple string replacement, no `html/template` package needed
4. ✅ **Output format**: Raw JSON embedded in markdown (transparent, debuggable)
5. ✅ **Repo/branch handling**: User responsible (CWD = repo, active branch used)
6. ✅ **Default output dir**: `bauer-output`

**Tasks:**

| Task | Status | Implementation                   | Notes                          |
| ---- | ------ | -------------------------------- | ------------------------------ |
| 2.1  | ✅     | `templates/instructions.md`      | Main prompt instructions       |
| 2.2  | ✅     | Embedded as JSON in instructions | Simplified approach            |
| 2.3  | 🔄     | Deferred to Phase 3              | PR creation not in scope yet   |
| 2.4  | ✅     | `go:embed` for all templates     | Templates bundled in binary    |
| 2.5  | ✅     | `internal/prompt/` package       | Renamed from "planner"         |
| 2.6  | ✅     | `PromptData` struct              | Simplified data model          |
| 2.7  | ✅     | `replaceVar()` helper            | Simple string replacement      |
| 2.8  | ✅     | Documented in template           | Path resolution algorithm      |
| 2.9  | 🔄     | Deferred                         | Custom templates not needed    |
| 2.10 | ✅     | `--output-dir` flag              | Default: `bauer-output`        |
| 2.11 | ✅     | `doc-suggestions.json`           | Generated for reference        |
| 2.12 | ✅     | `chunk-N-of-M.md` files          | Prompt files per chunk         |
| 2.13 | ✅     | 5 tests passing                  | Chunking, rendering, replacing |

**New Flags Added:**
- `--chunk-size` (default: 10) - Maximum locations per chunk
- `--output-dir` (default: "bauer-output") - Output directory for prompts

**Deliverables:**

- [x] Embedded template system with `go:embed`
- [x] Simple string replacement (no template engine needed)
- [x] Path resolution documented in template
- [x] Output file generation with numbered chunks (`chunk-1-of-3.md`)
- [x] Vanilla Framework patterns reference embedded
- [x] JSON schema documentation in template
- [x] Location-based chunking (simple & predictable)

**Removed from Scope:**
- ❌ `--target-repo` flag (user runs from repo directory)
- ❌ `--target-branch` flag (user checks out correct branch)
- ❌ Commit guidelines in template (out of scope)
- ❌ Document ID/URL in templates (unnecessary)
- ❌ Complex `html/template` rendering (overkill)
- ❌ Suggestion-count based chunking (too complex)

**Exit Criteria:** ✅ Met

```bash
# Generate prompts (default chunk size 10)
bau --doc-id "abc123" --credentials ./creds.json --dry-run
# Output: bauer-output/chunk-1-of-3.md (10 locations)
#         bauer-output/chunk-2-of-3.md (10 locations)
#         bauer-output/chunk-3-of-3.md (3 locations)

# Custom chunk size and output directory
bau --doc-id "abc123" --credentials ./creds.json --chunk-size 15 --output-dir my-prompts
```

**Implementation Highlights:**

```go
// Simple location-based chunking (no complex logic)
func ChunkLocations(groups []gdocs.LocationGroupedSuggestions, chunkSize int) 
    [][]gdocs.LocationGroupedSuggestions {
    for i := 0; i < len(groups); i += chunkSize {
        end := i + chunkSize
        if end > len(groups) { end = len(groups) }
        chunks = append(chunks, groups[i:end])
    }
}

// Simple string replacement (no html/template)
instructions = replaceVar(instructions, "DocumentTitle", data.DocumentTitle)

// Embed raw JSON for transparency
buf.WriteString("```json\n" + data.SuggestionsJSON + "\n```\n")
```

**Testing:** 39/39 tests passing (6 config + 28 gdocs + 5 prompt)

### Phase 3: Copilot & GitHub Integration ✅ COMPLETE

**Core Tasks (Complete):**

| Task | Status | Description                                                                                                   |
| ---- | ------ | ------------------------------------------------------------------------------------------------------------- |
| 3.1  | ✅     | Create `internal/copilot/` package that uses github/copilot-sdk/go (implemented as `internal/copilotcli/`)    |
| 3.2  | ✅     | Implement Copilot client initialization with `ClientOptions`                                                  |
| 3.3  | ✅     | Implement non-interactive execution using `Session.Send` and event streaming                                  |
| 3.4  | ⏸️     | Implement interactive REPL using SDK events and `session.Send` (skipped - not needed for BAU's use case)      |
| 3.5  | 📋     | Define safe tools using `copilot.DefineTool` to expose controlled repo operations (optional - deferred)       |
| 3.6  | 📋     | Provide `bau setup-copilot` helper to guide users through trust/permission steps (optional - deferred)        |

**Implementation Summary:**

- ✅ Created `internal/copilotcli/` package wrapping the GitHub Copilot SDK
- ✅ Implemented `NewClient()` with proper `ClientOptions` (stdio transport, configurable log level)
- ✅ Implemented `Start()` with health check ping verification
- ✅ Implemented `Stop()` with graceful shutdown and error collection
- ✅ Implemented `ExecuteChunk()` with:
  - Session creation with streaming enabled
  - Event handler for `assistant.message_delta`, `assistant.message`, `session.idle`, `session.error`
  - Absolute path resolution for chunk file attachments
  - 15-minute timeout protection with context cancellation support
- ✅ Integrated chunk execution into main flow with progress reporting
- ✅ Aligned progress output with spec's reporting format (numbered steps, checkmarks)
- ✅ Fixed configuration flag from `--models` to `--model` (singular)
- ✅ Updated tests to cover model configuration
- ✅ Implemented early return for dry-run mode

**Notes:**

- Tasks 3.4, 3.5, and 3.6 are optional enhancements that can be added later based on user feedback
- Core non-interactive execution flow is fully functional and tested
- Currently relies on Copilot's built-in tools; custom tools can be added via `copilot.DefineTool` if needed

---

## Testing Strategy

### Unit Tests

- Add tests for `internal/copilot` that mock the SDK interfaces where feasible.
- Use integration tests to run the actual Copilot CLI server locally (if available) in CI optional jobs.

### Integration Tests

- `TestCopilotSession` - start client, create session, send simple prompt, validate response shape (may be skipped in CI if CLI not available).
- `TestDefineTool` - verify that `DefineTool` handlers execute and return expected result when the SDK simulates a tool call.

---

## Appendices

### Appendix A: Environment Variables Reference

(Include `COPILOT_CLI_PATH` - SDK uses this env var; BAU also honors it.)

### Appendix F: CI/CD Considerations

Note: Copilot sessions require the Copilot CLI binary to be available. Interactive sessions are not suitable for CI. BAU should avoid starting interactive REPLs in CI; non-interactive `Send` calls may be usable if the runner has the CLI and appropriate tokens configured, but exercise caution. Use `--skip-copilot` in CI pipelines and run Copilot execution locally.

---

## Document History

| Version | Date       | Author   | Changes                                                                                                                                                   |
| ------- | ---------- | -------- | --------------------------------------------------------------------------------------------------------------------------------------------------------- |
| 1.0     | 2025-01-XX | BAU Team | Initial specification                                                                                                                                     |
| 1.1     | 2025-01-XX | BAU Team | Added target resolution, interactive mode details                                                                                                         |
| 1.2     | 2026-01-21 | BAU Team | Integrated GitHub Copilot SDK usage, replaced shell-based examples with SDK-based patterns, removed alternatives to using the SDK for Copilot integration |

---

_End of Technical Specification_
