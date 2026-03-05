# Bauer

A proof-of-concept Go application that extracts document content, suggestions (proposed edits), and comments from Google Docs using the Google Docs API and Google Drive API.

<!-- ## Installation

### Homebrew

First time installation

```
brew install britneywwc/bauer/bauer
```

Upgrade to a newer version or later

```
brew update
brew upgrade bauer
```

N.B. You need to install [Copilot CLI](https://docs.github.com/en/copilot/how-tos/set-up/install-copilot-cli) which is used by Bauer. -->

## Configuration

1. Install [Copilot CLI](https://docs.github.com/en/copilot/how-tos/set-up/install-copilot-cli)
2. Create a local credentials file: `cp credentials.json bau-credentials.json`
3. Get credentials from Google Cloud service or Bitwarden (internally)
4. Fill up `bau-credentials.json` with Google Cloud credentials (see [Generating Google Cloud credentials](https://developers.google.com/workspace/guides/create-credentials)).
5. Share copy document with service account

## Usage
1. Clone the project
2. Build the project with `task build`
3. Create a copy of the copy document and share it with the service account
4. Update `config.json` with `doc_id`, `github_repo` and `local_repo_path` (default is ubuntu.com repository)
5. Run Bauer

```bash
./bauer
```

### Parameters

Update `config.json` with the following parameters:

| Key                 | Type   | Default           | Description                                                                  |
| ------------------- | ------ | ----------------- | ---------------------------------------------------------------------------- |
| `doc_id`            | string | -                 | Google Doc ID to extract feedback from (required)                            |
| `credentials`       | string | `bau-credentials.json`                 | Path to service account JSON (required)                                      |
| `github_repo`       | string | `<owner>/<repo>`                | GitHub repository in format "owner/repo" or HTTPS URL (required)             |
| `chunk_size`        | int    | `1`               | Total number of chunks to create (default: 1, or 5 if page_refresh is true)  |
| `dry_run`           | bool   | `false`           | Run extraction and planning only; skip Copilot execution and PR creation     |
| `output_dir`        | string | `bauer-output`    | Output directory for generated files                                         |
| `model`             | string | `gpt-5-mini-high` | Copilot model to use for code generation                                     |
| `summary_model`     | string | `gpt-5-mini-high` | Copilot model to use for summary generation                                  |
| `page_refresh`      | bool   | `false`           | Whether this is a page refresh, or the default copy update                   |
| `branch_prefix`     | string | `bauer`           | Branch naming prefix                                                         |
| `local_repo_path`   | string | `/tmp/<repo>` | Local path where repository will be cloned                                   |

#### Example config.json

```json
{
  "doc_id": "XXX",
  "credentials": "bau-credentials.json",
  "github_repo": "canonical/ubuntu.com",
  "chunk_size": 1,
  "dry_run": false,
  "output_dir": "bauer-output",
  "model": "gpt-5-mini-high",
  "summary_model": "gpt-5-mini-high",
  "page_refresh": false,
  "branch_prefix": "bauer",
  "local_repo_path": "/tmp/ubuntu.com"
}
```


## API usage (on-going development)

The API server exposes a small HTTP surface for submitting jobs and checking health. Jobs run asynchronously and write outputs to `base-output-dir/<request-id>`.

### Run the API server

From Repository root:

```bash
task build-api
./bauer-api --config config.json
```

### Endpoints

#### POST /api/v1/job

Submit a job for a Google Doc.

Request body:

```json
{
  "doc_id": "<google-doc-id>",
  "chunk_size": 1,
  "page_refresh": false
}
```

Notes:

- `chunk_size` defaults to 1 if omitted.
- When `page_refresh` is true, the default chunk size becomes 5.

Responses:

- `202 Accepted` with body `{"code":202}` when the job is accepted.
- `400 Bad Request` for invalid JSON.

Example:

```bash
curl -X POST http://localhost:8090/api/v1/job \
        -H 'Content-Type: application/json' \
        -d '{"doc_id":"<google-doc-id>","chunk_size":2,"page_refresh":false}'
```

#### GET /api/v1/health

Simple health check.

Example:

```bash
curl http://localhost:8090/api/v1/health
```

## Local development

### Prerequisites

1. Install [`go`](https://golang.org/dl/)
2. Install [`task`](https://taskfile.dev/docs/installation)
3. Install [Copilot CLI](https://docs.github.com/en/copilot/how-tos/set-up/install-copilot-cli)

## Steps

1. Modify the [Taskfile](./Taskfile.yml) with your document ID and credentials path for convenience
2. Run the project with task

```
task run
```

## Documentation

For more information refer to [`ARCHITECTURE.md`](/docs/ARCHITECTURE.md)

## Future improvements

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
