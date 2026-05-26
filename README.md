# Bauer

> Info
> Start with [docs/specs](./docs/specs/). The current README may be out of date while the v2 and v2.1 work is in progress.

A proof-of-concept Go application that extracts document content, suggestions (proposed edits), and comments from Google Docs using the Google Docs API and Google Drive API.

## Local development

### Prerequisites

1. Install [`go`](https://golang.org/dl/)
2. Install [`task`](https://taskfile.dev/docs/installation)

### Steps
1. Build the project

```
task build
```
2. Modify the [Taskfile](./Taskfile.yml) with your document ID and credentials path for convenience
3. Run the project

```
./bauer --doc-id <doc-id> \
  --credentials <path-to-creds> \
  --parse-only
```


## Configuration

1. Create `credentials.json` file and copy the structure from the [default file](https://github.com/muhammadbassiony/Bauer/blob/main/credentials.json)
2. Get credentials from Google Cloud service or Bitwarden (internally)
3. Fill up `credentials.json` with Google Cloud credentials (see [Generating Google Cloud credentials](https://developers.google.com/workspace/guides/create-credentials)).
4. Share copy document with service account

## Usage

1.  Build Bauer locally using the Local development steps above (`task build`)
2. If running with GitHub integration (no `--parse-only`), ensure `copilot` is installed and authenticated
3. Get document ID from Google Document & share the document with the service account
4. Run Bauer

```bash
bauer --doc-id <your-document-id> --credentials ./credentials.json
```

5. Optional parameters

| Flag               | Type   | Default              | Description                                                                     | Requires Copilot |
| ------------------ | ------ | -------------------- | ------------------------------------------------------------------------------- | ---------------- |
| `--github-repo`    | string | (required if not parse-only) | GitHub repository (owner/repo or HTTPS URL)                              | Yes*             |
| `--credentials`    | string | `bau-test-creds.json` | Path to service account credentials JSON                                       | No               |
| `--local-repo-path` | string | `/tmp/ubuntu.com`    | Local path for cloned repository                                               | No               |
| `--dry-run`        | bool   | `false`              | Perform a dry run without creating PR                                          | Yes*             |
| `--output-dir`     | string | `bauer-output`       | Output directory for Bauer results                                             | No               |
| `--branch-prefix`  | string | `bauer`              | Branch naming prefix                                                            | No               |
| `--parse-only`     | bool   | `false`              | Parse document and output machine-readable JSON (skip GitHub integration) | No               |

*These flags require Copilot integration to be configured when performing GitHub operations (not needed for `--parse-only`)
<!-- ### Examples

#### Basic run

```bash
bauer --doc-id <your-document-id> --credentials ./credentials.json
```

#### Dry run (test without executing changes)

```bash
bauer --doc-id <your-document-id> \
        --credentials ./credentials.json \
        --dry-run
```

#### Custom chunk size and output directory

```bash
bauer --doc-id <your-document-id> \
        --credentials ./credentials.json \
        --chunk-size 5 \
        --output-dir ./results
```

#### Specify model

```bash
bauer --doc-id <your-document-id> \
        --credentials ./credentials.json \
        --model "claude-sonnet-4.5"
```

#### Run on a different repository
```bash
bauer --doc-id <your-document-id> \
        --credentials ./credentials.json \
        --target-repo ../my-other-repo
```

### Page refresh

```bash
bauer --doc-id <your-document-id> \
        --credentials ./credentials.json \
        --page-refresh
``` -->

## API usage

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


## Documentation

For more information refer to [`ARCHITECTURE.md`](/docs/ARCHITECTURE.md)

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