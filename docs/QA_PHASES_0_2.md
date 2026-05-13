# QA Guide: Phases 0, 1, and 2

This guide verifies the Bauer prerequisites for spec 002:

- source and agent abstraction are in place
- append-only artifacts work
- CLI config precedence is correct
- CLI modes behave correctly
- GitHub issue and PR flows are wired correctly

Use the spec in `docs/specs/001_v2_reconciliation.md` as the source of truth for expected behavior.

## Prerequisites

You need:

- `go`
- `gh`
- a Google service-account credentials file with access to the test doc
- a test repository with an `origin` remote
- GitHub auth via `BAUER_GITHUB_TOKEN`, `GITHUB_TOKEN`, `GH_TOKEN`, or `gh auth login`

Recommended shell setup:

```bash
export BAUER_DOC_ID="<test-doc-id>"
export BAUER_CREDENTIALS_PATH="/absolute/path/to/creds.json"
export BAUER_ARTIFACTS_DIR="$PWD/.tmp/bauer-artifacts"
export BAUER_OUTPUT_DIR="$PWD/.tmp/bauer-output"
```

Reset local test output before each scenario:

```bash
rm -rf .tmp/bauer-artifacts .tmp/bauer-output bauer-doc-suggestions.json
mkdir -p .tmp
```

## Automated validation

Run the full test suite first:

```bash
go test ./...
```

Expected result: all tests pass.

## Scenario 1: Standalone CLI

Run:

```bash
go run ./cmd/bauer --doc-id "$BAUER_DOC_ID" --credentials "$BAUER_CREDENTIALS_PATH"
```

Verify:

- command exits `0`
- chunk prompt files are written under `.tmp/bauer-output`
- a new run directory exists under `.tmp/bauer-artifacts`
- `.tmp/bauer-artifacts/runs.jsonl` contains one appended JSON line
- `bauer-doc-suggestions.json` exists for backward compatibility

## Scenario 2: Standalone Dry Run

Run:

```bash
go run ./cmd/bauer --doc-id "$BAUER_DOC_ID" --credentials "$BAUER_CREDENTIALS_PATH" --dry-run
```

Verify:

- command exits `0`
- output prints `Status: dry-run`
- chunk prompt files are written
- artifact run directory contains `extraction/` and `prompts/`
- no Copilot execution output is produced

## Scenario 3: Flags Override Env Vars

Run:

```bash
BAUER_DOC_ID="wrong-doc" go run ./cmd/bauer --doc-id "$BAUER_DOC_ID" --credentials "$BAUER_CREDENTIALS_PATH" --dry-run
```

Verify:

- the run succeeds using the explicit flag value, not `wrong-doc`
- there is no validation or fetch failure caused by the env var override path

## Scenario 4: Mutual Exclusion

Run:

```bash
go run ./cmd/bauer --doc-id "$BAUER_DOC_ID" --credentials "$BAUER_CREDENTIALS_PATH" --open-pr --open-issue
```

Verify:

- command exits `1`
- stderr explains that `--open-pr` and `--open-issue` are mutually exclusive
- no new artifact run directory is created

## Scenario 5: Open Issue

Inside a git repo with `origin` configured, run:

```bash
go run ./cmd/bauer --doc-id "$BAUER_DOC_ID" --credentials "$BAUER_CREDENTIALS_PATH" --open-issue
```

Verify:

- command exits `0`
- Copilot is not invoked
- a GitHub issue is created and the URL is printed
- the issue body includes doc ID, total suggestions, and chunk summary

## Scenario 6: Open PR Dry Run

Inside a clean test repo, make sure `origin` points to a repo you can push to, then run:

```bash
go run ./cmd/bauer --doc-id "$BAUER_DOC_ID" --credentials "$BAUER_CREDENTIALS_PATH" --open-pr --dry-run
```

Verify:

- command exits `0`
- Copilot applies changes locally
- output prints `dry-run: changes applied locally, PR creation skipped`
- no branch is pushed and no PR is created

This is the main regression check for the phase-1 dry-run fix.

## Scenario 7: Open PR

Inside a clean test repo, run:

```bash
go run ./cmd/bauer --doc-id "$BAUER_DOC_ID" --credentials "$BAUER_CREDENTIALS_PATH" --open-pr
```

Verify:

- command exits `0`
- a new branch is created from the repo default branch
- Bauer commits only code changes, not generated artifact files
- the branch is pushed
- a PR URL is printed

## Scenario 8: Artifact History Append-Only

Run any successful CLI scenario twice.

Verify:

- two distinct run directories exist under the artifacts dir
- `runs.jsonl` has two lines
- earlier run contents remain untouched

## Notes

- The README is explicitly marked as potentially out of date during the v2 and v2.1 transition. Prefer the spec and architecture docs when expected behavior and README differ.
- API verification is intentionally out of scope here; spec 001 phases 3 through 5 cover the API foundation and endpoints.
