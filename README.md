# Bauer

Bauer is a Go application that extracts document content, suggestions (proposed edits), and comments from Google Docs using the Google Docs API and Google Drive API.

> This README documents Bauer's **current** behavior (the `bauer` CLI and `bauer-api` server). The documents under [`docs/`](./docs/) describe **target designs** for future work — see [Documentation](#documentation).

## Local development

### Prerequisites

1. Install [`go`](https://golang.org/dl/)
2. Install [`task`](https://taskfile.dev/docs/installation)

### Steps
1. Build the project

```
task build
```
2. Run the project

```
./bauer --doc-id <doc-id> \
  --github-repo <github-repo>
```
`--doc-id` accepts either a bare document ID or a full Google Docs URL (e.g. `https://docs.google.com/document/d/<doc-id>`); the document ID and tab are extracted automatically.

Additionally, you can run it with `--parse-only` without creating an issue.


## Configuration

The service account credentials are read from environment variables (the same `APP_*` variables used by the API — see [Credentials via environment variables](#credentials-via-environment-variables)):

| Environment variable | Service account field |
| -------------------- | --------------------- |
| `APP_PROJECT_ID`     | `project_id`          |
| `APP_PRIVATE_KEY`    | `private_key`         |
| `APP_CLIENT_EMAIL`   | `client_email`        |
| `APP_CLIENT_ID`      | `client_id`           |

1. Get credentials from Google Cloud service or Bitwarden (internally) (see [Generating Google Cloud credentials](https://developers.google.com/workspace/guides/create-credentials)).
2. Export the `APP_*` variables in your shell or place them in a `.env` / `.env.local` file at the repository root (`task` loads these automatically).
3. Share the copy document with the service account.

## Usage

1.  Build Bauer locally using the Local development steps above (`task build`)
2. If running with GitHub issue creation (no `--parse-only`), ensure GitHub CLI auth is available
3. Get document ID from Google Document & share the document with the service account
4. Run Bauer

### GitHub CLI authentication

Run these once before using parse-and-issue mode:

```bash
gh auth login
gh auth status
```

If you prefer token auth in non-interactive environments:

```bash
export GH_TOKEN=<your_token>
gh auth status
```

```bash
task run-cli -- --doc-id <your-document-id> --parse-only
```

5. Parameters

| Flag               | Type   | Default              | Description                                                                     | Requires GitHub Auth |
| ------------------ | ------ | -------------------- | ------------------------------------------------------------------------------- | -------------------- |
| `--doc-id`         | string | (required)           | Google Doc ID or full Google Docs URL (e.g. `https://docs.google.com/document/d/<doc-id>/edit?tab=t.0`) | No               |
| `--github-repo`    | string | (required if not parse-only) | GitHub repository (owner/repo or HTTPS URL)                              | Yes*             |
| `--local-repo-path` | string | `/tmp/ubuntu.com`    | Local path for cloned repository                                               | No               |
| `--output-dir`     | string | `bauer-output`       | Output directory for Bauer results                                             | No               |
| `--branch-prefix`  | string | `bauer`              | Branch naming prefix                                                            | No               |
| `--parse-only`     | bool   | `false`              | Parse document and output machine-readable JSON only                           | No               |

Current execution modes:

1. Parse-only mode (`--parse-only`)
- Creates `bauer-output/bauer-parse-result.json`
- Does not push branches
- Does not create issues

2. Parse-and-issue mode (without `--parse-only`)
- Creates `bauer-output/bauer-parse-result.json`
- Creates a branch and pushes the parse file into that branch
- Opens a GitHub issue assigned to Copilot (with fallbacks) and includes branch/pinned/raw links to the prompt file

*GitHub auth is only required when using parse-and-issue mode.


## API usage

The API server exposes a small HTTP surface for triggering the PR-creation workflow and checking health.

Secrets are **never** accepted in request bodies. The Google service account credentials are read from environment variables (see below), and the GitHub token is resolved from the server environment (`GITHUB_TOKEN` / `GH_TOKEN` / `gh auth token`).

### Credentials via environment variables
Copy the `.env` file into `.env.local` and populate the Google API keys:

```bash
APP_PROJECT_ID=your-project
APP_PRIVATE_KEY="-----BEGIN PRIVATE KEY-----\n...\n-----END PRIVATE KEY-----\n"
APP_CLIENT_EMAIL=svc@your-project.iam.gserviceaccount.com
APP_CLIENT_ID=1234567890
```

### Run the API server

From the repository root:

```bash
task build
task run-server
```

`task run-server` reads the credentials from your environment (exported shell variables and/or the `.env` / `.env.local` files).
```bash
./bauer-api
```

The server listens on the port set by the `APP_PORT` environment variable (injected by the `go-framework` charm), defaulting to `:8080` when it is not set.

### Endpoints

#### GET /api/v1

Simple liveness check.

Example:

```bash
curl http://localhost:8080/api/v1
```

#### POST /api/v1

Trigger the PR-creation workflow for a Google Doc: extract suggestions, push the parse result to a branch on the target repository, and open a Copilot-assigned issue that drives PR creation.

Request body:

```json
{
  "doc_id": "<google-doc-id>",
  "github_repo": "owner/repo",
  "branch_prefix": "bauer",
  "page_refresh": false
}
```

Notes:

- `doc_id` and `github_repo` are required.
- `branch_prefix` defaults to `bauer` if omitted.

Responses:

- `201 Created` with `{"code":201,"status":"success","issue_url":"...","branch":"..."}` on success.
- `400 Bad Request` for invalid JSON or missing `doc_id` / `github_repo`.
- `500 Internal Server Error` if GitHub auth is not configured or the workflow fails.

Example:

```bash
curl -X POST http://localhost:8080/api/v1 \
        -H 'Content-Type: application/json' \
        -d '{"doc_id":"<google-doc-id>","github_repo":"owner/repo"}'
```


## Documentation

This README is the source of truth for how Bauer works today. The architecture documents describe **planned, not-yet-implemented** work:

- [`docs/ARCHITECTURE_v2.md`](./docs/ARCHITECTURE_v2.md) — target shared-core architecture (CLI + API + Jira webhook).
- [`docs/ARCHITECTURE_v2_1.md`](./docs/ARCHITECTURE_v2_1.md) — target design-aware source intake, starting with Figma.
- [`docs/specs/`](./docs/specs/) — the reconciliation plan (001) and Figma integration spec (002).

<!-- ## Future improvements

### Short term

- Automatically open PR with changes applied to the document using Google Docs API
- Improve prompt templates for better results (this requires a lot of trial and error)

for code improvements, you can also refer to our [todo](./todo.txt) list

### Long term

On the long term, BAUer should evolve into a full-fledged API service, with the following features:

- Automatic Jira ticket hooks to trigger workflows
- Unified service account with domain wide delegation
- Calling LLMs - with varying implementation complexity - via: 
        - calling LLM APIs directly
        - spinning up ephemeral Copilot CLI instances
        - self-hosted LLMs (can use open source models such as Llama, openAI OSS, deepseek, etc)
- Automatic PR creations and reviewer assignments


## Installation (WIP)

### [Snap](https://snapcraft.io/bauer)

```
sudo snap install bauer
```

### Homebrew

First time installation

```
brew install britneywwc/bauer/bauer
```

Upgrade to a newer version or later

```
brew update
brew upgrade bauer
``` -->